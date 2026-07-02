package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Client is the ROSA Hyperfleet API client
type Client struct {
	baseURL     string
	httpClient  *http.Client
	credentials aws.CredentialsProvider
	region      string
	accountID   string
	userID      string
	callerARN   string
}

// NewClient creates a new ROSA Hyperfleet API client
func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {
	// Parse and validate base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Ensure base URL has scheme
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}

	// Default to us-east-1 region
	region := "us-east-1"

	// Load AWS credentials from default chain
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := &Client{
		baseURL:     strings.TrimSuffix(parsedURL.String(), "/"),
		httpClient:  &http.Client{},
		credentials: cfg.Credentials,
		region:      region,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	// Wrap HTTP client with SigV4 signer
	client.httpClient.Transport = newSigV4RoundTripper(
		client.httpClient.Transport,
		client.credentials,
		client.region,
		client.accountID,
	)

	return client, nil
}

// doRequest executes an HTTP request and handles the response
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Build full URL
	fullURL := c.baseURL + path

	// Encode request body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for error response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseErrorResponse(resp)
	}

	// Decode response body
	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
