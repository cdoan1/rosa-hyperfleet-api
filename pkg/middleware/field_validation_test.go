package middleware

import (
	"testing"

	"github.com/openshift/rosa-regional-platform-api/internal/codegen/featuregate"
	"github.com/openshift/rosa-regional-platform-api/internal/codegen/validation"
)

func TestFieldValidator_ValidateCreate_MutableFieldAllowed(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"displayName": "my-cluster",
	}
	if err := fv.ValidateCreate(spec, featuregate.Default, nil); err != nil {
		t.Errorf("expected mutable field to be allowed on create, got: %v", err)
	}
}

func TestFieldValidator_ValidateCreate_ServiceSetFieldRejected(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"accountId": "123456789012",
	}
	err := fv.ValidateCreate(spec, featuregate.Default, nil)
	if err == nil {
		t.Fatal("expected service-set field to be rejected on create")
	}
	valErrs, ok := err.(validation.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(valErrs) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(valErrs))
	}
	if valErrs[0].FieldPath != "spec.accountId" {
		t.Errorf("expected field path spec.accountId, got %s", valErrs[0].FieldPath)
	}
}

func TestFieldValidator_ValidateCreate_MultipleServiceSetFieldsRejected(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"accountId":  "123456789012",
		"creatorARN": "arn:aws:iam::123456789012:user/someone",
		"internalId": "abc-123",
	}
	err := fv.ValidateCreate(spec, featuregate.Default, nil)
	if err == nil {
		t.Fatal("expected service-set fields to be rejected")
	}
	valErrs, ok := err.(validation.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(valErrs) != 3 {
		t.Errorf("expected 3 validation errors, got %d: %v", len(valErrs), err)
	}
}

func TestFieldValidator_ValidateUpdate_MutableFieldAllowed(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"displayName": "new-name",
	}
	existingSpec := map[string]interface{}{
		"displayName": "old-name",
	}
	if err := fv.ValidateUpdate(spec, existingSpec, featuregate.Default, nil); err != nil {
		t.Errorf("expected mutable field to be allowed on update, got: %v", err)
	}
}

func TestFieldValidator_ValidateUpdate_ServiceSetFieldRejected(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"accountId": "new-account",
	}
	existingSpec := map[string]interface{}{
		"accountId": "old-account",
	}
	err := fv.ValidateUpdate(spec, existingSpec, featuregate.Default, nil)
	if err == nil {
		t.Fatal("expected service-set field to be rejected on update")
	}
}

func TestFieldValidator_ValidateCreate_FeatureGatedFieldRejected(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"tags": map[string]string{"env": "prod"},
	}
	err := fv.ValidateCreate(spec, featuregate.Default, nil)
	if err == nil {
		t.Fatal("expected feature-gated field to be rejected when gate not enabled")
	}
}

func TestFieldValidator_ValidateCreate_FeatureGatedFieldAllowedWithTechPreview(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"tags": map[string]string{"env": "prod"},
	}
	if err := fv.ValidateCreate(spec, featuregate.TechPreviewNoUpgrade, nil); err != nil {
		t.Errorf("expected feature-gated field to be allowed with TechPreview, got: %v", err)
	}
}

func TestFieldValidator_ValidateCreate_UnknownFieldAllowed(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"someRandomField": "value",
	}
	if err := fv.ValidateCreate(spec, featuregate.Default, nil); err != nil {
		t.Errorf("expected unknown field to pass through, got: %v", err)
	}
}

func TestFieldValidator_ValidateCreate_NestedServiceSetFieldRejected(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"hostedCluster": map[string]interface{}{
			"pullSecret": "my-secret",
		},
	}
	err := fv.ValidateCreate(spec, featuregate.Default, nil)
	if err == nil {
		t.Fatal("expected nested service-set field spec.hostedCluster.pullSecret to be rejected")
	}
}

func TestFieldValidator_ValidateCreate_MixedFields(t *testing.T) {
	fv := NewFieldValidator()
	spec := map[string]interface{}{
		"displayName": "my-cluster",
		"accountId":   "123456789012",
	}
	err := fv.ValidateCreate(spec, featuregate.Default, nil)
	if err == nil {
		t.Fatal("expected validation error for service-set field even when mixed with valid fields")
	}
	valErrs, ok := err.(validation.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(valErrs) != 1 {
		t.Errorf("expected exactly 1 error (for accountId only), got %d: %v", len(valErrs), err)
	}
}

func TestFlattenWithPrefix(t *testing.T) {
	input := map[string]interface{}{
		"displayName": "my-cluster",
		"hostedCluster": map[string]interface{}{
			"channel": "stable",
			"fips":    true,
		},
	}
	result := flattenWithPrefix("spec", input)

	expected := map[string]interface{}{
		"spec.displayName":           "my-cluster",
		"spec.hostedCluster.channel": "stable",
		"spec.hostedCluster.fips":    true,
	}

	if len(result) != len(expected) {
		t.Fatalf("expected %d keys, got %d: %v", len(expected), len(result), result)
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("key %s: expected %v, got %v", k, v, result[k])
		}
	}
}

func TestFlattenWithPrefix_EmptyMap(t *testing.T) {
	result := flattenWithPrefix("spec", map[string]interface{}{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}
