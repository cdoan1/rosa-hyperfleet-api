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
		baseURL = "https://api.rosa-hyperfleet.example.com"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	accountID := os.Getenv("AWS_ACCOUNT_ID")

	// Create API client with AWS SigV4 authentication
	c, err := client.NewClient(
		baseURL,
		client.WithRegion(region),
		client.WithAccountID(accountID),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create cluster request
	clusterName := "example-cluster"
	req := &types.ClusterCreateRequest{
		Name: clusterName,
		Spec: types.ClusterCreateRequest_Spec{
			// Add cluster spec fields here based on your requirements
			// This is a minimal example
		},
	}

	// Create the cluster
	cluster, err := c.CreateCluster(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create cluster: %v", err)
	}

	fmt.Printf("Cluster created successfully!\n")
	fmt.Printf("Name: %s\n", cluster.Name)
	fmt.Printf("ID: %s\n", cluster.Id)
}
