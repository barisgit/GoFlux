package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types"
	"regexp"
	"sort"
	"strings"
	"time"

	"goflux/internal/typegen/types"

	"golang.org/x/tools/go/packages"
)

var (
	mapRegex          *regexp.Regexp
	keyPackageIndex   int
	keyTypeIndex      int
	valueArrayIndex   int
	valuePackageIndex int
	valueTypeIndex    int
)

func init() {
	// Regex to parse complex Go types like map[string][]User, *pkg.Type, etc.
	mapRegex = regexp.MustCompile(`(?:map\[(?:(?P<keyPackage>\w+)\.)?(?P<keyType>\w+)])?(?P<valueArray>\[])?(?:\*?(?P<valuePackage>\w+)\.)?(?P<valueType>.+)`)
	keyPackageIndex = mapRegex.SubexpIndex("keyPackage")
	keyTypeIndex = mapRegex.SubexpIndex("keyType")
	valueArrayIndex = mapRegex.SubexpIndex("valueArray")
	valuePackageIndex = mapRegex.SubexpIndex("valuePackage")
	valueTypeIndex = mapRegex.SubexpIndex("valueType")
}

// AnalyzeProject performs comprehensive analysis of a Go project
func AnalyzeProject(projectPath string, debug bool) (*types.APIAnalysis, error) {
	start := time.Now()

	// Load all packages with full type information
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedName,
		Dir:  projectPath,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	// Perform deep analysis of API patterns
	analysis := analyzeAPIPatterns(pkgs)

	elapsed := time.Since(start)

	if debug {
		fmt.Printf("Routes discovered:\n")
		for _, route := range analysis.Routes {
			fmt.Printf("   %s %s -> %s (req: %s, res: %s)\n",
				route.Method, route.Path, route.Handler, route.RequestType, route.ResponseType)
		}

		fmt.Printf("Types discovered:\n")
		for _, t := range analysis.TypeDefs {
			fmt.Printf("   %s (%d fields)\n", t.Name, len(t.Fields))
		}
	} else {
		fmt.Printf("GoFlux Type Generator: Discovered %d routes and %d types in %v\n",
			len(analysis.Routes), len(analysis.TypeDefs), elapsed)
	}

	return analysis, nil
}

func analyzeAPIPatterns(pkgs []*packages.Package) *types.APIAnalysis {
	analysis := &types.APIAnalysis{
		Routes:           []types.APIRoute{},
		UsedTypes:        make(map[string]*gotypes.Named),
		HandlerFuncs:     make(map[string]*ast.FuncDecl),
		ImportNamespaces: make(map[string]bool),
		EnumTypes:        make(map[string]types.TypeDefinition),
	}

	// Track router groups and their paths
	routerGroups := make(map[string]string)
	// Track function calls with router parameters
	functionRouterContext := make(map[string]string)

	// Step 1: Find all function declarations and route registrations
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			// First pass: find router group assignments and function calls
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					if node.Name != nil {
						analysis.HandlerFuncs[node.Name.Name] = node
					}
				case *ast.AssignStmt:
					// Look for router group assignments like: users := api.Group("/users")
					if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
						if ident, ok := node.Lhs[0].(*ast.Ident); ok {
							if call, ok := node.Rhs[0].(*ast.CallExpr); ok {
								if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Group" {
									if len(call.Args) > 0 {
										if pathLit, ok := call.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
											groupPath := strings.Trim(pathLit.Value, `"`)
											routerGroups[ident.Name] = groupPath
										}
									}
								}
							}
						}
					}
				case *ast.CallExpr:
					// Look for function calls like: setupUserRoutes(users, database)
					if ident, ok := node.Fun.(*ast.Ident); ok {
						funcName := ident.Name
						if strings.HasPrefix(funcName, "setup") && strings.HasSuffix(funcName, "Routes") {
							if len(node.Args) >= 1 {
								if routerArg, ok := node.Args[0].(*ast.Ident); ok {
									if groupPath, exists := routerGroups[routerArg.Name]; exists {
										functionRouterContext[funcName] = groupPath
									}
								}
							}
						}
					}
				}
				return true
			})

			// Second pass: find route registrations with function context
			for _, decl := range file.Decls {
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					funcName := ""
					if funcDecl.Name != nil {
						funcName = funcDecl.Name.Name
					}

					// Check if this function has a known router context
					var currentGroupPath string
					if groupPath, exists := functionRouterContext[funcName]; exists {
						currentGroupPath = groupPath
					}

					// Look for route calls within this function
					ast.Inspect(funcDecl, func(n ast.Node) bool {
						if call, ok := n.(*ast.CallExpr); ok {
							if route := extractRouteFromCallWithContext(call, pkg.TypesInfo, routerGroups, currentGroupPath); route != nil {
								analysis.Routes = append(analysis.Routes, *route)
							}
						}
						return true
					})
				}
			}
		}
	}

	// Step 2: Analyze each route's handler function to discover used types
	for i, route := range analysis.Routes {
		handlerName := extractHandlerName(route.Handler)
		if handlerFunc, exists := analysis.HandlerFuncs[handlerName]; exists {
			analyzeHandlerFunction(handlerFunc, &analysis.Routes[i], analysis.UsedTypes, pkgs)
		}
	}

	// Step 3: Generate type definitions for all discovered types
	analysis.TypeDefs = generateTypeDefinitions(analysis.UsedTypes, pkgs)

	return analysis
}

