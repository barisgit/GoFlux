package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barisgit/goflux/config"
	"github.com/barisgit/goflux/cli/internal/typegen/types"
)

// generateBasicJSNestedObject generates JavaScript API methods with destructured parameters
func generateBasicJSNestedObject(content *strings.Builder, nested types.NestedAPI, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Sort keys for consistent output
	keys := make([]string, 0, len(nested))
	for key := range nested {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		value := nested[key]

		// Quote key if it contains hyphens or other special characters
		quotedKey := key
		if strings.Contains(key, "-") || strings.Contains(key, " ") || !isValidJSIdentifier(key) {
			quotedKey = fmt.Sprintf(`"%s"`, key)
		}

		switch v := value.(type) {
		case types.APIMethod:
			// Generate the method code first to extract docstring
			templateData := createMethodTemplateDataJS(v)
			methodCode, err := executeMethodTemplate(basicMethodTemplate, templateData)

			if err != nil {
				// Fallback to original approach without docstring extraction
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))
				panic(err) // TODO: fix this
			} else {
				// Extract docstring and method body separately
				lines := strings.Split(methodCode, "\n")
				var docstringLines []string
				var methodBodyLines []string
				inDocstring := false

				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "/**") {
						inDocstring = true
						docstringLines = append(docstringLines, line)
					} else if inDocstring && strings.HasSuffix(trimmedLine, "*/") {
						inDocstring = false
						docstringLines = append(docstringLines, line)
					} else if inDocstring {
						docstringLines = append(docstringLines, line)
					} else {
						methodBodyLines = append(methodBodyLines, line)
					}
				}

				// Write docstring first (with proper indentation)
				if len(docstringLines) > 0 {
					for _, line := range docstringLines {
						if line != "" {
							content.WriteString(indentStr)
						}
						content.WriteString(line)
						content.WriteString("\n")
					}
				}

				// Write property name and method body
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))

				// Write method body (skip empty first line if it exists)
				for j, line := range methodBodyLines {
					if j == 0 && strings.TrimSpace(line) == "" {
						continue
					}
					if j > 0 && line != "" {
						content.WriteString(indentStr)
					}
					content.WriteString(line)
					if j < len(methodBodyLines)-1 {
						content.WriteString("\n")
					}
				}
			}
		case types.NestedAPI:
			// Generate nested object
			content.WriteString(fmt.Sprintf("%s%s: {\n", indentStr, quotedKey))
			generateBasicJSNestedObject(content, v, indent+1)
			content.WriteString(fmt.Sprintf("%s}", indentStr))
		}

		if i < len(keys)-1 {
			content.WriteString(",")
		}
		content.WriteString("\n")
	}
}

// generateBasicTSNestedObject generates TypeScript API methods with proper TypeScript types
func generateBasicTSNestedObject(content *strings.Builder, nested types.NestedAPI, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Sort keys for consistent output
	keys := make([]string, 0, len(nested))
	for key := range nested {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		value := nested[key]

		// Quote key if it contains hyphens or other special characters
		quotedKey := key
		if strings.Contains(key, "-") || strings.Contains(key, " ") || !isValidJSIdentifier(key) {
			quotedKey = fmt.Sprintf(`"%s"`, key)
		}

		switch v := value.(type) {
		case types.APIMethod:
			// Generate the method code first to extract docstring
			templateData := createMethodTemplateData(v, "basic-ts", false)
			methodCode, err := executeMethodTemplate(basicTSMethodTemplate, templateData)

			if err != nil {
				// Fallback to original approach without docstring extraction
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))
				panic(err) // TODO: fix this
			} else {
				// Extract docstring and method body separately
				lines := strings.Split(methodCode, "\n")
				var docstringLines []string
				var methodBodyLines []string
				inDocstring := false

				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "/**") {
						inDocstring = true
						docstringLines = append(docstringLines, line)
					} else if inDocstring && strings.HasSuffix(trimmedLine, "*/") {
						inDocstring = false
						docstringLines = append(docstringLines, line)
					} else if inDocstring {
						docstringLines = append(docstringLines, line)
					} else {
						methodBodyLines = append(methodBodyLines, line)
					}
				}

				// Write docstring first (with proper indentation)
				if len(docstringLines) > 0 {
					for _, line := range docstringLines {
						if line != "" {
							content.WriteString(indentStr)
						}
						content.WriteString(line)
						content.WriteString("\n")
					}
				}

				// Write property name and method body
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))

				// Write method body (skip empty first line if it exists)
				for j, line := range methodBodyLines {
					if j == 0 && strings.TrimSpace(line) == "" {
						continue
					}
					if j > 0 && line != "" {
						content.WriteString(indentStr)
					}
					content.WriteString(line)
					if j < len(methodBodyLines)-1 {
						content.WriteString("\n")
					}
				}
			}
		case types.NestedAPI:
			// Generate nested object
			content.WriteString(fmt.Sprintf("%s%s: {\n", indentStr, quotedKey))
			generateBasicTSNestedObject(content, v, indent+1)
			content.WriteString(fmt.Sprintf("%s}", indentStr))
		}

		if i < len(keys)-1 {
			content.WriteString(",")
		}
		content.WriteString("\n")
	}
}

