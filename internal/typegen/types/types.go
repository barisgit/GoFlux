package types

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// APIRoute represents a discovered API route
type APIRoute struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	Handler      string `json:"handler"`
	RequestType  string `json:"requestType"`
	ResponseType string `json:"responseType"`
	Description  string `json:"description"`
}

// TypeDefinition represents a Go struct converted to TypeScript
type TypeDefinition struct {
	Name        string      `json:"name"`
	Fields      []FieldInfo `json:"fields"`
	PackageName string      `json:"packageName"`
	IsEnum      bool        `json:"isEnum"`
	EnumValues  []string    `json:"enumValues,omitempty"`
}

// FieldInfo represents a field in a struct
type FieldInfo struct {
	Name     string `json:"name"`
	TypeName string `json:"type"`
	JSONTag  string `json:"jsonTag"`
	Optional bool   `json:"optional"`
	IsArray  bool   `json:"isArray"`
}

// APIAnalysis contains the complete analysis results
type APIAnalysis struct {
	Routes           []APIRoute
	UsedTypes        map[string]*types.Named
	TypeDefs         []TypeDefinition
	HandlerFuncs     map[string]*ast.FuncDecl
	ImportNamespaces map[string]bool
	EnumTypes        map[string]TypeDefinition
}

// TypeDiscovery handles recursive type discovery
type TypeDiscovery struct {
	SeenTypes        map[string]bool
	PackageTypes     map[string]map[string]*types.Named
	ImportNamespaces map[string]bool
}

// APIMethod represents a generated API method
type APIMethod struct {
	Route       APIRoute
	MethodName  string
	HasIDParam  bool
	HasBodyData bool
}

// NestedAPI represents the nested API structure
type NestedAPI map[string]interface{} // Can contain either another NestedAPI or APIMethod

// NewTypeDiscovery creates a new TypeDiscovery instance
func NewTypeDiscovery() *TypeDiscovery {
	return &TypeDiscovery{
		SeenTypes:        make(map[string]bool),
		PackageTypes:     make(map[string]map[string]*types.Named),
		ImportNamespaces: make(map[string]bool),
	}
}

// DiscoverTypesRecursively performs enhanced recursive type discovery
func (td *TypeDiscovery) DiscoverTypesRecursively(t types.Type, analysis *APIAnalysis, pkgs []*packages.Package) {
	switch typ := t.(type) {
	case *types.Named:
		td.processNamedType(typ, analysis, pkgs)
	case *types.Pointer:
		td.DiscoverTypesRecursively(typ.Elem(), analysis, pkgs)
	case *types.Slice:
		td.DiscoverTypesRecursively(typ.Elem(), analysis, pkgs)
	case *types.Map:
		td.DiscoverTypesRecursively(typ.Key(), analysis, pkgs)
		td.DiscoverTypesRecursively(typ.Elem(), analysis, pkgs)
	case *types.Struct:
		td.processStructType(typ, analysis, pkgs)
	}
}

func (td *TypeDiscovery) processNamedType(named *types.Named, analysis *APIAnalysis, pkgs []*packages.Package) {
	typeName := named.Obj().Name()
	packagePath := ""
	if named.Obj().Pkg() != nil {
		packagePath = named.Obj().Pkg().Path()
	}

	fullName := packagePath + "." + typeName
	if td.SeenTypes[fullName] {
		return
	}
	td.SeenTypes[fullName] = true

	// Skip built-in and standard library types we don't want to generate
	if packagePath == "" || strings.HasPrefix(packagePath, "time") ||
		strings.HasPrefix(packagePath, "context") || isFiberType(typeName) {
		return
	}

	analysis.UsedTypes[typeName] = named

	// Process underlying type if it's a struct
	if structType, ok := named.Underlying().(*types.Struct); ok {
		td.processStructFields(structType, analysis, pkgs)
	}
}

func (td *TypeDiscovery) processStructType(structType *types.Struct, analysis *APIAnalysis, pkgs []*packages.Package) {
	td.processStructFields(structType, analysis, pkgs)
}

func (td *TypeDiscovery) processStructFields(structType *types.Struct, analysis *APIAnalysis, pkgs []*packages.Package) {
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if !field.Exported() {
			continue
		}
		td.DiscoverTypesRecursively(field.Type(), analysis, pkgs)
	}
}

// Helper function to check if a type is from fiber package
func isFiberType(typeName string) bool {
	fiberTypes := []string{"Ctx", "Map", "Config", "Error"}
	for _, ft := range fiberTypes {
		if typeName == ft {
			return true
		}
	}
	return false
}
