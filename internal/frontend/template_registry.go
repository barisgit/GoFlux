package frontend

import (
	"fmt"
	"strings"

	"github.com/barisgit/goflux/internal/config"
)

// TemplateInfo contains metadata about a frontend template
type TemplateInfo struct {
	Name        string
	Description string
	Framework   string
	InstallCmd  string
	DevCmd      string
	BuildCmd    string
	TypesDir    string
	LibDir      string
	StaticGen   config.StaticGenConfig
}

// TemplateRegistry manages available frontend templates
type TemplateRegistry struct {
	hardcodedTemplates map[string]*TemplateInfo
}

// NewTemplateRegistry creates a new template registry with built-in templates
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		hardcodedTemplates: make(map[string]*TemplateInfo),
	}

	// Register built-in templates
	registry.registerBuiltinTemplates()

	return registry
}

// registerBuiltinTemplates registers all hardcoded templates
func (r *TemplateRegistry) registerBuiltinTemplates() {
	// GoFlux Default template
	r.hardcodedTemplates["default"] = &TemplateInfo{
		Name:        "default",
		Description: "GoFlux default template with TanStack Router, React, and Tailwind",
		Framework:   "tanstack-router",
		InstallCmd:  "", // Will be copied from filesystem
		DevCmd:      "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     false,
			BuildSSRCmd: "cd frontend && pnpm build:ssr",
			GenerateCmd: "",
			Routes:      []string{"/", "/about"},
			SPARouting:  true,
		},
	}

	// TanStack Router template
	r.hardcodedTemplates["tanstack-router"] = &TemplateInfo{
		Name:        "tanstack-router",
		Description: "TanStack Router with TypeScript",
		Framework:   "tanstack-router",
		InstallCmd:  "pnpx create-tsrouter-app@latest . --template file-router",
		DevCmd:      "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     false,
			BuildSSRCmd: "cd frontend && pnpm build:ssr",
			GenerateCmd: "pnpx tsx scripts/generate-static.ts",
			Routes:      []string{"/", "/about"},
			SPARouting:  true,
		},
	}

	// Next.js template
	r.hardcodedTemplates["nextjs"] = &TemplateInfo{
		Name:        "nextjs",
		Description: "Next.js with TypeScript and Tailwind",
		Framework:   "nextjs",
		InstallCmd:  "pnpm create next-app@latest . --typescript --tailwind --eslint --app --src-dir --import-alias '@/*' --yes",
		DevCmd:      "cd frontend && pnpm dev --port {{port}}",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     true,
			BuildSSRCmd: "cd frontend && pnpm build && pnpm export",
			GenerateCmd: "",
			Routes:      []string{},
			SPARouting:  false,
		},
	}

	// Vite + React template
	r.hardcodedTemplates["vite-react"] = &TemplateInfo{
		Name:        "vite-react",
		Description: "Vite with React and TypeScript",
		Framework:   "vite-react",
		InstallCmd:  "pnpm create vite@latest . -- --template react-ts",
		DevCmd:      "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     false,
			BuildSSRCmd: "",
			GenerateCmd: "",
			Routes:      []string{},
			SPARouting:  false,
		},
	}

	// Minimal template (just package.json and basic structure)
	r.hardcodedTemplates["minimal"] = &TemplateInfo{
		Name:        "minimal",
		Description: "Minimal TypeScript setup",
		Framework:   "minimal",
		InstallCmd:  "", // Will be handled by generator
		DevCmd:      "cd frontend && pnpm dev --port {{port}}",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     false,
			BuildSSRCmd: "",
			GenerateCmd: "",
			Routes:      []string{},
			SPARouting:  false,
		},
	}

	// Vue 3 with TypeScript
	r.hardcodedTemplates["vue"] = &TemplateInfo{
		Name:        "vue",
		Description: "Vue 3 with TypeScript and Vite",
		Framework:   "vue",
		InstallCmd:  "pnpm create vue@latest . -- --typescript --yes",
		DevCmd:      "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     false,
			BuildSSRCmd: "",
			GenerateCmd: "",
			Routes:      []string{},
			SPARouting:  false,
		},
	}
	// SvelteKit template with TypeScript and non-interactive setup
	r.hardcodedTemplates["sveltekit"] = &TemplateInfo{
		Name:        "sveltekit",
		Description: "SvelteKit with TypeScript (minimal template)",
		Framework:   "sveltekit",
		InstallCmd:  "pnpx sv create . --template=minimal --types=ts --no-add-ons --install=pnpm",
		DevCmd:      "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:    "cd frontend && pnpm build",
		TypesDir:    "src/types",
		LibDir:      "src/lib",
		StaticGen: config.StaticGenConfig{
			Enabled:     true,
			BuildSSRCmd: "cd frontend && pnpm build",
			GenerateCmd: "",
			Routes:      []string{},
			SPARouting:  false,
		},
	}
}

// GetHardcodedTemplate returns a hardcoded template by name
func (r *TemplateRegistry) GetHardcodedTemplate(name string) (*TemplateInfo, bool) {
	template, exists := r.hardcodedTemplates[name]
	return template, exists
}

// ListHardcodedTemplates returns all available hardcoded template names
func (r *TemplateRegistry) ListHardcodedTemplates() []string {
	var names []string
	for name := range r.hardcodedTemplates {
		names = append(names, name)
	}
	return names
}

// GetTemplateByFramework returns a template that matches the framework name
func (r *TemplateRegistry) GetTemplateByFramework(framework string) (*TemplateInfo, bool) {
	// Normalize framework name
	framework = strings.ToLower(framework)

	// Direct match
	if template, exists := r.hardcodedTemplates[framework]; exists {
		return template, true
	}

	// Fuzzy matching for common patterns
	switch {
	case strings.Contains(framework, "tanstack"):
		return r.hardcodedTemplates["tanstack-router"], true
	case strings.Contains(framework, "next"):
		return r.hardcodedTemplates["nextjs"], true
	case strings.Contains(framework, "vite") && strings.Contains(framework, "react"):
		return r.hardcodedTemplates["vite-react"], true
	case strings.Contains(framework, "vue"):
		return r.hardcodedTemplates["vue"], true
	case strings.Contains(framework, "svelte"):
		return r.hardcodedTemplates["sveltekit"], true
	default:
		return nil, false
	}
}

// GetTemplateDescription returns a formatted description of a template
func (r *TemplateRegistry) GetTemplateDescription(name string) string {
	if template, exists := r.hardcodedTemplates[name]; exists {
		return fmt.Sprintf("%s - %s", template.Name, template.Description)
	}
	return ""
}

// GetAllTemplates returns all hardcoded templates
func (r *TemplateRegistry) GetAllTemplates() map[string]*TemplateInfo {
	return r.hardcodedTemplates
}
