package mapper

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestIsValidSubnetID(t *testing.T) {
	tests := []struct {
		name     string
		subnetID string
		valid    bool
	}{
		{
			name:     "valid subnet ID - 8 chars",
			subnetID: "subnet-abc12345",
			valid:    true,
		},
		{
			name:     "valid subnet ID - 17 chars",
			subnetID: "subnet-0a1b2c3d4e5f67890",
			valid:    true,
		},
		{
			name:     "valid subnet ID - mixed hex",
			subnetID: "subnet-0123456789abcdef0",
			valid:    true,
		},
		{
			name:     "invalid - too short",
			subnetID: "subnet-abc",
			valid:    false,
		},
		{
			name:     "invalid - too long",
			subnetID: "subnet-0123456789abcdef012",
			valid:    false,
		},
		{
			name:     "invalid - missing prefix",
			subnetID: "abc12345",
			valid:    false,
		},
		{
			name:     "invalid - wrong prefix",
			subnetID: "vpc-abc12345",
			valid:    false,
		},
		{
			name:     "invalid - uppercase hex",
			subnetID: "subnet-ABC12345",
			valid:    false,
		},
		{
			name:     "invalid - non-hex characters",
			subnetID: "subnet-xyz12345",
			valid:    false,
		},
		{
			name:     "invalid - empty",
			subnetID: "",
			valid:    false,
		},
		{
			name:     "invalid - special characters",
			subnetID: "subnet-abc_1234",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSubnetID(tt.subnetID)
			assert.Equal(t, tt.valid, result, "subnet ID: %s", tt.subnetID)
		})
	}
}

func TestMapSubnetToCloudConfig_EmptySubnetIDs(t *testing.T) {
	config, err := MapSubnetToCloudConfig(context.TODO(), aws.Config{}, "arn:aws:iam::123456789012:role/installer", []string{})
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "subnet_ids is required and cannot be empty")
}

func TestMapSubnetToCloudConfig_InvalidSubnetID(t *testing.T) {
	invalidSubnetIDs := []string{"invalid-subnet-id"}
	config, err := MapSubnetToCloudConfig(context.TODO(), aws.Config{}, "arn:aws:iam::123456789012:role/installer", invalidSubnetIDs)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "invalid subnet ID format")
}
