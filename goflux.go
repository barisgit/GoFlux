// Package goflux provides the GoFlux framework for building full-stack Go applications.
// This allows users to import: github.com/barisgit/goflux
package goflux

import (
	"github.com/barisgit/goflux/base"
	"github.com/barisgit/goflux/openapi"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"
)

// AddOpenAPICommand adds an OpenAPI generation command to any cobra CLI
// This is a convenience function that wraps the dev package
func AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API) {
	base.AddOpenAPICommand(rootCmd, apiProvider)
}

// OpenAPI generation utilities - re-export from openapi package
var (
	GenerateSpecToFile = openapi.GenerateSpecToFile
	GenerateSpec       = openapi.GenerateSpec
	GenerateSpecYAML   = openapi.GenerateSpecYAML
	GetRouteCount      = openapi.GetRouteCount
)

// Health check utilities - re-export from base package
var (
	AddHealthCheck    = base.AddHealthCheck
	CustomHealthCheck = base.CustomHealthCheck
)

// Re-export types from base package
type StaticConfig = base.StaticConfig
type StaticResponse = base.StaticResponse
