package generator

import (
	"regexp"
	"strings"

	"github.com/barisgit/goflux/cli/internal/typegen/config"
	"github.com/barisgit/goflux/cli/internal/typegen/processor"
)

// GeneratorUtils provides generator-specific utility functions
type GeneratorUtils struct {
	processor *processor.TypeProcessor
}

// NewGeneratorUtils creates a new generator utils instance
func NewGeneratorUtils(casingConfig *config.CasingConfig) *GeneratorUtils {
	return &GeneratorUtils{
		processor: processor.NewTypeProcessor(casingConfig),
	}
}

// Legacy function for backward compatibility - use processor instead
func SanitizeTypeScriptTypeName(name string) string {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.ProcessTypeName(name)
}

// buildRequestPath builds a request path with parameter substitution
func buildRequestPath(path string, hasIDParam bool) string {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.BuildRequestPath(path, hasIDParam)
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
		requestPath = re1.ReplaceAllString(requestPath, "/${id}")
		re2 := regexp.MustCompile(`/\{[^}]+\}`)
		requestPath = re2.ReplaceAllString(requestPath, "/${id}")

		if hasBodyData {
			// Use variables.id for PUT/PATCH with both ID and body
			re3 := regexp.MustCompile(`\$\{id\}`)
			requestPath = re3.ReplaceAllString(requestPath, "${variables.id}")
		} else {
			// Use variables directly for DELETE
			re3 := regexp.MustCompile(`\$\{id\}`)
			requestPath = re3.ReplaceAllString(requestPath, "${variables}")
		}
	}

	return requestPath
}

// Legacy utility functions - use processor instead where possible
func capitalize(s string) string {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.ProcessMethodName(s)
}

func singularize(s string) string {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.Singularize(s)
}

func isValidJSIdentifier(s string) bool {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.IsValidJSIdentifier(s)
}

func extractCleanDescription(description string) string {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.CleanDescription(description)
}

func contains(slice []string, item string) bool {
	utils := NewGeneratorUtils(config.DefaultCasingConfig())
	return utils.processor.ContainsString(slice, item)
}
