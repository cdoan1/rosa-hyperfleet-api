package middleware

import (
	"github.com/cdoan1/hyperfleet-api-codegen/pkg/featuregate"
	"github.com/cdoan1/hyperfleet-api-codegen/pkg/validation"
)

type FieldValidator struct {
	validator *validation.Validator
}

func NewFieldValidator() *FieldValidator {
	return &FieldValidator{
		validator: validation.NewValidator(),
	}
}

func (fv *FieldValidator) ValidateCreate(spec map[string]interface{}, featureSet featuregate.FeatureSet, enabledGates []string) error {
	fields := flattenWithPrefix("spec", spec)
	return fv.validator.Validate(&validation.Request{
		Operation:    validation.OperationCreate,
		Fields:       fields,
		FeatureSet:   featureSet,
		EnabledGates: enabledGates,
	})
}

func (fv *FieldValidator) ValidateUpdate(spec, existingSpec map[string]interface{}, featureSet featuregate.FeatureSet, enabledGates []string) error {
	fields := flattenWithPrefix("spec", spec)
	existing := flattenWithPrefix("spec", existingSpec)
	return fv.validator.Validate(&validation.Request{
		Operation:      validation.OperationUpdate,
		Fields:         fields,
		ExistingFields: existing,
		FeatureSet:     featureSet,
		EnabledGates:   enabledGates,
	})
}

func flattenWithPrefix(prefix string, m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		key := prefix + "." + k
		if nested, ok := v.(map[string]interface{}); ok {
			for nk, nv := range flattenWithPrefix(key, nested) {
				result[nk] = nv
			}
		} else {
			result[key] = v
		}
	}
	return result
}
