# ROSA Regional Platform API Go SDK

Official Go SDK for the ROSA Regional Platform API. This SDK provides a type-safe, idiomatic Go interface for managing ROSA hosted clusters, nodepools, authorization, and other platform resources.

## Installation

```bash
go get github.com/openshift-online/rosa-hyperfleet-api-sdk@latest
```

## Features

- **Type-safe API**: Generated types from OpenAPI specification
- **AWS SigV4 Authentication**: Automatic request signing for API Gateway
- **Comprehensive Coverage**: Support for all platform resources:
  - Clusters (create, list, get, update, delete, status)
  - NodePools (CRUD operations and status)
  - Management Clusters (create, list, get)
  - Authorization (policies, groups, attachments, admins)
  - Accounts (enable, list, get, disable)
  - Trusted Actions/ZOA (run, list, describe, audit)
- **Context-aware**: All operations support context.Context for cancellation and timeouts
- **Error handling**: Strongly-typed API errors with helper functions

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/openshift-online/rosa-hyperfleet-api-sdk/pkg/client"
    "github.com/openshift-online/rosa-hyperfleet-api-sdk/pkg/types"
)

func main() {
    // Create client
    c, err := client.NewClient(
        client.WithRegion("us-east-1"),
        client.WithBaseURL("https://xyz.execute-api.us-east-1.amazonaws.com/prod"),
        client.WithAccountID("123456789012"),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Create a cluster
    cluster, err := c.CreateCluster(context.Background(), &types.ClusterCreateRequest{
        Name: "my-rosa-cluster",
        Spec: types.ClusterCreateRequest_Spec{
            AdditionalProperties: map[string]interface{}{
                "provider": "aws",
                "region":   "us-east-1",
                "version":  "4.14.0",
            },
        },
    })
    if err != nil {
        log.Fatalf("Failed to create cluster: %v", err)
    }

    fmt.Printf("Created cluster: %s (ID: %s)\n", cluster.Name, cluster.Id)
}
```

## Authentication

The SDK uses AWS Signature Version 4 (SigV4) to authenticate requests to the API Gateway. AWS credentials are automatically loaded from:

1. **Environment variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
2. **Shared credentials file**: `~/.aws/credentials`
3. **IAM role**: When running on EC2, ECS, Lambda, or other AWS services

### Using a specific AWS profile

```go
import (
    "context"
    
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/openshift-online/rosa-hyperfleet-api-sdk/pkg/client"
)

cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile("my-profile"))
if err != nil {
    log.Fatal(err)
}

c, err := client.NewClient(
    client.WithAWSConfig(cfg),
    client.WithBaseURL("https://api.example.com"),
)
```

## Usage Examples

### Cluster Management

#### List clusters

```go
clusters, err := c.ListClusters(ctx, &client.ListClustersOptions{
    Limit:  50,
    Offset: 0,
    Status: "Ready",
})
if err != nil {
    log.Fatal(err)
}

for _, cluster := range clusters.Items {
    fmt.Printf("- %s (Status: %s)\n", cluster.Name, cluster.Status)
}
```

#### Get cluster details

```go
cluster, err := c.GetCluster(ctx, "cluster-123")
if err != nil {
    if client.IsNotFound(err) {
        fmt.Println("Cluster not found")
    } else {
        log.Fatal(err)
    }
}
```

#### Update cluster

```go
cluster, err := c.UpdateCluster(ctx, "cluster-123", &types.ClusterUpdateRequest{
    Spec: types.ClusterUpdateRequest_Spec{
        AdditionalProperties: map[string]interface{}{
            "replicas": 3,
        },
    },
})
```

#### Delete cluster

```go
err := c.DeleteCluster(ctx, "cluster-123", &client.DeleteClusterOptions{
    Force: false,
})
```

#### Get cluster status

```go
status, err := c.GetClusterStatus(ctx, "cluster-123")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cluster status: %s\n", status.Status)
```

### NodePool Management

#### Create nodepool

```go
replicas := int64(3)
nodepool, err := c.CreateNodePool(ctx, &types.NodePoolCreateRequest{
    ClusterId: "cluster-123",
    Name:      "worker-nodes",
    Spec: &types.NodePoolCreateRequest_Spec{
        Replicas: &replicas,
    },
})
```

#### List nodepools for a cluster

```go
nodepools, err := c.ListNodePools(ctx, &client.ListNodePoolsOptions{
    ClusterID: "cluster-123",
})
```

### Authorization

#### Check authorization

```go
result, err := c.CheckAuthorization(ctx, &types.CheckAuthorizationRequest{
    Principal: "arn:aws:iam::123456789012:user/alice",
    Action:    "rosa:clusters:create",
    Resource:  "arn:aws:rosa:us-east-1:123456789012:cluster/*",
})

