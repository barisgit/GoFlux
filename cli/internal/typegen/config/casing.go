package config

import (
	"regexp"
	"strings"
	"unicode"
)

// CaseType represents different naming conventions
type CaseType int

const (
	Unknown            CaseType = iota
	CamelCase                   // camelCase
	PascalCase                  // PascalCase
	SnakeCase                   // snake_case
	KebabCase                   // kebab-case
	ScreamingSnakeCase          // SCREAMING_SNAKE_CASE
	DotCase                     // dot.case
)

// CasingConfig holds configuration for case conversion
type CasingConfig struct {
	TypeNames     CaseType `json:"typeNames" yaml:"typeNames"`         // Default: PascalCase
	FieldNames    CaseType `json:"fieldNames" yaml:"fieldNames"`       // Default: CamelCase
	MethodNames   CaseType `json:"methodNames" yaml:"methodNames"`     // Default: CamelCase
	ConstantNames CaseType `json:"constantNames" yaml:"constantNames"` // Default: ScreamingSnakeCase
	VariableNames CaseType `json:"variableNames" yaml:"variableNames"` // Default: CamelCase
	FileNames     CaseType `json:"fileNames" yaml:"fileNames"`         // Default: KebabCase
}

// DefaultCasingConfig returns the default casing configuration
func DefaultCasingConfig() *CasingConfig {
	return &CasingConfig{
		TypeNames:     PascalCase,
		FieldNames:    CamelCase,
		MethodNames:   CamelCase,
		ConstantNames: ScreamingSnakeCase,
		VariableNames: CamelCase,
		FileNames:     KebabCase,
	}
}

// CaseConverter provides robust case conversion functionality
type CaseConverter struct {
	config *CasingConfig
}

// NewCaseConverter creates a new case converter with the given configuration
func NewCaseConverter(config *CasingConfig) *CaseConverter {
	if config == nil {
		config = DefaultCasingConfig()
	}
	return &CaseConverter{config: config}
}

// DetectCase attempts to detect the case type of a string
func (c *CaseConverter) DetectCase(s string) CaseType {
	if s == "" {
		return Unknown
	}

	// Check for specific patterns
	switch {
	case c.isPascalCase(s):
		return PascalCase
	case c.isCamelCase(s):
		return CamelCase
	case c.isScreamingSnakeCase(s):
		return ScreamingSnakeCase
	case c.isSnakeCase(s):
		return SnakeCase
	case c.isKebabCase(s):
		return KebabCase
	case c.isDotCase(s):
		return DotCase
	default:
		return Unknown
	}
}

// Convert converts a string from one case to another
func (c *CaseConverter) Convert(s string, targetCase CaseType) string {
	if s == "" {
		return s
	}

	// First, break the string into words
	words := c.splitIntoWords(s)
	if len(words) == 0 {
		return s
	}

	// Convert to target case
	switch targetCase {
	case CamelCase:
		return c.toCamelCase(words)
	case PascalCase:
		return c.toPascalCase(words)
	case SnakeCase:
		return c.toSnakeCase(words)
	case KebabCase:
		return c.toKebabCase(words)
	case ScreamingSnakeCase:
		return c.toScreamingSnakeCase(words)
	case DotCase:
		return c.toDotCase(words)
	default:
		return s
	}
}

// ConvertTypeName converts a string to the configured type name case
func (c *CaseConverter) ConvertTypeName(s string) string {
	return c.Convert(s, c.config.TypeNames)
}

// ConvertFieldName converts a string to the configured field name case
func (c *CaseConverter) ConvertFieldName(s string) string {
	return c.Convert(s, c.config.FieldNames)
}

// ConvertMethodName converts a string to the configured method name case
func (c *CaseConverter) ConvertMethodName(s string) string {
	return c.Convert(s, c.config.MethodNames)
}

// ConvertConstantName converts a string to the configured constant name case
func (c *CaseConverter) ConvertConstantName(s string) string {
	return c.Convert(s, c.config.ConstantNames)
}

// ConvertVariableName converts a string to the configured variable name case
func (c *CaseConverter) ConvertVariableName(s string) string {
	return c.Convert(s, c.config.VariableNames)
}

// ConvertFileName converts a string to the configured file name case
func (c *CaseConverter) ConvertFileName(s string) string {
	return c.Convert(s, c.config.FileNames)
}

// Helper methods for case detection
func (c *CaseConverter) isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// First character must be uppercase
	if !unicode.IsUpper(rune(s[0])) {
		return false
	}
	// Should not contain separators
	return !strings.ContainsAny(s, "_-. ")
}

