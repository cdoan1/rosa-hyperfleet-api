package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/client"
	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

func main() {
	ctx := context.Background()

	// Get configuration from environment
	baseURL := os.Getenv("ROSA_HYPERFLEET_API_URL")
	if baseURL == "" {
		log.Fatal("ROSA_HYPERFLEET_API_URL environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	accountID := os.Getenv("AWS_ACCOUNT_ID")
	if accountID == "" {
		log.Fatal("AWS_ACCOUNT_ID environment variable is required")
	}

	// Create API client with AWS SigV4 authentication
	c, err := client.NewClient(
		baseURL,
		client.WithRegion(region),
		client.WithAccountID(accountID),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Example: Create a ROSA HCP cluster
	// This demonstrates the structure for a ROSA hosted control plane cluster
	clusterName := "rosa-hcp-example"
	versionStr := "4.14.0"

	req := &types.ClusterCreateRequest{
		Name: clusterName,
		Spec: types.ClusterCreateRequest_Spec{
			// Version information
			AdditionalProperties: map[string]interface{}{
				"version": versionStr,
				"region":  region,
			},
		},
	}

	// Create the cluster
	cluster, err := c.CreateCluster(ctx, req)
	if err != nil {
		if client.IsForbidden(err) {
			log.Fatalf("Access denied. Check your AWS credentials and permissions: %v", err)
		} else if client.IsBadRequest(err) {
			log.Fatalf("Invalid cluster specification: %v", err)
		}
		log.Fatalf("Failed to create cluster: %v", err)
	}

	fmt.Printf("ROSA HCP Cluster created successfully!\n")
	fmt.Printf("Name: %s\n", cluster.Name)
	fmt.Printf("ID: %s\n", cluster.Id)
	fmt.Printf("Created at: %s\n", cluster.CreatedAt)

	// You can poll for cluster status
	fmt.Println("\nPolling for cluster status...")
	status, err := c.GetClusterStatus(ctx, cluster.Id)
	if err != nil {
		log.Printf("Warning: Failed to get cluster status: %v", err)
	} else {
		fmt.Printf("Cluster Status: %+v\n", status)
	}
}
