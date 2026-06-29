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
	principalARN := os.Getenv("PRINCIPAL_ARN")
	if principalARN == "" {
		fmt.Fprintf(os.Stderr, "PRINCIPAL_ARN environment variable required\n")
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

	// Check authorization
	action := "rosa:clusters:create"
	resource := "arn:aws:rosa:us-east-1:123456789012:cluster/*"

	response, err := c.CheckAuthorization(context.Background(), &types.CheckAuthorizationRequest{
		Action:    action,
		Principal: principalARN,
		Resource:  resource,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check authorization: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Authorization Check:\n")
	fmt.Printf("  Principal: %s\n", principalARN)
	fmt.Printf("  Action: %s\n", action)
	fmt.Printf("  Resource: %s\n", resource)
	fmt.Printf("  Decision: %s\n", response.Decision)
	if response.Reason != nil && *response.Reason != "" {
		fmt.Printf("  Reason: %s\n", *response.Reason)
	}
}
