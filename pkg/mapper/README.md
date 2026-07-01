# ROSA CLI to HostedCluster Mapper

This package implements the transformation logic required to convert ROSA CLI cluster specifications into HostedCluster Custom Resource format.

## Overview

The ROSA CLI sends cluster configuration data with `operator_iam_roles` and `subnet_ids` fields that must be mapped to the HostedCluster CR format expected by HyperShift. This mapper package performs that transformation.

## Components

### Types (`types.go`)

- `OperatorIAMRole`: Represents an operator IAM role from ROSA CLI
- `AWSRolesRef`: Represents the `rolesRef` structure in HostedCluster CR
- `CloudProviderConfig`: Represents the `cloudProviderConfig` structure in HostedCluster CR

### Role Mapping (`roles.go`)

Maps the `operator_iam_roles` array to the `platform.aws.rolesRef` structure.

#### Mapping Table

| Namespace | Role Name | HostedCluster Field |
|-----------|-----------|---------------------|
| `openshift-cloud-network` | `cloud-credentials` | `rolesRef.networkARN` |
| `openshift-cluster-csi-drivers` | `ebs-cloud-credentials` | `rolesRef.storageARN` |
| `openshift-cloud-network-config-controller` | `cloud-network-config-controller-cloud-credentials` | `rolesRef.imageRegistryARN` |
| `kube-system` | `kube-controller-manager` | `rolesRef.kubeCloudControllerARN` |
| `kube-system` | `capa-controller-manager` | `rolesRef.nodePoolManagementARN` |
| `kube-system` | `control-plane-operator` | `rolesRef.controlPlaneOperatorARN` |
| `openshift-ingress-operator` | `ingress-operator-cloud-credentials` | `rolesRef.ingressARN` |
| `kube-system` | `kms-provider` | *(Not mapped to rolesRef)* |

#### Validation

- All 7 required roles must be present
- Role ARNs must match pattern: `arn:aws:iam::\d{12}:role/.*`
- No duplicate roles allowed
- Namespace and name must match exactly (case-sensitive)

### Subnet Mapping (`subnet.go`)

Maps the `subnet_ids` array to the `platform.aws.cloudProviderConfig` structure.

#### Process

1. Takes the first subnet ID from the `subnet_ids` array
2. Queries AWS EC2 API to get subnet details
3. Extracts VPC ID and availability zone
4. Populates `cloudProviderConfig`:
   - `subnet.id` = first subnet ID
   - `vpc` = VPC ID from AWS
   - `zone` = availability zone from AWS

#### Validation

- `subnet_ids` array must not be empty
- Subnet ID must match pattern: `subnet-[0-9a-f]{8,17}`
- Subnet must exist in AWS
- Subnet must have VPC ID and availability zone

### Transformation (`transform.go`)

Main entry point that orchestrates the complete transformation.

#### Function: `TransformClusterSpec`

```go
func TransformClusterSpec(
    ctx context.Context,
    spec map[string]interface{},
    ec2Client *ec2.Client,
) (map[string]interface{}, error)
```

**Input**: Raw cluster spec from ROSA CLI with:
- `operator_iam_roles` (array)
- `subnet_ids` (array)

**Output**: Transformed spec with:
- `platform.aws.rolesRef` (object)
- `platform.aws.cloudProviderConfig` (object)
- Original fields preserved

**Notes**:
- Creates a copy of the spec to avoid mutation
- Removes `operator_iam_roles` after transformation
- Preserves `subnet_ids` for other uses
- Preserves existing `platform.aws` fields

## Integration

### Cluster Handler Integration

The mapper is integrated into the cluster creation handler (`pkg/handlers/cluster.go`):

```go
// In NewClusterHandler:
ec2Client := initEC2Client(context.Background(), logger)

// In Create method:
if h.ec2Client != nil {
    transformedSpec, err := mapper.TransformClusterSpec(ctx, req.Spec, h.ec2Client)
    if err != nil {
        // Handle error
        return
    }
    req.Spec = transformedSpec
}
```

### AWS Configuration

The EC2 client uses the default AWS credential chain:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. IAM role (when running on EC2 or ECS)
3. Shared credentials file (`~/.aws/credentials`)

Required IAM permission:
```json
{
  "Effect": "Allow",
  "Action": "ec2:DescribeSubnets",
  "Resource": "*"
}
```

## Example Transformation

### Input (from ROSA CLI)

```json
{
  "name": "mytest-hf",
  "spec": {
    "operator_iam_roles": [
      {
        "name": "cloud-credentials",
        "namespace": "openshift-cloud-network",
        "role_arn": "arn:aws:iam::123456789012:role/rosa/mytest-op-openshift-cloud-network-cloud-credentials"
      },
      // ... 7 more roles
    ],
    "subnet_ids": ["subnet-abc123", "subnet-def456"],
    "region": "us-east-1"
  }
}
```

### Output (to Hyperfleet API)

```json
{
  "name": "mytest-hf",
  "spec": {
    "platform": {
      "aws": {
        "region": "us-east-1",
        "rolesRef": {
          "networkARN": "arn:aws:iam::123456789012:role/rosa/mytest-op-openshift-cloud-network-cloud-credentials",
          "storageARN": "...",
          "imageRegistryARN": "...",
          "kubeCloudControllerARN": "...",
          "nodePoolManagementARN": "...",
          "controlPlaneOperatorARN": "...",
          "ingressARN": "..."
        },
        "cloudProviderConfig": {
          "subnet": {
            "id": "subnet-abc123"
          },
          "vpc": "vpc-0x1y2z3a4b5c6d7e8f",
          "zone": "us-east-1a"
        }
      }
    },
    "subnet_ids": ["subnet-abc123", "subnet-def456"]
  }
}
```

## Error Handling

| Error Condition | Error Message |
|----------------|---------------|
| Empty subnet_ids | `subnet_ids is required and cannot be empty` |
| Invalid subnet ID format | `invalid subnet ID format: <id>` |
| Invalid role ARN | `invalid role ARN format for <key>: <arn>` |
| Missing required role | `missing required operator role: <role>` |
| Duplicate role | `duplicate <role> role found: <key>` |
| Subnet not found in AWS | `failed to describe subnet <id>: <error>` |

## Testing

Run unit tests:
```bash
go test ./pkg/mapper/... -v
```

Tests cover:
- ✅ Valid operator role mapping with all 8 roles
- ✅ Missing required role detection
- ✅ Duplicate role detection
- ✅ Invalid ARN format validation
- ✅ KMS provider role properly ignored
- ✅ Valid and invalid subnet ID formats
- ✅ Empty subnet_ids array handling
- ✅ Spec transformation with existing platform data
- ✅ Error parsing for malformed input

## References

- [Hyperfleet API Mapping Specification](/tmp/hyperfleet-api-mapping-specification.md)
- HyperShift HostedCluster API: `hypershift.openshift.io/v1beta1`
- ROSA CLI: https://github.com/openshift/rosa
