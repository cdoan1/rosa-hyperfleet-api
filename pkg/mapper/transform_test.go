package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOperatorRoles_Success(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"name":      "cloud-credentials",
			"namespace": "openshift-cloud-network",
			"role_arn":  "arn:aws:iam::123456789012:role/test",
		},
		map[string]interface{}{
			"name":      "ebs-cloud-credentials",
			"namespace": "openshift-cluster-csi-drivers",
			"role_arn":  "arn:aws:iam::123456789012:role/test2",
		},
	}

	roles, err := parseOperatorRoles(raw)
	require.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Equal(t, "cloud-credentials", roles[0].Name)
	assert.Equal(t, "openshift-cloud-network", roles[0].Namespace)
	assert.Equal(t, "arn:aws:iam::123456789012:role/test", roles[0].RoleARN)
}

func TestParseOperatorRoles_NotArray(t *testing.T) {
	raw := "not an array"
	roles, err := parseOperatorRoles(raw)
	assert.Error(t, err)
	assert.Nil(t, roles)
	assert.Contains(t, err.Error(), "must be an array")
}

func TestParseOperatorRoles_InvalidElement(t *testing.T) {
	raw := []interface{}{
		"not a map",
	}
	roles, err := parseOperatorRoles(raw)
	assert.Error(t, err)
	assert.Nil(t, roles)
	assert.Contains(t, err.Error(), "must be an object")
}

func TestParseOperatorRoles_MissingFields(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"name":      "cloud-credentials",
			"namespace": "openshift-cloud-network",
			// Missing role_arn
		},
	}
	roles, err := parseOperatorRoles(raw)
	assert.Error(t, err)
	assert.Nil(t, roles)
	assert.Contains(t, err.Error(), "missing required fields")
}

func TestParseSubnetIDs_Success(t *testing.T) {
	raw := []interface{}{
		"subnet-abc123",
		"subnet-def456",
	}

	subnets, err := parseSubnetIDs(raw)
	require.NoError(t, err)
	assert.Equal(t, []string{"subnet-abc123", "subnet-def456"}, subnets)
}

func TestParseSubnetIDs_NotArray(t *testing.T) {
	raw := "not an array"
	subnets, err := parseSubnetIDs(raw)
	assert.Error(t, err)
	assert.Nil(t, subnets)
	assert.Contains(t, err.Error(), "must be an array")
}

func TestParseSubnetIDs_InvalidElement(t *testing.T) {
	raw := []interface{}{
		123, // Not a string
	}
	subnets, err := parseSubnetIDs(raw)
	assert.Error(t, err)
	assert.Nil(t, subnets)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestSetRolesRef(t *testing.T) {
	spec := make(map[string]interface{})
	rolesRef := &AWSRolesRef{
		NetworkARN:              "arn:aws:iam::123456789012:role/network",
		StorageARN:              "arn:aws:iam::123456789012:role/storage",
		ImageRegistryARN:        "arn:aws:iam::123456789012:role/registry",
		KubeCloudControllerARN:  "arn:aws:iam::123456789012:role/kube",
		NodePoolManagementARN:   "arn:aws:iam::123456789012:role/capa",
		ControlPlaneOperatorARN: "arn:aws:iam::123456789012:role/cpo",
		IngressARN:              "arn:aws:iam::123456789012:role/ingress",
	}

	err := setRolesRef(spec, rolesRef)
	require.NoError(t, err)

	// Verify structure
	platform, ok := spec["platform"].(map[string]interface{})
	require.True(t, ok)

	aws, ok := platform["aws"].(map[string]interface{})
	require.True(t, ok)

	rolesRefMap, ok := aws["rolesRef"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "arn:aws:iam::123456789012:role/network", rolesRefMap["networkARN"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/storage", rolesRefMap["storageARN"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/registry", rolesRefMap["imageRegistryARN"])
}

func TestSetRolesRef_ExistingPlatform(t *testing.T) {
	spec := map[string]interface{}{
		"platform": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": "us-east-1",
			},
		},
	}
	rolesRef := &AWSRolesRef{
		NetworkARN: "arn:aws:iam::123456789012:role/network",
		StorageARN: "arn:aws:iam::123456789012:role/storage",
	}

	err := setRolesRef(spec, rolesRef)
	require.NoError(t, err)

	// Verify existing data is preserved
	platform := spec["platform"].(map[string]interface{})
	aws := platform["aws"].(map[string]interface{})
	assert.Equal(t, "us-east-1", aws["region"])
	assert.NotNil(t, aws["rolesRef"])
}

func TestSetCloudProviderConfig(t *testing.T) {
	spec := make(map[string]interface{})
	config := &CloudProviderConfig{
		SubnetID: "subnet-abc123",
		VpcID:    "vpc-xyz789",
		Zone:     "us-east-1a",
	}

	err := setCloudProviderConfig(spec, config)
	require.NoError(t, err)

	// Verify structure
	platform, ok := spec["platform"].(map[string]interface{})
	require.True(t, ok)

	aws, ok := platform["aws"].(map[string]interface{})
	require.True(t, ok)

	cloudConfig, ok := aws["cloudProviderConfig"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "vpc-xyz789", cloudConfig["vpc"])
	assert.Equal(t, "us-east-1a", cloudConfig["zone"])

	subnet, ok := cloudConfig["subnet"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "subnet-abc123", subnet["id"])
}

func TestSetCloudProviderConfig_ExistingPlatform(t *testing.T) {
	spec := map[string]interface{}{
		"platform": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": "us-east-1",
				"rolesRef": map[string]interface{}{
					"networkARN": "arn:aws:iam::123456789012:role/network",
				},
			},
		},
	}
	config := &CloudProviderConfig{
		SubnetID: "subnet-abc123",
		VpcID:    "vpc-xyz789",
		Zone:     "us-east-1a",
	}

	err := setCloudProviderConfig(spec, config)
	require.NoError(t, err)

	// Verify existing data is preserved
	platform := spec["platform"].(map[string]interface{})
	aws := platform["aws"].(map[string]interface{})
	assert.Equal(t, "us-east-1", aws["region"])

	rolesRef := aws["rolesRef"].(map[string]interface{})
	assert.Equal(t, "arn:aws:iam::123456789012:role/network", rolesRef["networkARN"])

	assert.NotNil(t, aws["cloudProviderConfig"])
}

func TestMarshalSpec(t *testing.T) {
	spec := map[string]interface{}{
		"platform": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": "us-east-1",
			},
		},
	}

	result, err := MarshalSpec(spec)
	require.NoError(t, err)
	assert.Contains(t, result, "platform")
	assert.Contains(t, result, "region")
	assert.Contains(t, result, "us-east-1")
}