func extractRouteFromCallWithContext(call *ast.CallExpr, info *gotypes.Info, routerGroups map[string]string, functionGroupPath string) *types.APIRoute {
	// Look for: router.Method("/path", handlerFunc)
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		method := sel.Sel.Name

		if !isHTTPMethod(method) {
			return nil
		}

		if len(call.Args) < 2 {
			return nil
		}

		// Extract path
		pathLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || pathLit.Kind != token.STRING {
			return nil
		}

		path := strings.Trim(pathLit.Value, `"`)

		// Skip Swagger documentation routes
		if strings.Contains(path, "/docs") || strings.Contains(path, "swagger") {
			return nil
		}

		// Determine the router being used and get its group path
		var groupPath string
		if x, ok := sel.X.(*ast.Ident); ok {
			if prefix, exists := routerGroups[x.Name]; exists {
				// Direct router group usage
				groupPath = prefix
			} else if x.Name == "router" && functionGroupPath != "" {
				// Router parameter in function with known context
				groupPath = functionGroupPath
			}
		}

		// Combine group path with route path
		fullPath := groupPath + path
		if !strings.HasPrefix(fullPath, "/") {
			fullPath = "/" + fullPath
		}

		// Extract handler function name
		var handler string
		switch h := call.Args[1].(type) {
		case *ast.SelectorExpr:
			if x, ok := h.X.(*ast.Ident); ok {
				handler = x.Name + "." + h.Sel.Name
			}
		case *ast.Ident:
			handler = h.Name
		default:
			return nil
		}

		return &types.APIRoute{
			Method:      strings.ToUpper(method),
			Path:        "/api" + fullPath,
			Handler:     handler,
			Description: generateDescription(method, fullPath),
		}
	}

	return nil
}

