package main

import (
	"fmt"
	"testing"

	"github.com/barisgit/goflux"
)

func TestImport(t *testing.T) {
	fmt.Println("✅ Successfully imported GoFlux!")
	fmt.Printf("📦 Available functions: %T\n", goflux.AddHealthCheck)
	fmt.Printf("🔧 OpenAPI utilities: %T\n", goflux.GenerateSpec)
}
