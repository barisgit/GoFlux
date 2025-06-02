package features

import (
	"fmt"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// GreetOptions configures the greeting display
type GreetOptions struct {
	ServiceName string
	Version     string
	Host        string
	Port        int
	ProxyPort   int
	DevMode     bool
	DocsPath    string // Optional: path to API docs (e.g., "/api/docs")
	OpenAPIPath string // Optional: path to OpenAPI spec (e.g., "/api/openapi")
}

// Greet displays the GoFlux logo and service information
func Greet(api huma.API, opts GreetOptions) {
	// GoFlux ASCII logo
	logo := `
 ██████╗  ██████╗ ███████╗██╗     ██╗   ██╗██╗  ██╗
██╔════╝ ██╔═══██╗██╔════╝██║     ██║   ██║╚██╗██╔╝
██║  ███╗██║   ██║█████╗  ██║     ██║   ██║ ╚███╔╝ 
██║   ██║██║   ██║██╔══╝  ██║     ██║   ██║ ██╔██╗ 
╚██████╔╝╚██████╔╝██║     ███████╗╚██████╔╝██╔╝ ██╗
 ╚═════╝  ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝ ╚═╝  ╚═╝`

	// Print the logo with some styling
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println(logo)
	fmt.Println(strings.Repeat("═", 60))

	// Service information
	if opts.ServiceName != "" {
		fmt.Printf("🚀 %s", opts.ServiceName)
		if opts.Version != "" {
			fmt.Printf(" v%s", opts.Version)
		}
		fmt.Println()
	}

	// Server information
	if opts.DevMode {
		fmt.Println("🛠️  Development mode enabled")
		if opts.Host != "" && opts.ProxyPort > 0 {
			if opts.ProxyPort > 0 {
				fmt.Printf("🌐 Proxy running on \x1b[32mhttp://%s:%d\x1b[0m\n", opts.Host, opts.ProxyPort)
				fmt.Printf("   (Direct access available on port %d)\n", opts.Port)
			}

			addr := fmt.Sprintf("%s:%d", opts.Host, opts.ProxyPort)
			// API documentation links
			if opts.DocsPath != "" {
				fmt.Printf("📚 API docs: http://%s%s\n", addr, opts.DocsPath)
			}
			if opts.OpenAPIPath != "" {
				fmt.Printf("📋 OpenAPI spec: http://%s%s.json\n", addr, opts.OpenAPIPath)
			}
		}
	} else if opts.Host != "" && opts.Port > 0 {
		addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
		fmt.Printf("🌐 Server running on http://%s\n", addr)
		if opts.DocsPath != "" {
			fmt.Printf("📚 API docs: http://%s%s\n", addr, opts.DocsPath)
		}
		if opts.OpenAPIPath != "" {
			fmt.Printf("📋 OpenAPI spec: http://%s%s.json\n", addr, opts.OpenAPIPath)
		}
	}

	fmt.Println(strings.Repeat("═", 60))
}

// QuickGreet is a simplified version that takes fewer parameters
func QuickGreet(serviceName, version, host string, port int) {
	// Simple GoFlux brand
	fmt.Println(strings.Repeat("═", 50))
	fmt.Println("    ⚡ GoFlux Framework")
	fmt.Println(strings.Repeat("═", 50))

	if serviceName != "" {
		fmt.Printf("🚀 %s", serviceName)
		if version != "" {
			fmt.Printf(" v%s", version)
		}
		fmt.Println()
	}

	if host != "" && port > 0 {
		fmt.Printf("🌐 Server: http://%s:%d\n", host, port)
	}

	fmt.Println(strings.Repeat("═", 50))
}
