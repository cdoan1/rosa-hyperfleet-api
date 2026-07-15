package conversion

import "encoding/json"

// SpecToMap converts a typed spec struct to map[string]any via JSON round-trip.
// Used at boundaries that require maps (validator, Hyperfleet wire protocol).
func SpecToMap(spec any) (map[string]any, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// MapToSpec converts a map[string]any to a typed spec struct via JSON round-trip.
// Used to parse Hyperfleet wire responses into typed structs.
func MapToSpec[T any](m map[string]any) (*T, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var spec T
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
