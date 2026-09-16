package mirror

import (
	"strings"
	"unicode"
)

// DeriveFilename converts a type name to a conventional Go filename
// Examples:
//   ClusterNetworking -> networking_types.go
//   PlatformSpec -> platformspec_types.go
//   APIServerNetworking -> apiservernetworking_types.go
func DeriveFilename(typeName string) string {
	// Strip common suffixes to get the core name
	baseName := typeName
	for _, suffix := range []string{"Spec", "Config", "Configuration", "Status"} {
		baseName = strings.TrimSuffix(baseName, suffix)
	}

	// If we stripped something, use the base name
	// Otherwise use the full type name
	if baseName != typeName && baseName != "" {
		typeName = baseName
	}

	// Convert to lowercase with underscores between words
	return toSnakeCase(typeName) + "_types.go"
}

// toSnakeCase converts PascalCase to snake_case
// Examples:
//   ClusterNetworking -> cluster_networking
//   APIServer -> api_server
//   HTTPProxy -> http_proxy
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Add underscore before uppercase letters (except at start)
		if i > 0 && unicode.IsUpper(r) {
			// Check if previous char is lowercase or next char is lowercase
			// This handles sequences like "HTTPProxy" -> "http_proxy" not "h_t_t_p_proxy"
			prevLower := unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

			if prevLower || nextLower {
				result.WriteRune('_')
			}
		}

		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}