func analyzeHandlerFunction(fn *ast.FuncDecl, route *types.APIRoute, usedTypes map[string]*gotypes.Named, pkgs []*packages.Package) {
	// Find the package containing this function
	var pkg *packages.Package
	for _, p := range pkgs {
		for _, file := range p.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				if n == fn {
					pkg = p
					return false
				}
				return true
			})
			if pkg != nil {
				break
			}
		}
		if pkg != nil {
			break
		}
	}

	if pkg == nil {
		return
	}

	// Initialize type discovery for this handler
	td := types.NewTypeDiscovery()
	analysis := &types.APIAnalysis{
		UsedTypes:        usedTypes,
		ImportNamespaces: make(map[string]bool),
	}

	// Analyze function body to find request and response types
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			// Look for c.BodyParser(&variable) calls to find request types
			case *ast.CallExpr:
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "BodyParser" && len(node.Args) > 0 {
						// Extract the type from &variable
						if unary, ok := node.Args[0].(*ast.UnaryExpr); ok && unary.Op == token.AND {
							if ident, ok := unary.X.(*ast.Ident); ok {
								if varType := pkg.TypesInfo.TypeOf(ident); varType != nil {
									// Use enhanced type discovery
									td.DiscoverTypesRecursively(varType, analysis, pkgs)

									if named := extractNamedType(varType); named != nil {
										typeName := named.Obj().Name()
										if !isFiberType(typeName) {
											// Set request type for different HTTP methods
											if route.Method == "POST" {
												route.RequestType = fmt.Sprintf("Omit<%s, 'id'>", typeName)
											} else if route.Method == "PUT" || route.Method == "PATCH" {
												route.RequestType = fmt.Sprintf("Partial<%s>", typeName)
											} else {
												route.RequestType = typeName
											}
										}
									}
								}
							}
						}
					} else if sel.Sel.Name == "JSON" && len(node.Args) > 0 {
						// Analyze c.JSON() calls for response types with enhanced discovery
						if argType := pkg.TypesInfo.TypeOf(node.Args[0]); argType != nil {
							td.DiscoverTypesRecursively(argType, analysis, pkgs)

							// Handle slice returns first (like users, posts)
							if slice, ok := argType.(*gotypes.Slice); ok {
								if named := extractNamedType(slice.Elem()); named != nil {
									typeName := named.Obj().Name()
									if !isFiberType(typeName) {
										route.ResponseType = typeName + "[]"
									}
								}
							} else if named := extractNamedType(argType); named != nil {
								// Handle direct struct returns
								typeName := named.Obj().Name()
								if !isFiberType(typeName) {
									route.ResponseType = typeName
								}
							}
						}
					}
				}
			}
			return true
		})
	}

	// If no response type found, check if it's a single item GET by ID
	if route.ResponseType == "" && route.Method == "GET" && strings.Contains(route.Path, "/:id") {
		// Infer from route path (e.g., /users/:id -> User)
		pathParts := strings.Split(route.Path, "/")
		for _, part := range pathParts {
			if !strings.Contains(part, ":") && part != "" && part != "api" {
				resourceName := capitalize(singularize(part))
				// Check if this type exists in usedTypes or find it
				for typeName := range usedTypes {
					if typeName == resourceName {
						route.ResponseType = typeName
						return
					}
				}
				// If not found, try to find it in packages and discover its dependencies
				for _, p := range pkgs {
					if obj := p.Types.Scope().Lookup(resourceName); obj != nil {
						if named, ok := obj.Type().(*gotypes.Named); ok {
							td.DiscoverTypesRecursively(named, analysis, pkgs)
							route.ResponseType = resourceName
							return
						}
					}
				}
			}
		}
	}
}

func generateTypeDefinitions(usedTypes map[string]*gotypes.Named, pkgs []*packages.Package) []types.TypeDefinition {
	var typeDefs []types.TypeDefinition

	for typeName, namedType := range usedTypes {
		if structType, ok := namedType.Underlying().(*gotypes.Struct); ok {
			// Find the AST node for this type to get field tags
			var astStruct *ast.StructType
			var packageName string

			if namedType.Obj().Pkg() != nil {
				packageName = getPackageName(namedType.Obj().Pkg().Path())
			}

			for _, pkg := range pkgs {
				for _, file := range pkg.Syntax {
					ast.Inspect(file, func(n ast.Node) bool {
						if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == typeName {
							if st, ok := ts.Type.(*ast.StructType); ok {
								astStruct = st
								return false
							}
						}
						return true
					})
					if astStruct != nil {
						break
					}
				}
				if astStruct != nil {
					break
				}
			}

			typeDef := analyzeStructType(typeName, structType, astStruct, packageName)
			typeDefs = append(typeDefs, typeDef)
		}
	}

	// Sort type definitions for consistent output
	sort.Slice(typeDefs, func(i, j int) bool {
		return typeDefs[i].Name < typeDefs[j].Name
	})

	return typeDefs
}

func analyzeStructType(name string, structType *gotypes.Struct, astStruct *ast.StructType, packageName string) types.TypeDefinition {
	def := types.TypeDefinition{
		Name:        name,
		Fields:      []types.FieldInfo{},
		PackageName: packageName,
		IsEnum:      false,
	}

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)

		if !field.Exported() {
			continue
		}

		var jsonTag string
		if astStruct != nil && i < len(astStruct.Fields.List) {
			if astStruct.Fields.List[i].Tag != nil {
				tag := strings.Trim(astStruct.Fields.List[i].Tag.Value, "`")
				jsonTag = extractJSONTag(tag)
			}
		}

		if jsonTag == "-" {
			continue
		}

		// Use enhanced type conversion
		fieldTypeName := goTypeToTypeScriptType(field.Type().String())

		fieldInfo := types.FieldInfo{
			Name:     field.Name(),
			TypeName: fieldTypeName,
			JSONTag:  jsonTag,
			Optional: isPointerType(field.Type()),
			IsArray:  isSliceType(field.Type()),
		}

		def.Fields = append(def.Fields, fieldInfo)
	}

	return def
}

