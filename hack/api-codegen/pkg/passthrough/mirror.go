package passthrough

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// ScanForMirrorTypes scans the output directory for local mirror types that
// should be used instead of upstream types. Returns a map of upstream type name -> local type name.
func (g *Generator) ScanForMirrorTypes(outputDir string) error {
	g.localMirrorTypes = make(map[string]string)

	// Parse all Go files in the output directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, outputDir, func(fi os.FileInfo) bool {
		name := fi.Name()
		return !strings.HasSuffix(name, "_test.go") &&
			!strings.HasPrefix(name, "zz_generated")
	}, parser.ParseComments)

	if err != nil {
		return fmt.Errorf("parsing output directory %s: %w", outputDir, err)
	}

	// Scan for types with +hyperfleet:upstream-reduced-object marker
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					// Check if this type has the upstream-reduced-object marker
					if genDecl.Doc != nil {
						for _, comment := range genDecl.Doc.List {
							text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
							if strings.HasPrefix(text, "+hyperfleet:upstream-reduced-object=") {
								// Extract upstream type reference
								// e.g., "+hyperfleet:upstream-reduced-object=hypershiftv1beta1.ClusterNetworking"
								upstreamRef := strings.TrimPrefix(text, "+hyperfleet:upstream-reduced-object=")

								// Strip package prefix to get just the type name
								// "hypershiftv1beta1.ClusterNetworking" -> "ClusterNetworking"
								parts := strings.Split(upstreamRef, ".")
								if len(parts) > 0 {
									upstreamTypeName := parts[len(parts)-1]
									localTypeName := typeSpec.Name.Name
									g.localMirrorTypes[upstreamTypeName] = localTypeName
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// resolveTypeName checks if an upstream type has a local mirror and returns the appropriate type name
func (g *Generator) resolveTypeName(typeName string) string {
	// Check if this is a qualified type like "hypershiftv1beta1.ClusterNetworking"
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) == 2 {
			packageAlias := parts[0]
			upstreamTypeName := parts[1]

			// If this is from our source package and has a local mirror, use it
			if packageAlias == g.SourcePackageAlias {
				if localName, exists := g.localMirrorTypes[upstreamTypeName]; exists {
					return localName
				}
			}
		}
	}

	// No mirror found, return as-is
	return typeName
}