if result.Decision == "ALLOW" {
    fmt.Println("Authorized")
} else {
    fmt.Printf("Denied: %s\n", *result.Reason)
}
```

#### Create authorization policy

```go
policy, err := c.CreatePolicy(ctx, &types.CreatePolicyRequest{
    Name:        "cluster-admins",
    Description: strPtr("Allow cluster creation"),
    Policy:      strPtr("permit(principal in Group::\"admins\", action == Action::\"rosa:clusters:create\", resource);"),
})
```

### Account Management

#### Enable account

```go
privileged := true
account, err := c.EnableAccount(ctx, &types.EnableAccountRequest{
    AccountId:  "123456789012",
    Privileged: &privileged,
})
```

### Trusted Actions (ZOA)

#### Run a trusted action

```go
result, err := c.RunTrustedAction(ctx, "upgrade-cluster", &client.TrustedActionRequest{
    ClusterID: "cluster-123",
    Params: map[string]interface{}{
        "version": "4.15.0",
    },
})

fmt.Printf("Execution ID: %s\n", result.ExecutionID)
```

## Error Handling

The SDK provides helper functions to check for specific error types:

```go
cluster, err := c.GetCluster(ctx, "cluster-123")
if err != nil {
    switch {
    case client.IsNotFound(err):
        fmt.Println("Cluster not found")
    case client.IsForbidden(err):
        fmt.Println("Access denied")
    case client.IsUnauthorized(err):
        fmt.Println("Authentication failed")
    case client.IsBadRequest(err):
        fmt.Println("Invalid request")
    default:
        log.Fatalf("API error: %v", err)
    }
}
```

## Configuration Options

### Client Configuration

```go
c, err := client.NewClient(
    // Required: API Gateway base URL
    client.WithBaseURL("https://xyz.execute-api.us-east-1.amazonaws.com/prod"),
    
    // Required: AWS region for SigV4 signing
    client.WithRegion("us-east-1"),
    
    // Optional: AWS account ID (sent in X-Amz-Account-Id header)
    client.WithAccountID("123456789012"),
    
    // Optional: Custom AWS config
    client.WithAWSConfig(awsConfig),
    
    // Optional: Custom HTTP client
    client.WithHTTPClient(&http.Client{
        Timeout: 60 * time.Second,
    }),
    
    // Optional: Request timeout
    client.WithTimeout(30 * time.Second),
    
    // Optional: Custom user agent
    client.WithUserAgent("my-app/1.0.0"),
)
```

## Examples

Complete working examples are available in the [`examples/`](./examples/) directory:

- [`create_cluster`](./examples/create_cluster/) - Create a new cluster
- [`list_clusters`](./examples/list_clusters/) - List all clusters
- [`authz_check`](./examples/authz_check/) - Check authorization

## Development

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Generating Types

Types are auto-generated from the OpenAPI specification:

```bash
make generate
```

### Linting

```bash
make lint
```

## Versioning

This SDK follows semantic versioning. Git tags use the `sdk/vX.Y.Z` format to allow independent versioning from the API server.

## License

Apache 2.0

## Contributing

Contributions are welcome! Please open an issue or pull request.
