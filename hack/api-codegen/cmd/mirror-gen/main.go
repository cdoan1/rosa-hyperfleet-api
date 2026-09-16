package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift-online/rosa-hyperfleet-api/hack/api-codegen/pkg/mirror"
)

func main() {
	var (
		importPath       string
		typeName         string
		outputPath       string
		defaultWriteMode string
		defaultOpenAPI   bool
		packageName      string
	)

	flag.StringVar(&importPath, "import-path", "", "Go import path to resolve via go.mod (required)")
	flag.StringVar(&typeName, "type", "", "Type name to generate mirror for (required)")
	flag.StringVar(&outputPath, "output", "api/v1alpha1", "Output directory or file path (default: api/v1alpha1)")
	flag.StringVar(&defaultWriteMode, "default-write-mode", "mutable", "Default write mode for fields (mutable, immutable, service-set)")
	flag.BoolVar(&defaultOpenAPI, "default-openapi", true, "Default value for +k8s:openapi-gen marker")
	flag.StringVar(&packageName, "package", "v1alpha1", "Package name for generated code")
	flag.Parse()

	// Validate flags
	if importPath == "" || typeName == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Validate write mode
	validWriteModes := map[string]bool{
		"mutable":     true,
		"immutable":   true,
		"service-set": true,
	}
	if !validWriteModes[defaultWriteMode] {
		log.Fatalf("Invalid write mode: %s (must be mutable, immutable, or service-set)", defaultWriteMode)
	}

	// Determine output file path
	// If outputPath is a directory or doesn't end in .go, derive filename from type
	outputFile := outputPath
	if !strings.HasSuffix(outputPath, ".go") {
		// Derive filename from type name
		filename := mirror.DeriveFilename(typeName)
		outputFile = filepath.Join(outputPath, filename)
		log.Printf("Derived output file: %s", outputFile)
	}

	// Create generator
	log.Printf("Resolving import path: %s", importPath)
	gen, err := mirror.NewGeneratorFromImportPath(importPath, typeName)
	if err != nil {
		log.Fatalf("Failed to resolve import path: %v", err)
	}
	log.Printf("Resolved to directory: %s", gen.SourceDir)

	gen.OutputPackage = packageName
	gen.DefaultWriteMode = defaultWriteMode
	gen.DefaultOpenAPI = defaultOpenAPI

	// Load source files
	log.Printf("Loading source files from: %s", gen.SourceDir)
	if err := gen.LoadSourceFiles(gen.SourceDir); err != nil {
		log.Fatalf("Failed to load source files: %v", err)
	}

	log.Printf("Loaded %d source files", len(gen.ParsedFiles()))

	// Generate mirror type scaffold
	log.Printf("Generating mirror type scaffold for: %s", typeName)
	if err := gen.Generate(outputFile); err != nil {
		log.Fatalf("Failed to generate: %v", err)
	}

	fmt.Printf("Successfully generated mirror type scaffold in %s\n", outputFile)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review and customize markers in %s\n", outputFile)
	fmt.Printf("  2. Run: make generate-deepcopy\n")
	fmt.Printf("  3. Update references to use the new mirror type\n")
}
