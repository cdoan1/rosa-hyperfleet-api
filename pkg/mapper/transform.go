package mapper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TransformClusterSpec transforms the cluster spec from ROSA CLI format
// to HostedCluster format by mapping operator roles and subnet configuration
func TransformClusterSpec(
	ctx context.Context,
	spec map[string]interface{},
	awsConfig aws.Config,
) (map[string]interface{}, error) {
	// Create a copy of the spec to avoid mutating the original
	transformed := make(map[string]interface{})
	for k, v := range spec {
		transformed[k] = v
	}

	// Extract operator_iam_roles if present
	if operatorRolesRaw, exists := transformed["operator_iam_roles"]; exists {
		operatorRoles, err := parseOperatorRoles(operatorRolesRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse operator_iam_roles: %w", err)
		}

		// Map to rolesRef
		rolesRef, err := MapOperatorRolesToRolesRef(operatorRoles)
		if err != nil {
			return nil, fmt.Errorf("failed to map operator roles: %w", err)
		}

		// Add rolesRef to platform.aws section
		if err := setRolesRef(transformed, rolesRef); err != nil {
			return nil, fmt.Errorf("failed to set rolesRef: %w", err)
		}

		// Remove operator_iam_roles from spec as it's been transformed
		delete(transformed, "operator_iam_roles")
	}

	// Extract subnet_ids if present
	if subnetIDsRaw, exists := transformed["subnet_ids"]; exists {
		subnetIDs, err := parseSubnetIDs(subnetIDsRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse subnet_ids: %w", err)
		}

		// Extract installer_role_arn (required for assuming role to describe subnets)
		installerRoleARN, _ := transformed["installer_role_arn"].(string)
		if installerRoleARN == "" {
			return nil, fmt.Errorf("installer_role_arn is required when subnet_ids is provided")
		}

		// Map to cloudProviderConfig
		cloudConfig, err := MapSubnetToCloudConfig(ctx, awsConfig, installerRoleARN, subnetIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to map subnet configuration: %w", err)
		}

		// Add cloudProviderConfig to platform.aws section
		if err := setCloudProviderConfig(transformed, cloudConfig); err != nil {
			return nil, fmt.Errorf("failed to set cloudProviderConfig: %w", err)
		}

		// Keep subnet_ids in the spec for other uses, but also store in platform.aws
	}

	return transformed, nil
}

// parseOperatorRoles extracts operator roles from the raw interface value
func parseOperatorRoles(raw interface{}) ([]OperatorIAMRole, error) {
	rolesArray, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("operator_iam_roles must be an array")
	}

	roles := make([]OperatorIAMRole, 0, len(rolesArray))
	for i, roleRaw := range rolesArray {
		roleMap, ok := roleRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("operator_iam_roles[%d] must be an object", i)
		}

		name, _ := roleMap["name"].(string)
		namespace, _ := roleMap["namespace"].(string)
		roleARN, _ := roleMap["role_arn"].(string)

		if name == "" || namespace == "" || roleARN == "" {
			return nil, fmt.Errorf("operator_iam_roles[%d] missing required fields (name, namespace, role_arn)", i)
		}

		roles = append(roles, OperatorIAMRole{
			Name:      name,
			Namespace: namespace,
			RoleARN:   roleARN,
		})
	}

	return roles, nil
}

// parseSubnetIDs extracts subnet IDs from the raw interface value
func parseSubnetIDs(raw interface{}) ([]string, error) {
	subnetArray, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("subnet_ids must be an array")
	}

	subnets := make([]string, 0, len(subnetArray))
	for i, subnetRaw := range subnetArray {
		subnet, ok := subnetRaw.(string)
		if !ok {
			return nil, fmt.Errorf("subnet_ids[%d] must be a string", i)
		}
		subnets = append(subnets, subnet)
	}

	return subnets, nil
}

// setRolesRef adds rolesRef to the platform.aws section of the spec
func setRolesRef(spec map[string]interface{}, rolesRef *AWSRolesRef) error {
	// Convert rolesRef to map for easier manipulation
	rolesRefMap := map[string]interface{}{
		"networkARN":              rolesRef.NetworkARN,
		"storageARN":              rolesRef.StorageARN,
		"imageRegistryARN":        rolesRef.ImageRegistryARN,
		"kubeCloudControllerARN":  rolesRef.KubeCloudControllerARN,
		"nodePoolManagementARN":   rolesRef.NodePoolManagementARN,
		"controlPlaneOperatorARN": rolesRef.ControlPlaneOperatorARN,
		"ingressARN":              rolesRef.IngressARN,
	}

	// Ensure platform.aws structure exists
	platform, ok := spec["platform"].(map[string]interface{})
	if !ok {
		platform = make(map[string]interface{})
		spec["platform"] = platform
	}

	aws, ok := platform["aws"].(map[string]interface{})
	if !ok {
		aws = make(map[string]interface{})
		platform["aws"] = aws
	}

	// Set rolesRef
	aws["rolesRef"] = rolesRefMap

	return nil
}

// setCloudProviderConfig adds cloudProviderConfig to the platform.aws section of the spec
func setCloudProviderConfig(spec map[string]interface{}, config *CloudProviderConfig) error {
	cloudConfigMap := map[string]interface{}{
		"subnet": map[string]interface{}{
			"id": config.SubnetID,
		},
		"vpc":  config.VpcID,
		"zone": config.Zone,
	}

	// Ensure platform.aws structure exists
	platform, ok := spec["platform"].(map[string]interface{})
	if !ok {
		platform = make(map[string]interface{})
		spec["platform"] = platform
	}

	aws, ok := platform["aws"].(map[string]interface{})
	if !ok {
		aws = make(map[string]interface{})
		platform["aws"] = aws
	}

	// Set cloudProviderConfig
	aws["cloudProviderConfig"] = cloudConfigMap

	return nil
}

// Marshal helper for debugging
func MarshalSpec(spec map[string]interface{}) (string, error) {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
