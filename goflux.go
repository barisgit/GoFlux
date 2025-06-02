// Package goflux provides the GoFlux framework for building full-stack Go applications.
// This allows users to import: github.com/barisgit/goflux
package goflux

import (
	"github.com/barisgit/goflux/goflux"
	"github.com/barisgit/goflux/openapi"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"
)

// AddOpenAPICommand adds an OpenAPI generation command to any cobra CLI
// This is a convenience function that wraps the dev package
func AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API) {
	goflux.AddOpenAPICommand(rootCmd, apiProvider)
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
	AddHealthCheck    = goflux.AddHealthCheck
	CustomHealthCheck = goflux.CustomHealthCheck
)

// Greeting utilities - re-export from base package
var (
	Greet      = goflux.Greet
	QuickGreet = goflux.QuickGreet
)

// File upload utilities - re-export from base package
var (
	NewFile               = goflux.NewFile
	NewFileList           = goflux.NewFileList
	NewFileUploadResponse = goflux.NewFileUploadResponse
	GetFileFromForm       = goflux.GetFileFromForm
	GetFormValue          = goflux.GetFormValue
)

// File upload errors - re-export from base package
var (
	ErrNoFileUploaded     = goflux.ErrNoFileUploaded
	ErrFileTooLarge       = goflux.ErrFileTooLarge
	ErrInvalidFileType    = goflux.ErrInvalidFileType
	ErrTooManyFiles       = goflux.ErrTooManyFiles
	ErrInvalidFileContent = goflux.ErrInvalidFileContent
	NewFileUploadError    = goflux.NewFileUploadError
)

// Re-export types from base package
type StaticConfig = goflux.StaticConfig
type StaticResponse = goflux.StaticResponse
type GreetOptions = goflux.GreetOptions

// File upload types - re-export from base package
type File = goflux.File
type FileList = goflux.FileList
type FileUploadResponseBody = goflux.FileUploadResponseBody
type FileInfo = goflux.FileInfo
type FileUploadError = goflux.FileUploadError