func (c *CaseConverter) isCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// First character must be lowercase
	if !unicode.IsLower(rune(s[0])) {
		return false
	}
	// Should not contain separators
	return !strings.ContainsAny(s, "_-. ")
}

func (c *CaseConverter) isSnakeCase(s string) bool {
	if !strings.Contains(s, "_") {
		return false
	}
	// Should be all lowercase with underscores
	return strings.ToLower(s) == s && !strings.ContainsAny(s, "- .")
}

func (c *CaseConverter) isScreamingSnakeCase(s string) bool {
	if !strings.Contains(s, "_") {
		return false
	}
	// Should be all uppercase with underscores
	return strings.ToUpper(s) == s && !strings.ContainsAny(s, "- .")
}

func (c *CaseConverter) isKebabCase(s string) bool {
	if !strings.Contains(s, "-") {
		return false
	}
	// Should be all lowercase with hyphens
	return strings.ToLower(s) == s && !strings.ContainsAny(s, "_ .")
}

func (c *CaseConverter) isDotCase(s string) bool {
	if !strings.Contains(s, ".") {
		return false
	}
	// Should be all lowercase with dots
	return strings.ToLower(s) == s && !strings.ContainsAny(s, "_- ")
}

// splitIntoWords breaks a string into words regardless of the input case
func (c *CaseConverter) splitIntoWords(s string) []string {
	if s == "" {
		return nil
	}

	var words []string

	// Handle different separators
	if strings.ContainsAny(s, "_-. ") {
		// Split on separators
		separatorRegex := regexp.MustCompile(`[_\-.\s]+`)
		words = separatorRegex.Split(s, -1)
	} else {
		// Handle camelCase/PascalCase by splitting on uppercase letters
		words = c.splitCamelCase(s)
	}

	// Filter out empty strings and normalize
	var result []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" {
			result = append(result, strings.ToLower(word))
		}
	}

	return result
}

// splitCamelCase splits camelCase or PascalCase strings into words
func (c *CaseConverter) splitCamelCase(s string) []string {
	var words []string
	var currentWord strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// Check if this is not an acronym
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
		currentWord.WriteRune(r)
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// Case conversion methods
func (c *CaseConverter) toCamelCase(words []string) string {
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	for i, word := range words {
		if i == 0 {
			result.WriteString(strings.ToLower(word))
		} else {
			result.WriteString(c.capitalize(word))
		}
	}
	return result.String()
}

func (c *CaseConverter) toPascalCase(words []string) string {
	var result strings.Builder
	for _, word := range words {
		result.WriteString(c.capitalize(word))
	}
	return result.String()
}

func (c *CaseConverter) toSnakeCase(words []string) string {
	var result []string
	for _, word := range words {
		result = append(result, strings.ToLower(word))
	}
	return strings.Join(result, "_")
}

func (c *CaseConverter) toKebabCase(words []string) string {
	var result []string
	for _, word := range words {
		result = append(result, strings.ToLower(word))
	}
	return strings.Join(result, "-")
}

func (c *CaseConverter) toScreamingSnakeCase(words []string) string {
	var result []string
	for _, word := range words {
		result = append(result, strings.ToUpper(word))
	}
	return strings.Join(result, "_")
}

func (c *CaseConverter) toDotCase(words []string) string {
	var result []string
	for _, word := range words {
		result = append(result, strings.ToLower(word))
	}
	return strings.Join(result, ".")
}

func (c *CaseConverter) capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// Singularize attempts to convert plural words to singular
func (c *CaseConverter) Singularize(s string) string {
	if len(s) <= 1 {
		return s
	}

	// Simple pluralization rules - extend as needed
	lower := strings.ToLower(s)

	// Handle special cases
	specialCases := map[string]string{
		"children": "child",
		"people":   "person",
		"men":      "man",
		"women":    "woman",
		"feet":     "foot",
		"teeth":    "tooth",
		"geese":    "goose",
		"mice":     "mouse",
	}

	if singular, exists := specialCases[lower]; exists {
		// Preserve original casing
		if strings.ToUpper(s) == s {
			return strings.ToUpper(singular)
		} else if unicode.IsUpper(rune(s[0])) {
			return c.capitalize(singular)
		}
		return singular
	}

	// Regular rules
	if strings.HasSuffix(lower, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(lower, "es") && len(s) > 2 {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(lower, "s") && len(s) > 1 {
		return s[:len(s)-1]
	}

	return s
}

// IsValidJSIdentifier checks if a string is a valid JavaScript identifier
func (c *CaseConverter) IsValidJSIdentifier(s string) bool {
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