// generateAxiosNestedObject generates Axios-style API methods
func generateAxiosNestedObject(content *strings.Builder, nested types.NestedAPI, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Sort keys for consistent output
	keys := make([]string, 0, len(nested))
	for key := range nested {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		value := nested[key]

		quotedKey := key
		if strings.Contains(key, "-") || strings.Contains(key, " ") || !isValidJSIdentifier(key) {
			quotedKey = fmt.Sprintf(`"%s"`, key)
		}

		switch v := value.(type) {
		case types.APIMethod:
			// Generate the method code first to extract docstring
			templateData := createMethodTemplateData(v, "axios", false)
			methodCode, err := executeMethodTemplate(axiosMethodTemplate, templateData)

			if err != nil {
				// Fallback to original approach without docstring extraction
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))
				panic(err) // TODO: fix this
			} else {
				// Extract docstring and method body separately
				lines := strings.Split(methodCode, "\n")
				var docstringLines []string
				var methodBodyLines []string
				inDocstring := false

				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "/**") {
						inDocstring = true
						docstringLines = append(docstringLines, line)
					} else if inDocstring && strings.HasSuffix(trimmedLine, "*/") {
						inDocstring = false
						docstringLines = append(docstringLines, line)
					} else if inDocstring {
						docstringLines = append(docstringLines, line)
					} else {
						methodBodyLines = append(methodBodyLines, line)
					}
				}

				// Write docstring first (with proper indentation)
				if len(docstringLines) > 0 {
					for _, line := range docstringLines {
						if line != "" {
							content.WriteString(indentStr)
						}
						content.WriteString(line)
						content.WriteString("\n")
					}
				}

				// Write property name and method body
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))

				// Write method body (skip empty first line if it exists)
				for j, line := range methodBodyLines {
					if j == 0 && strings.TrimSpace(line) == "" {
						continue
					}
					if j > 0 && line != "" {
						content.WriteString(indentStr)
					}
					content.WriteString(line)
					if j < len(methodBodyLines)-1 {
						content.WriteString("\n")
					}
				}
			}
		case types.NestedAPI:
			content.WriteString(fmt.Sprintf("%s%s: {\n", indentStr, quotedKey))
			generateAxiosNestedObject(content, v, indent+1)
			content.WriteString(fmt.Sprintf("%s}", indentStr))
		}

		if i < len(keys)-1 {
			content.WriteString(",")
		}
		content.WriteString("\n")
	}
}

// generateTRPCNestedObject generates tRPC-like nested structure with React Query hooks
func generateTRPCNestedObject(content *strings.Builder, nested types.NestedAPI, config *config.APIClientConfig, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Sort keys for consistent output
	keys := make([]string, 0, len(nested))
	for key := range nested {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		value := nested[key]

		quotedKey := key
		if strings.Contains(key, "-") || strings.Contains(key, " ") || !isValidJSIdentifier(key) {
			quotedKey = fmt.Sprintf(`"%s"`, key)
		}

		switch v := value.(type) {
		case types.APIMethod:
			// Generate the method code first to extract docstring
			var methodCode string
			var err error

			templateData := createMethodTemplateData(v, "trpc-like", config.ReactQuery.Enabled)

			if v.Route.Method == "GET" {
				methodCode, err = executeMethodTemplate(trpcGetMethodTemplate, templateData)
			} else {
				methodCode, err = executeMethodTemplate(trpcMutationMethodTemplate, templateData)
			}

			if err != nil {
				// Fallback to original approach without docstring extraction
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))
				panic(err) // TODO: fix this
			} else {
				// Extract docstring and method body separately
				lines := strings.Split(methodCode, "\n")
				var docstringLines []string
				var methodBodyLines []string
				inDocstring := false

				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "/**") {
						inDocstring = true
						docstringLines = append(docstringLines, line)
					} else if inDocstring && strings.HasSuffix(trimmedLine, "*/") {
						inDocstring = false
						docstringLines = append(docstringLines, line)
					} else if inDocstring {
						docstringLines = append(docstringLines, line)
					} else {
						methodBodyLines = append(methodBodyLines, line)
					}
				}

				// Write docstring first (with proper indentation)
				if len(docstringLines) > 0 {
					for _, line := range docstringLines {
						if line != "" {
							content.WriteString(indentStr)
						}
						content.WriteString(line)
						content.WriteString("\n")
					}
				}

				// Write property name and method body
				content.WriteString(fmt.Sprintf("%s%s: ", indentStr, quotedKey))

				// Write method body (skip empty first line if it exists)
				for j, line := range methodBodyLines {
					if j == 0 && strings.TrimSpace(line) == "" {
						continue
					}
					if j > 0 && line != "" {
						content.WriteString(indentStr)
					}
					content.WriteString(line)
					if j < len(methodBodyLines)-1 {
						content.WriteString("\n")
					}
				}
			}
		case types.NestedAPI:
			content.WriteString(fmt.Sprintf("%s%s: {\n", indentStr, quotedKey))
			generateTRPCNestedObject(content, v, config, indent+1)
			content.WriteString(fmt.Sprintf("%s}", indentStr))
		}

		if i < len(keys)-1 {
			content.WriteString(",")
		}
		content.WriteString("\n")
	}
}
