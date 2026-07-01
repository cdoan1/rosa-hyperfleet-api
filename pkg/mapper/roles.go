package mapper

import (
	"fmt"
	"regexp"
)

var arnPattern = regexp.MustCompile(`^arn:aws:iam::\d{12}:role/.+$`)

// MapOperatorRolesToRolesRef converts the operator_iam_roles array from ROSA CLI
// into the rolesRef structure required by the HostedCluster CR
func MapOperatorRolesToRolesRef(operatorRoles []OperatorIAMRole) (*AWSRolesRef, error) {
	rolesRef := &AWSRolesRef{}

	// Track which roles we've found
	foundRoles := make(map[string]bool)

	for _, role := range operatorRoles {
		key := fmt.Sprintf("%s/%s", role.Namespace, role.Name)

		// Validate role ARN format
		if !isValidARN(role.RoleARN) {
			return nil, fmt.Errorf("invalid role ARN format for %s: %s", key, role.RoleARN)
		}

		switch {
		// CORRECTED: networkARN comes from cloud-network-config-controller, NOT openshift-cloud-network
		case role.Namespace == "openshift-cloud-network-config-controller" &&
			role.Name == "cloud-network-config-controller-cloud-credentials":
			if foundRoles["network"] {
				return nil, fmt.Errorf("duplicate network role found: %s", key)
			}
			rolesRef.NetworkARN = role.RoleARN
			foundRoles["network"] = true

		case role.Namespace == "openshift-cluster-csi-drivers" && role.Name == "ebs-cloud-credentials":
			if foundRoles["storage"] {
				return nil, fmt.Errorf("duplicate storage role found: %s", key)
			}
			rolesRef.StorageARN = role.RoleARN
			foundRoles["storage"] = true

		// CORRECTED: imageRegistryARN comes from openshift-image-registry, NOT cloud-network-config-controller
		case role.Namespace == "openshift-image-registry" &&
			role.Name == "installer-cloud-credentials":
			if foundRoles["imageRegistry"] {
				return nil, fmt.Errorf("duplicate imageRegistry role found: %s", key)
			}
			rolesRef.ImageRegistryARN = role.RoleARN
			foundRoles["imageRegistry"] = true

		case role.Namespace == "kube-system" && role.Name == "kube-controller-manager":
			if foundRoles["kubeCloudController"] {
				return nil, fmt.Errorf("duplicate kubeCloudController role found: %s", key)
			}
			rolesRef.KubeCloudControllerARN = role.RoleARN
			foundRoles["kubeCloudController"] = true

		case role.Namespace == "kube-system" && role.Name == "capa-controller-manager":
			if foundRoles["nodePoolManagement"] {
				return nil, fmt.Errorf("duplicate nodePoolManagement role found: %s", key)
			}
			rolesRef.NodePoolManagementARN = role.RoleARN
			foundRoles["nodePoolManagement"] = true

		case role.Namespace == "kube-system" && role.Name == "control-plane-operator":
			if foundRoles["controlPlaneOperator"] {
				return nil, fmt.Errorf("duplicate controlPlaneOperator role found: %s", key)
			}
			rolesRef.ControlPlaneOperatorARN = role.RoleARN
			foundRoles["controlPlaneOperator"] = true

		// CORRECTED: ingressARN uses 'cloud-credentials' name, NOT 'ingress-operator-cloud-credentials'
		case role.Namespace == "openshift-ingress-operator" &&
			role.Name == "cloud-credentials":
			if foundRoles["ingress"] {
				return nil, fmt.Errorf("duplicate ingress role found: %s", key)
			}
			rolesRef.IngressARN = role.RoleARN
			foundRoles["ingress"] = true

		case role.Namespace == "kube-system" && role.Name == "kms-provider":
			// KMS provider is not part of rolesRef
			// It will be used elsewhere in the HC spec for encryption configuration
			continue
		}
	}

	// Validate all required roles were found
	requiredRoles := []string{
		"network", "storage", "imageRegistry", "kubeCloudController",
		"nodePoolManagement", "controlPlaneOperator", "ingress",
	}

	for _, required := range requiredRoles {
		if !foundRoles[required] {
			// Build a helpful error message showing what was received
			var receivedRoles []string
			for _, role := range operatorRoles {
				receivedRoles = append(receivedRoles, fmt.Sprintf("%s/%s", role.Namespace, role.Name))
			}
			return nil, fmt.Errorf("missing required operator role: %s (received %d roles: %v)",
				required, len(operatorRoles), receivedRoles)
		}
	}

	return rolesRef, nil
}

// isValidARN validates AWS IAM role ARN format
func isValidARN(arn string) bool {
	return arnPattern.MatchString(arn)
}
