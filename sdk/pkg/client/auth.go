package client

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// sigV4RoundTripper wraps an http.RoundTripper and signs requests with AWS SigV4
type sigV4RoundTripper struct {
	transport   http.RoundTripper
	credentials aws.CredentialsProvider
	region      string
	accountID   string
	userID      string
	callerARN   string
}

// RoundTrip executes a single HTTP transaction, signing the request with AWS SigV4
func (s *sigV4RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = req.Clone(req.Context())

	// Add custom headers
	if s.accountID != "" {
		req.Header.Set("X-Amz-Account-Id", s.accountID)
	}
	if s.userID != "" {
		req.Header.Set("X-Amz-User-Id", s.userID)
	}
	if s.callerARN != "" {
		req.Header.Set("X-Amz-Caller-Arn", s.callerARN)
	}

	// Get AWS credentials
	creds, err := s.credentials.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Compute payload hash
	var payloadHash string
	if req.Body != nil {
		// Read the body for hashing
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()

		// Compute SHA256 hash
		hash := sha256.Sum256(bodyBytes)
		payloadHash = fmt.Sprintf("%x", hash)

		// Restore the body
		req.Body = io.NopCloser(io.Reader(io.MultiReader(
			io.NopCloser(io.Reader(&bodyReader{data: bodyBytes})),
		)))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(&bodyReader{data: bodyBytes}), nil
		}
		req.ContentLength = int64(len(bodyBytes))
	} else {
		// Empty payload hash
		hash := sha256.Sum256([]byte{})
		payloadHash = fmt.Sprintf("%x", hash)
	}

	// Create SigV4 signer
	signer := v4.NewSigner()

	// Sign the request
	// Use "execute-api" as the service name for API Gateway
	err = signer.SignHTTP(req.Context(), creds, req, payloadHash, "execute-api", s.region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Execute the request
	return s.transport.RoundTrip(req)
}

// bodyReader is a simple io.Reader for reading body bytes
type bodyReader struct {
	data []byte
	pos  int
}

func (b *bodyReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

// newSigV4RoundTripper creates a new SigV4 signing round tripper
func newSigV4RoundTripper(transport http.RoundTripper, credentials aws.CredentialsProvider, region, accountID string) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &sigV4RoundTripper{
		transport:   transport,
		credentials: credentials,
		region:      region,
		accountID:   accountID,
	}
}
