package goflux

import (
	"github.com/barisgit/goflux/pkg/base"
	"github.com/barisgit/goflux/pkg/dev"
	"github.com/barisgit/goflux/pkg/openapi"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"
)

// AddOpenAPICommand adds an OpenAPI generation command to any cobra CLI
// This is a convenience function that wraps the dev package
func AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API) {
	dev.AddOpenAPICommand(rootCmd, apiProvider)
}

// OpenAPI generation utilities - re-export from openapi package
var (
	GenerateSpecToFile = openapi.GenerateSpecToFile
	GenerateSpec       = openapi.GenerateSpec
	GenerateSpecYAML   = openapi.GenerateSpecYAML
	GetRouteCount      = openapi.GetRouteCount
)

var (
	AddHealthCheck    = base.AddHealthCheck
	CustomHealthCheck = base.CustomHealthCheck
	StaticHandler     = base.StaticHandler
)

// Re-export types from base package
type StaticConfig = base.StaticConfig
