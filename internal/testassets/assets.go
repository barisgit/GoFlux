package testassets

import "embed"

// TestFS holds the embedded test assets for use across adapter tests.
//
//go:embed assets/*
var TestFS embed.FS

// Empty embed.FS for basic tests
var EmptyFS embed.FS
