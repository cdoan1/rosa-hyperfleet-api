package client

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client) error

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

// WithRegion sets the AWS region for SigV4 signing
func WithRegion(region string) ClientOption {
	return func(c *Client) error {
		c.region = region
		return nil
	}
}

// WithAccountID sets the account ID (added as X-Amz-Account-Id header)
func WithAccountID(accountID string) ClientOption {
	return func(c *Client) error {
		c.accountID = accountID
		return nil
	}
}

// WithCredentials sets the AWS credentials provider
func WithCredentials(credentials aws.CredentialsProvider) ClientOption {
	return func(c *Client) error {
		c.credentials = credentials
		return nil
	}
}

// WithUserID sets the user ID (added as X-Amz-User-Id header)
func WithUserID(userID string) ClientOption {
	return func(c *Client) error {
		c.userID = userID
		return nil
	}
}

// WithCallerARN sets the caller ARN (added as X-Amz-Caller-Arn header)
func WithCallerARN(callerARN string) ClientOption {
	return func(c *Client) error {
		c.callerARN = callerARN
		return nil
	}
}
