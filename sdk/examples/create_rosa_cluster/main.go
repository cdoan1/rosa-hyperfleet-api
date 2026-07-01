package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/client"
	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

func main() {
	// Configuration from environment variables
	baseURL := os.Getenv("ROSA_API_URL")
	if baseURL == "" {
		baseURL = "https://api.us-east-1.rosa.example.com"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	accountID := os.Getenv("AWS_ACCOUNT_ID")
	if accountID == "" {
		fmt.Fprintf(os.Stderr, "AWS_ACCOUNT_ID environment variable required\n")
		os.Exit(1)
	}

	// Create SDK client
	c, err := client.NewClient(
		client.WithBaseURL(baseURL),
		client.WithRegion(region),
		client.WithAccountID(accountID),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// Create cluster using ROSA CLI format
	// The API will transform operator_iam_roles to platform.aws.rolesRef
	// and query AWS to populate platform.aws.cloudProviderConfig from subnet_ids
	name := "my-rosa-hcp-cluster"
	operatorRolePrefix := "my-cluster-op"

	operatorIamRoles := []struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		RoleArn   string `json:"role_arn"`
	}{
		{
			Name:      "cloud-network-config-controller-cloud-credentials",
			Namespace: "openshift-cloud-network-config-controller",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-openshift-cloud-network-config-controller-cloud-network-config-controller-cloud-credentials", accountID, operatorRolePrefix),
		},
		{
			Name:      "ebs-cloud-credentials",
			Namespace: "openshift-cluster-csi-drivers",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-openshift-cluster-csi-drivers-ebs-cloud-credentials", accountID, operatorRolePrefix),
		},
		{
			Name:      "installer-cloud-credentials",
			Namespace: "openshift-image-registry",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-openshift-image-registry-installer-cloud-credentials", accountID, operatorRolePrefix),
		},
		{
			Name:      "kube-controller-manager",
			Namespace: "kube-system",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-kube-system-kube-controller-manager", accountID, operatorRolePrefix),
		},
		{
			Name:      "capa-controller-manager",
			Namespace: "kube-system",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-kube-system-capa-controller-manager", accountID, operatorRolePrefix),
		},
		{
			Name:      "control-plane-operator",
			Namespace: "kube-system",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-kube-system-control-plane-operator", accountID, operatorRolePrefix),
		},
		{
			Name:      "cloud-credentials",
			Namespace: "openshift-ingress-operator",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-openshift-ingress-operator-cloud-credentials", accountID, operatorRolePrefix),
		},
		{
			Name:      "kms-provider",
			Namespace: "kube-system",
			RoleArn:   fmt.Sprintf("arn:aws:iam::%s:role/rosa/%s-kube-system-kms-provider", accountID, operatorRolePrefix),
		},
	}

	installerRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/rosa/ManagedOpenShift-HCP-ROSA-Installer-Role", accountID)
	supportRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/rosa/ManagedOpenShift-HCP-ROSA-Support-Role", accountID)
	oidcConfigID := "2r8p7g9fogag2lg31hph07larks4jpql" // Replace with actual OIDC config ID

	// Subnet IDs - replace with your actual subnet IDs
	subnetIDs := os.Getenv("SUBNET_IDS")
	if subnetIDs == "" {
		fmt.Fprintf(os.Stderr, "SUBNET_IDS environment variable required (comma-separated list)\n")
		os.Exit(1)
	}

	cluster, err := c.CreateCluster(context.Background(), &types.ClusterCreateRequest{
		Name: name,
		Spec: types.ClusterCreateRequest_Spec{
			OperatorIamRoles: &operatorIamRoles,
			InstallerRoleArn: &installerRoleArn,
			SupportRoleArn:   &supportRoleArn,
			OidcConfigId:     &oidcConfigID,
			AdditionalProperties: map[string]interface{}{
				"region":     region,
				"subnet_ids": []string{"subnet-0a1b2c3d4e5f6789", "subnet-9k8l7m6n5o4p3q2r"}, // Replace with actual
			},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create cluster: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created ROSA HCP cluster:\n")
	fmt.Printf("  ID: %s\n", cluster.Id)
	fmt.Printf("  Name: %s\n", cluster.Name)
	if cluster.Status != nil && cluster.Status.Phase != nil {
		fmt.Printf("  Status: %s\n", *cluster.Status.Phase)
	}
	fmt.Printf("\nThe API has transformed:\n")
	fmt.Printf("  - operator_iam_roles → platform.aws.rolesRef\n")
	fmt.Printf("  - subnet_ids → platform.aws.cloudProviderConfig (with VPC and AZ from AWS)\n")
}
