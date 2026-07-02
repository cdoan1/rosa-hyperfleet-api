# ROSA Hyperfleet API Go SDK

Official Go SDK for the ROSA Hyperfleet API (formerly Regional Platform API).

## Installation

```bash
go get github.com/openshift-online/rosa-hyperfleet-api/sdk@latest
```

### Testing from Branch

During the testing phase, install from the development branch:

```bash
go get github.com/cdoan1/rosa-hyperfleet-api/sdk@ROSAENG-60659.2
```

Or use a replace directive in your `go.mod`:

```go
replace github.com/openshift-online/rosa-hyperfleet-api/sdk => github.com/cdoan1/rosa-hyperfleet-api/sdk ROSAENG-60659.2
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/client"
)

func main() {
	ctx := context.Background()

	// Create client with AWS SigV4 authentication
	c, err := client.NewClient(
		"https://api.rosa-hyperfleet.example.com",
		client.WithRegion("us-east-1"),
		client.WithAccountID("123456789012"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// List clusters
	clusters, err := c.ListClusters(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, cluster := range clusters {
		fmt.Printf("Cluster: %s (ID: %s)\n", cluster.Name, cluster.ID)
	}
}
```

## Features

- **Type-safe API client** generated from OpenAPI specification
- **AWS SigV4 authentication** for API Gateway
- **Support for all platform resources**:
  - Clusters and NodePools
  - Management Clusters
  - Authorization (accounts, policies, groups, attachments)
  - Trusted Actions (ZOA)
- **Context-aware operations** for cancellation and timeouts
- **Comprehensive error handling** with typed error helpers

## Authentication

The SDK uses AWS SigV4 signing to authenticate requests to the ROSA Hyperfleet API Gateway. AWS credentials are loaded automatically using the standard AWS SDK credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (when running on EC2/ECS/Lambda)

### Custom Credentials

```go
import (
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/client"
)

c, err := client.NewClient(
	"https://api.rosa-hyperfleet.example.com",
	client.WithCredentials(credentials.NewStaticCredentialsProvider(
		"ACCESS_KEY_ID",
		"SECRET_ACCESS_KEY",
		"SESSION_TOKEN",
	)),
)
```

## Configuration Options

```go
// Base URL
client.WithBaseURL("https://api.example.com")

// AWS Region
client.WithRegion("us-west-2")

// Account ID (sets X-Amz-Account-Id header)
client.WithAccountID("123456789012")

// Custom HTTP client
client.WithHTTPClient(customHTTPClient)

// AWS credentials provider
client.WithCredentials(credsProvider)
```

## Usage Examples

### Cluster Management

```go
// Create cluster
cluster, err := c.CreateCluster(ctx, &types.ClusterCreateRequest{
	Name: "my-cluster",
	Spec: types.ClusterCreateRequest_Spec{
		// ... cluster spec
	},
})

// Get cluster
cluster, err := c.GetCluster(ctx, clusterID)

// List clusters
clusters, err := c.ListClusters(ctx)

// Delete cluster
err := c.DeleteCluster(ctx, clusterID)
```

### NodePool Management

```go
// Create nodepool
nodepool, err := c.CreateNodePool(ctx, &types.NodePoolCreateRequest{
	ClusterId: clusterID,
	Spec: types.NodePoolCreateRequest_Spec{
		// ... nodepool spec
	},
})

// List nodepools
nodepools, err := c.ListNodePools(ctx)
```

### Authorization

```go
// Check authorization
result, err := c.CheckAuthorization(ctx, &types.AuthzCheckRequest{
	Principal: "arn:aws:iam::123456789012:user/alice",
	Action:    "cluster:create",
	Resource:  "arn:rosa:cluster:*",
})

if result.Decision == "allow" {
	fmt.Println("Authorized")
}
```

### Error Handling

```go
cluster, err := c.GetCluster(ctx, clusterID)
if err != nil {
	if client.IsNotFound(err) {
		fmt.Println("Cluster not found")
	} else if client.IsForbidden(err) {
		fmt.Println("Access denied")
	} else {
		log.Fatal(err)
	}
}
```

## Examples

See the `examples/` directory for complete working examples:

- [`create_cluster/`](examples/create_cluster/) - Create a cluster
- [`list_clusters/`](examples/list_clusters/) - List all clusters
- [`create_rosa_cluster/`](examples/create_rosa_cluster/) - Create a ROSA HCP cluster

## Development

### Prerequisites

```bash
# Install oapi-codegen
make install-tools
```

### Building

```bash
# Generate types from OpenAPI spec
make generate

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint
```

### Regenerating Types

When the OpenAPI specification changes:

```bash
cd sdk
make generate
```

This regenerates `pkg/types/generated.go` from `../openapi/openapi.yaml`.

## Versioning

The SDK uses semantic versioning with git tags in the format `sdk/vX.Y.Z`. This allows the SDK to be versioned independently from the API server.

Example:
- SDK v0.1.0: `sdk/v0.1.0`
- SDK v0.2.0: `sdk/v0.2.0`

## License

Apache License 2.0

## Support

For issues and questions:
- GitHub Issues: https://github.com/openshift-online/rosa-hyperfleet-api/issues
- Documentation: https://github.com/openshift-online/rosa-hyperfleet-api/tree/main/docs
