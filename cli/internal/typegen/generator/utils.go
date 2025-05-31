package generator

import (
	"regexp"
	"strings"
)

// String manipulation utilities

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r - 'A' + 'a')
	}
	return result.String()
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

func isValidJSIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// Check if the first character is valid (letter, underscore, or dollar sign)
	firstChar := rune(s[0])
	if !((firstChar >= 'a' && firstChar <= 'z') ||
		(firstChar >= 'A' && firstChar <= 'Z') ||
		firstChar == '_' || firstChar == '$') {
		return false
	}

	// Check if remaining characters are valid (letters, digits, underscores, or dollar signs)
	for _, char := range s[1:] {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '$') {
			return false
		}
	}

	return true
}

// extractCleanDescription extracts only the main description, removing parameter information
func extractCleanDescription(description string) string {
	if description == "" {
		return ""
	}

	// Split by double newlines to separate sections
	sections := strings.Split(description, "\n\n")

	// Return only the first section (main description)
	if len(sections) > 0 {
		return strings.TrimSpace(sections[0])
	}

	return strings.TrimSpace(description)
}

// Helper function for slice contains check
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// buildRequestPath builds a request path with parameter substitution
func buildRequestPath(path string, hasIDParam bool) string {
	requestPath := path
	// Remove "/api" prefix if it exists
	if strings.HasPrefix(requestPath, "/api") {
		requestPath = requestPath[4:]
	}

	if hasIDParam {
		re1 := regexp.MustCompile(`/:[^/]+`)
		requestPath = re1.ReplaceAllString(requestPath, "/$${id}")
		re2 := regexp.MustCompile(`/\{[^}]+\}`)
		requestPath = re2.ReplaceAllString(requestPath, "/$${id}")
	}
	return requestPath
}

// buildRequestPathForMutation builds request path for React Query mutations
func buildRequestPathForMutation(path string, hasIDParam, hasBodyData bool) string {
	requestPath := path
	// Remove "/api" prefix if it exists
	if strings.HasPrefix(requestPath, "/api") {
		requestPath = requestPath[4:]
	}

	if hasIDParam {
		re1 := regexp.MustCompile(`/:[^/]+`)
		requestPath = re1.ReplaceAllString(requestPath, "/$${id}")
		re2 := regexp.MustCompile(`/\{[^}]+\}`)
		requestPath = re2.ReplaceAllString(requestPath, "/$${id}")

		if hasBodyData {
			// Use variables.id for PUT/PATCH with both ID and body
			re3 := regexp.MustCompile(`\$\{id\}`)
			requestPath = re3.ReplaceAllString(requestPath, "$${variables.id}")
		} else {
			// Use variables directly for DELETE
			re3 := regexp.MustCompile(`\$\{id\}`)
			requestPath = re3.ReplaceAllString(requestPath, "$${variables}")
		}
	}

	return requestPath
}