func extractJSONTag(tag string) string {
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "json:") {
			jsonPart := strings.Trim(part[5:], "\"")
			parts := strings.Split(jsonPart, ",")
			if len(parts) > 0 && parts[0] != "" {
				return parts[0]
			}
		}
	}
	return ""
}

func goTypeToTypeScriptType(input string) string {
	// Enhanced type conversion logic from the original
	// Handle array types with fully qualified names
	if strings.HasPrefix(input, "[]") && strings.Contains(input, "/") {
		elementType := strings.TrimPrefix(input, "[]")
		parts := strings.Split(elementType, ".")
		if len(parts) > 0 {
			typeName := parts[len(parts)-1]
			if isCustomType(typeName) {
				return "Array<" + typeName + ">"
			}
		}
	}

	// Handle fully qualified type names
	if strings.Contains(input, "/") && !strings.HasPrefix(input, "map[") && !strings.HasPrefix(input, "[]") {
		parts := strings.Split(input, ".")
		if len(parts) > 0 {
			typeName := parts[len(parts)-1]
			if isCustomType(typeName) {
				return typeName
			}
		}
	}

	// Basic type mappings
	switch {
	case input == "interface{}" || input == "interface {}":
		return "any"
	case input == "string":
		return "string"
	case input == "error":
		return "Error"
	case strings.HasPrefix(input, "int") ||
		strings.HasPrefix(input, "uint") ||
		strings.HasPrefix(input, "float"):
		return "number"
	case input == "bool":
		return "boolean"
	case strings.HasPrefix(input, "time.Time"):
		return "string"
	case strings.HasPrefix(input, "*"):
		return goTypeToTypeScriptType(strings.TrimPrefix(input, "*"))
	default:
		if isCustomType(input) {
			return input
		}
		return "any"
	}
}

func isCustomType(typeName string) bool {
	if len(typeName) == 0 {
		return false
	}

	// Check if first character is uppercase (Go convention for exported types)
	firstChar := rune(typeName[0])
	if firstChar < 'A' || firstChar > 'Z' {
		return false
	}

	// Exclude known non-custom types that might be capitalized
	excludedTypes := []string{"Time", "Duration", "Context", "Error"}
	for _, excluded := range excludedTypes {
		if typeName == excluded {
			return false
		}
	}

	return true
}

func isPointerType(t gotypes.Type) bool {
	_, isPtr := t.(*gotypes.Pointer)
	return isPtr
}

func isSliceType(t gotypes.Type) bool {
	_, isSlice := t.(*gotypes.Slice)
	return isSlice
}

func extractHandlerName(handler string) string {
	parts := strings.Split(handler, ".")
	return parts[len(parts)-1]
}

func isHTTPMethod(method string) bool {
	httpMethods := []string{"Get", "Post", "Put", "Delete", "Patch", "Options", "Head"}
	for _, m := range httpMethods {
		if method == m {
			return true
		}
	}
	return false
}

func generateDescription(method, path string) string {
	action := "Execute"
	if method == "GET" {
		if strings.Contains(path, ":id") {
			action = "Get"
		} else {
			action = "List"
		}
	} else if method == "POST" {
		action = "Create"
	} else if method == "PUT" || method == "PATCH" {
		action = "Update"
	} else if method == "DELETE" {
		action = "Delete"
	}

	// Extract resource name from path, handling nested routes
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	var resourceParts []string

	for _, part := range pathParts {
		if !strings.Contains(part, ":") && part != "" {
			resourceParts = append(resourceParts, part)
		}
	}

	resource := strings.Join(resourceParts, " ")
	if resource == "" {
		resource = "resource"
	}

	return fmt.Sprintf("%s %s", action, resource)
}

// Helper functions
func isFiberType(typeName string) bool {
	fiberTypes := []string{"Ctx", "Map", "Config", "Error"}
	for _, ft := range fiberTypes {
		if typeName == ft {
			return true
		}
	}
	return false
}

func extractNamedType(t gotypes.Type) *gotypes.Named {
	switch typ := t.(type) {
	case *gotypes.Named:
		return typ
	case *gotypes.Pointer:
		return extractNamedType(typ.Elem())
	case *gotypes.Slice:
		return extractNamedType(typ.Elem())
	default:
		return nil
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func singularize(s string) string {
	if strings.HasSuffix(s, "s") && len(s) > 1 {
		return s[:len(s)-1]
	}
	return s
}

func getPackageName(packagePath string) string {
	if packagePath == "" {
		return ""
	}
	parts := strings.Split(packagePath, "/")
	return parts[len(parts)-1]
}
