package mirror

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// NewGeneratorFromImportPath creates a generator by resolving an import path via go.mod
func NewGeneratorFromImportPath(importPath, typeName string) (*Generator, error) {
	// Find a module context to run go list from (try api/ first, then root)
	moduleDirs := []string{"api", "."}
	var sourceDir string
	var lastErr error

	for _, modDir := range moduleDirs {
		cmd := exec.Command("go", "list", "-f", "{{.Dir}}", importPath)
		cmd.Dir = modDir
		out, err := cmd.Output()
		if err != nil {
			lastErr = err
			continue
		}

		sourceDir = strings.TrimSpace(string(out))
		if sourceDir != "" {
			break
		}
	}

	if sourceDir == "" {
		if lastErr != nil {
			return nil, fmt.Errorf("failed to resolve import path %s: %w", importPath, lastErr)
		}
		return nil, fmt.Errorf("import path %s resolved to empty directory", importPath)
	}

	// Verify directory exists
	if _, err := os.Stat(sourceDir); err != nil {
		return nil, fmt.Errorf("resolved directory %s does not exist: %w", sourceDir, err)
	}

	// Create package alias from import path
	alias := extractPackageAlias(importPath)

	gen := NewGenerator(sourceDir, typeName)
	gen.SourcePackage = importPath
	gen.SourcePackageAlias = alias

	return gen, nil
}

// extractPackageAlias extracts a sensible package alias from an import path
// e.g. "github.com/openshift/hypershift/api/hypershift/v1beta1" → "hypershiftv1beta1"
func extractPackageAlias(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return "upstream"
	}

	// For paths like .../hypershift/v1beta1, combine last two parts
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		secondLast := parts[len(parts)-2]

		// If last part is a version (v1beta1, v1alpha1, etc.), combine with previous
		if strings.HasPrefix(lastPart, "v") && len(lastPart) > 1 {
			alias := secondLast + lastPart
			return strings.ReplaceAll(alias, "-", "")
		}
	}

	// Otherwise just use the last part
	return strings.ReplaceAll(parts[len(parts)-1], "-", "")
}
