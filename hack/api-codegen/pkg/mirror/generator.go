package mirror

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

const mirrorTemplate = `package {{ .PackageName }}

{{ if .Imports }}
import (
{{- range .Imports }}
	{{ . }}
{{- end }}
)
{{ end }}

// {{ .Type.Name }} mirrors upstream {{ .Type.SourceName }} with custom markers.
// +hyperfleet:upstream-reduced-object={{ .Type.SourcePackageAlias }}.{{ .Type.SourceName }}
type {{ .Type.Name }} struct {
{{- range .Type.Fields }}
	{{- if .Doc }}
	// {{ .Doc }}
	{{- end }}
	{{- range .Markers }}
	// {{ . }}
	{{- end }}
	{{ .Name }} {{ .Type }} ` + "`json:\"{{ .JSONTag }}\"`" + `
{{- end }}
}
`

type templateData struct {
	PackageName string
	Imports     []string
	Type        *TypeDef
}

// Generate creates Go source file for the mirror type
func (g *Generator) Generate(outputFile string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate TypeDef
	typeDef, err := g.GenerateTypeDef()
	if err != nil {
		return fmt.Errorf("generating type def: %w", err)
	}

	// Collect unique imports needed
	imports := g.collectImports(typeDef)

	// Prepare template data
	data := templateData{
		PackageName: g.OutputPackage,
		Imports:     imports,
		Type:        typeDef,
	}

	// Execute template
	tmpl, err := template.New("mirror").Parse(mirrorTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		if writeErr := os.WriteFile(outputFile+".unformatted", buf.Bytes(), 0644); writeErr != nil {
			return fmt.Errorf("formatting generated code: %w (also failed to write unformatted: %v)", err, writeErr)
		}
		return fmt.Errorf("formatting generated code: %w (unformatted written to %s.unformatted)", err, outputFile)
	}

	// Write to file
	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	return nil
}

// collectImports extracts unique imports needed for the generated type
func (g *Generator) collectImports(typeDef *TypeDef) []string {
	importSet := make(map[string]bool)

	// Add source package import with alias
	if g.SourcePackage != "" && g.SourcePackageAlias != "" {
		importSet[fmt.Sprintf(`%s "%s"`, g.SourcePackageAlias, g.SourcePackage)] = true
	}

	for _, field := range typeDef.Fields {
		// Strip pointer/slice/map prefixes to extract the base type
		typeStr := field.Type
		for strings.HasPrefix(typeStr, "*") || strings.HasPrefix(typeStr, "[]") {
			typeStr = strings.TrimPrefix(typeStr, "*")
			typeStr = strings.TrimPrefix(typeStr, "[]")
		}
		if strings.HasPrefix(typeStr, "map[") {
			// Extract value type from map[K]V
			if idx := strings.LastIndex(typeStr, "]"); idx != -1 && idx+1 < len(typeStr) {
				typeStr = typeStr[idx+1:]
				typeStr = strings.TrimPrefix(typeStr, "*")
			}
		}

		// Extract package from qualified type names
		if strings.Contains(typeStr, ".") {
			// Common k8s/OpenShift imports
			imports := map[string]string{
				"configv1.":  `configv1 "github.com/openshift/api/config/v1"`,
				"corev1.":    `corev1 "k8s.io/api/core/v1"`,
				"metav1.":    `metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"`,
				"rbacv1.":    `rbacv1 "k8s.io/api/rbac/v1"`,
				"networkv1.": `networkv1 "k8s.io/api/networking/v1"`,
			}

			for prefix, imp := range imports {
				if strings.HasPrefix(typeStr, prefix) {
					importSet[imp] = true
				}
			}
		}
	}

	// Convert set to sorted slice for deterministic output
	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	return imports
}
