package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapOperatorRolesToRolesRef_Success(t *testing.T) {
	operatorRoles := []OperatorIAMRole{
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-cloud-network",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cloud-network-cloud-credentials",
		},
		{
			Name:      "ebs-cloud-credentials",
			Namespace: "openshift-cluster-csi-drivers",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cluster-csi-drivers-ebs-cloud-credentials",
		},
		{
			Name:      "cloud-network-config-controller-cloud-credentials",
			Namespace: "openshift-cloud-network-config-controller",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cloud-network-config-controller-cloud-network-config-controller-cloud-credentials",
		},
		{
			Name:      "kube-controller-manager",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-kube-controller-manager",
		},
		{
			Name:      "capa-controller-manager",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-capa-controller-manager",
		},
		{
			Name:      "control-plane-operator",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-control-plane-operator",
		},
		{
			Name:      "ingress-operator-cloud-credentials",
			Namespace: "openshift-ingress-operator",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-openshift-ingress-operator-ingress-operator-cloud-credentials",
		},
		{
			Name:      "kms-provider",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-kms-provider",
		},
	}

	rolesRef, err := MapOperatorRolesToRolesRef(operatorRoles)
	require.NoError(t, err)
	require.NotNil(t, rolesRef)

	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cloud-network-cloud-credentials", rolesRef.NetworkARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cluster-csi-drivers-ebs-cloud-credentials", rolesRef.StorageARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cloud-network-config-controller-cloud-network-config-controller-cloud-credentials", rolesRef.ImageRegistryARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-kube-controller-manager", rolesRef.KubeCloudControllerARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-capa-controller-manager", rolesRef.NodePoolManagementARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-kube-system-control-plane-operator", rolesRef.ControlPlaneOperatorARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/rosa/test-op-openshift-ingress-operator-ingress-operator-cloud-credentials", rolesRef.IngressARN)
}

func TestMapOperatorRolesToRolesRef_MissingRole(t *testing.T) {
	// Missing ingress role
	operatorRoles := []OperatorIAMRole{
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-cloud-network",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-network",
		},
		{
			Name:      "ebs-cloud-credentials",
			Namespace: "openshift-cluster-csi-drivers",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-storage",
		},
		{
			Name:      "cloud-network-config-controller-cloud-credentials",
			Namespace: "openshift-cloud-network-config-controller",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-registry",
		},
		{
			Name:      "kube-controller-manager",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-kube",
		},
		{
			Name:      "capa-controller-manager",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-capa",
		},
		{
			Name:      "control-plane-operator",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-cpo",
		},
		// Missing ingress role
	}

	rolesRef, err := MapOperatorRolesToRolesRef(operatorRoles)
	assert.Error(t, err)
	assert.Nil(t, rolesRef)
	assert.Contains(t, err.Error(), "missing required operator role: ingress")
}

func TestMapOperatorRolesToRolesRef_DuplicateRole(t *testing.T) {
	operatorRoles := []OperatorIAMRole{
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-cloud-network",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-network-1",
		},
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-cloud-network",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-network-2",
		},
	}

	rolesRef, err := MapOperatorRolesToRolesRef(operatorRoles)
	assert.Error(t, err)
	assert.Nil(t, rolesRef)
	assert.Contains(t, err.Error(), "duplicate network role")
}

func TestMapOperatorRolesToRolesRef_InvalidARN(t *testing.T) {
	operatorRoles := []OperatorIAMRole{
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-cloud-network",
			RoleARN:   "invalid-arn-format",
		},
	}

	rolesRef, err := MapOperatorRolesToRolesRef(operatorRoles)
	assert.Error(t, err)
	assert.Nil(t, rolesRef)
	assert.Contains(t, err.Error(), "invalid role ARN format")
}

func TestIsValidARN(t *testing.T) {
	tests := []struct {
		name  string
		arn   string
		valid bool
	}{
		{
			name:  "valid ARN",
			arn:   "arn:aws:iam::123456789012:role/rosa/test-role",
			valid: true,
		},
		{
			name:  "valid ARN with path",
			arn:   "arn:aws:iam::123456789012:role/path/to/role",
			valid: true,
		},
		{
			name:  "invalid - missing arn prefix",
			arn:   "aws:iam::123456789012:role/test",
			valid: false,
		},
		{
			name:  "invalid - wrong account format",
			arn:   "arn:aws:iam::12345:role/test",
			valid: false,
		},
		{
			name:  "invalid - not iam",
			arn:   "arn:aws:s3:::bucket-name",
			valid: false,
		},
		{
			name:  "invalid - empty",
			arn:   "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidARN(tt.arn)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestMapOperatorRolesToRolesRef_KMSProviderIgnored(t *testing.T) {
	// Test that KMS provider role is properly ignored and doesn't cause errors
	operatorRoles := []OperatorIAMRole{
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-cloud-network",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-network",
		},
		{
			Name:      "ebs-cloud-credentials",
			Namespace: "openshift-cluster-csi-drivers",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-storage",
		},
		{
			Name:      "cloud-network-config-controller-cloud-credentials",
			Namespace: "openshift-cloud-network-config-controller",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-registry",
		},
		{
			Name:      "kube-controller-manager",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-kube",
		},
		{
			Name:      "capa-controller-manager",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-capa",
		},
		{
			Name:      "control-plane-operator",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-cpo",
		},
		{
			Name:      "ingress-operator-cloud-credentials",
			Namespace: "openshift-ingress-operator",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-ingress",
		},
		{
			Name:      "kms-provider",
			Namespace: "kube-system",
			RoleARN:   "arn:aws:iam::123456789012:role/rosa/test-kms",
		},
	}

	rolesRef, err := MapOperatorRolesToRolesRef(operatorRoles)
	require.NoError(t, err)
	require.NotNil(t, rolesRef)

	// Verify all 7 required roles are present
	assert.NotEmpty(t, rolesRef.NetworkARN)
	assert.NotEmpty(t, rolesRef.StorageARN)
	assert.NotEmpty(t, rolesRef.ImageRegistryARN)
	assert.NotEmpty(t, rolesRef.KubeCloudControllerARN)
	assert.NotEmpty(t, rolesRef.NodePoolManagementARN)
	assert.NotEmpty(t, rolesRef.ControlPlaneOperatorARN)
	assert.NotEmpty(t, rolesRef.IngressARN)
}
