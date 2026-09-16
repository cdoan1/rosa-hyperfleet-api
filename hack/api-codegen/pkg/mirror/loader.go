package mirror

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// LoadSourceFiles loads and parses Go source files from a directory
func (g *Generator) LoadSourceFiles(sourceDir string) error {
	fset := token.NewFileSet()

	// Parse all Go files in the directory
	//nolint:staticcheck // ParseDir is sufficient for our use case
	pkgs, err := parser.ParseDir(fset, sourceDir, func(fi os.FileInfo) bool {
		// Skip test files and generated files
		name := fi.Name()
		return !strings.HasSuffix(name, "_test.go") &&
			!strings.HasPrefix(name, "zz_generated")
	}, parser.ParseComments)

	if err != nil {
		return fmt.Errorf("parsing directory %s: %w", sourceDir, err)
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("no packages found in %s", sourceDir)
	}

	// Store parsed files
	g.parsedFiles = make(map[string]*ast.File)
	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			g.parsedFiles[filename] = file
		}
	}

	return nil
}

// GenerateTypeDef creates a mirror type definition for a source type
func (g *Generator) GenerateTypeDef() (*TypeDef, error) {
	// Find the type definition across all parsed files
	var typeSpec *ast.TypeSpec
	for _, file := range g.parsedFiles {
		ast.Inspect(file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == g.TypeName {
				typeSpec = ts
				return false
			}
			return true
		})
		if typeSpec != nil {
			break
		}
	}

	if typeSpec == nil {
		return nil, fmt.Errorf("type %s not found in parsed files", g.TypeName)
	}

	// Ensure it's a struct type
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("type %s is not a struct", g.TypeName)
	}

	typeDef := &TypeDef{
		Name:               g.TypeName,
		SourceName:         g.TypeName,
		SourcePackage:      g.SourcePackage,
		SourcePackageAlias: g.SourcePackageAlias,
		Doc:                fmt.Sprintf("%s mirrors upstream %s with custom markers.", g.TypeName, g.TypeName),
		Fields:             make([]FieldDef, 0),
	}

	// Process each field
	for _, field := range structType.Fields.List {
		// Skip fields without names (embedded types)
		if len(field.Names) == 0 {
			continue
		}

		for _, name := range field.Names {
			// Skip unexported fields
			if !name.IsExported() {
				continue
			}

			fieldDef := g.createFieldDef(name.Name, field)
			typeDef.Fields = append(typeDef.Fields, fieldDef)
		}
	}

	return typeDef, nil
}

// createFieldDef creates a field definition with default markers
func (g *Generator) createFieldDef(fieldName string, field *ast.Field) FieldDef {
	fieldDef := FieldDef{
		Name: fieldName,
		Type: g.typeToString(field.Type),
	}

	// Extract JSON tag
	if field.Tag != nil {
		tag := strings.Trim(field.Tag.Value, "`")
		if jsonTag := parseStructTag(tag, "json"); jsonTag != "" {
			fieldDef.JSONTag = jsonTag
		}
	}

	// Extract documentation from upstream comments
	if field.Doc != nil {
		var docLines []string
		for _, comment := range field.Doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			// Skip marker lines
			if text != "" && !strings.HasPrefix(text, "+") {
				docLines = append(docLines, text)
			}
		}
		if len(docLines) > 0 {
			fieldDef.Doc = strings.Join(docLines, " ")
		}
	}

	// Add default markers
	fieldDef.Markers = g.generateDefaultMarkers()

	return fieldDef
}

// generateDefaultMarkers creates default markers based on generator config
func (g *Generator) generateDefaultMarkers() []string {
	markers := make([]string, 0, 2)

	if g.DefaultOpenAPI {
		markers = append(markers, "+k8s:openapi-gen=true")
	} else {
		markers = append(markers, "+k8s:openapi-gen=false")
	}

	if g.DefaultWriteMode != "" {
		markers = append(markers, fmt.Sprintf("+hyperfleet:write-mode=%s", g.DefaultWriteMode))
	}

	return markers
}

// typeToString converts an AST type expression to a string, preserving upstream references
func (g *Generator) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		typeName := t.Name
		// Check if this is a type defined in the source package (not a builtin)
		if g.SourcePackageAlias != "" && g.isSourcePackageType(typeName) {
			return g.SourcePackageAlias + "." + typeName
		}
		return typeName
	case *ast.StarExpr:
		return "*" + g.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + g.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + g.typeToString(t.Key) + "]" + g.typeToString(t.Value)
	case *ast.SelectorExpr:
		// Preserve upstream package references like configv1.URL
		return g.typeToString(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

// isSourcePackageType checks if a type name is defined in the source package
func (g *Generator) isSourcePackageType(typeName string) bool {
	// Built-in types don't need qualification
	builtins := map[string]bool{
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true, "uint": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	}

	if builtins[typeName] {
		return false
	}

	// Check if the type is defined in the parsed source files
	for _, file := range g.parsedFiles {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == typeName {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// parseStructTag extracts a specific tag value from struct tag string
func parseStructTag(tag, key string) string {
	// Simple tag parser - handles: `json:"name,omitempty" yaml:"name"`
	parts := strings.Fields(tag)
	prefix := key + `:"`

	for _, part := range parts {
		if strings.HasPrefix(part, prefix) {
			value := strings.TrimPrefix(part, prefix)
			value = strings.TrimSuffix(value, `"`)
			return value
		}
	}

	return ""
}
