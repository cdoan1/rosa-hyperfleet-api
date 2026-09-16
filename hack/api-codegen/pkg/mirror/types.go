package mirror

import (
	"go/ast"
)

// Generator generates mirror type scaffolds from upstream types
type Generator struct {
	// SourceDir is the directory containing source Go files
	SourceDir string

	// TypeName is the type name to generate mirror for
	TypeName string

	// OutputPackage is the package name for generated code
	OutputPackage string

	// DefaultWriteMode is the default write mode for fields
	DefaultWriteMode string

	// DefaultOpenAPI is the default value for +k8s:openapi-gen marker
	DefaultOpenAPI bool

	// SourcePackage is the import path of the source package
	SourcePackage string

	// SourcePackageAlias is the alias to use for the source package import
	SourcePackageAlias string

	// parsedFiles holds parsed AST of source files
	parsedFiles map[string]*ast.File
}

// ParsedFiles returns the parsed files
func (g *Generator) ParsedFiles() map[string]*ast.File {
	return g.parsedFiles
}

// TypeDef represents a generated mirror type definition
type TypeDef struct {
	// Name is the generated type name
	Name string

	// SourceName is the original type name
	SourceName string

	// SourcePackage is the import path being mirrored
	SourcePackage string

	// SourcePackageAlias is the alias for the import
	SourcePackageAlias string

	// Fields are the struct fields
	Fields []FieldDef

	// Doc is the type documentation
	Doc string
}

// FieldDef represents a single field in a mirror type
type FieldDef struct {
	// Name is the Go field name
	Name string

	// Type is the Go type (as a string)
	Type string

	// JSONTag is the json struct tag
	JSONTag string

	// Doc is the field documentation
	Doc string

	// Markers are the Go markers to include
	Markers []string
}

// NewGenerator creates a new mirror generator
func NewGenerator(sourceDir, typeName string) *Generator {
	return &Generator{
		SourceDir:        sourceDir,
		TypeName:         typeName,
		OutputPackage:    "v1alpha1",
		DefaultWriteMode: "mutable",
		DefaultOpenAPI:   true,
		parsedFiles:      make(map[string]*ast.File),
	}
}
