package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cdoan1/rosa-hyperfleet-api-sdk/pkg/client"
	"github.com/cdoan1/rosa-hyperfleet-api-sdk/pkg/types"
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

	// Create cluster
	name := "my-rosa-cluster"

	cluster, err := c.CreateCluster(context.Background(), &types.ClusterCreateRequest{
		Name: name,
		Spec: types.ClusterCreateRequest_Spec{
			AdditionalProperties: map[string]interface{}{
				"provider": "aws",
				"region":   "us-east-1",
				"version":  "4.14.0",
			},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create cluster: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created cluster:\n")
	fmt.Printf("  ID: %s\n", cluster.Id)
	fmt.Printf("  Name: %s\n", cluster.Name)
	if cluster.Status != nil && cluster.Status.Phase != nil {
		fmt.Printf("  Status: %s\n", *cluster.Status.Phase)
	}
}
