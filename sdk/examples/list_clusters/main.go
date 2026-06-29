package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/client"
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

	// List clusters
	clusters, err := c.ListClusters(context.Background(), &client.ListClustersOptions{
		Limit: 50,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list clusters: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d clusters (showing %d):\n\n", clusters.Total, len(clusters.Items))
	for _, cluster := range clusters.Items {
		fmt.Printf("- ID: %s\n", cluster.Id)
		fmt.Printf("  Name: %s\n", cluster.Name)
		if cluster.Status != nil && cluster.Status.Phase != nil {
			fmt.Printf("  Status: %s\n", *cluster.Status.Phase)
		}
		fmt.Println()
	}
}
