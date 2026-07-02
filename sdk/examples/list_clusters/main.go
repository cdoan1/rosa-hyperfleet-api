package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/client"
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

	// List all clusters
	clusters, err := c.ListClusters(ctx)
	if err != nil {
		log.Fatalf("Failed to list clusters: %v", err)
	}

	// Display clusters
	fmt.Printf("Found %d clusters:\n\n", len(clusters))

	for i, cluster := range clusters {
		fmt.Printf("%d. Name: %s\n", i+1, cluster.Name)
		fmt.Printf("   ID: %s\n", cluster.Id)
		fmt.Printf("   Created: %s\n", cluster.CreatedAt)
		fmt.Println()
	}
}
