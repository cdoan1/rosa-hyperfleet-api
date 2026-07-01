# SDK Release Notes - ROSA CLI Format Support

**Date:** 2026-07-01  
**Version:** Updated from commit `9e1433b`

## Summary

The SDK has been updated to support creating ROSA HCP clusters using the ROSA CLI format. The Regional Platform API now accepts cluster specifications with `operator_iam_roles` and `subnet_ids` fields and automatically transforms them to the HostedCluster format required by HyperShift.

## What's New

### 1. ROSA CLI Format Support

The SDK now includes type-safe fields for ROSA CLI cluster creation:

```go
type ClusterCreateRequest_Spec struct {
    OperatorIamRoles *[]struct {
        Name      string `json:"name"`
        Namespace string `json:"namespace"`
        RoleArn   string `json:"role_arn"`
    } `json:"operator_iam_roles,omitempty"`
    
    InstallerRoleArn *string `json:"installer_role_arn,omitempty"`
    SupportRoleArn   *string `json:"support_role_arn,omitempty"`
    OidcConfigId     *string `json:"oidc_config_id,omitempty"`
    
    // subnet_ids and region are in AdditionalProperties
}
```

### 2. HostedCluster Format Types

The SDK also includes types for the transformed HostedCluster format:

```go
type ClusterCreateRequest_Spec struct {
    Platform *struct {
        Aws *struct {
            RolesRef *struct {
                NetworkARN              *string
                StorageARN              *string
                ImageRegistryARN        *string
                KubeCloudControllerARN  *string
                NodePoolManagementARN   *string
                ControlPlaneOperatorARN *string
                IngressARN              *string
            }
            CloudProviderConfig *struct {
                Subnet *struct {
                    Id *string
                }
                Vpc  *string
                Zone *string
            }
        }
    }
}
```

### 3. Server-Side Transformation

When you send a cluster create request with ROSA CLI format, the API automatically:

1. **Maps operator roles** from the `operator_iam_roles` array to `platform.aws.rolesRef`
2. **Queries AWS EC2** to get VPC and availability zone from the first subnet in `subnet_ids`
3. **Populates** `platform.aws.cloudProviderConfig` with network metadata

## Migration Guide

### Before (Generic Format)

```go
cluster, err := c.CreateCluster(ctx, &types.ClusterCreateRequest{
    Name: "my-cluster",
    Spec: types.ClusterCreateRequest_Spec{
        AdditionalProperties: map[string]interface{}{
            "provider": "aws",
            "region":   "us-east-1",
        },
    },
})
```

### After (ROSA CLI Format)

```go
// Define operator roles
operatorIamRoles := []struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    RoleArn   string `json:"role_arn"`
}{
    {
        Name:      "cloud-credentials",
        Namespace: "openshift-cloud-network",
        RoleArn:   "arn:aws:iam::123456789012:role/rosa/test-op-openshift-cloud-network-cloud-credentials",
    },
    // ... 6 more required roles
}

installerRoleArn := "arn:aws:iam::123456789012:role/rosa/ManagedOpenShift-HCP-ROSA-Installer-Role"
supportRoleArn := "arn:aws:iam::123456789012:role/rosa/ManagedOpenShift-HCP-ROSA-Support-Role"
oidcConfigID := "2r8p7g9fogag2lg31hph07larks4jpql"

// Create cluster with ROSA CLI format
cluster, err := c.CreateCluster(ctx, &types.ClusterCreateRequest{
    Name: "my-rosa-hcp-cluster",
    Spec: types.ClusterCreateRequest_Spec{
        OperatorIamRoles: &operatorIamRoles,
        InstallerRoleArn: &installerRoleArn,
        SupportRoleArn:   &supportRoleArn,
        OidcConfigId:     &oidcConfigID,
        AdditionalProperties: map[string]interface{}{
            "region":     "us-east-1",
            "subnet_ids": []string{"subnet-0a1b2c3d4e5f6789", "subnet-9k8l7m6n5o4p3q2r"},
        },
    },
})
```

## Required Operator Roles

When using ROSA CLI format, you must provide all 7 required operator roles:

| Namespace | Role Name | Mapped To |
|-----------|-----------|-----------|
| `openshift-cloud-network` | `cloud-credentials` | `networkARN` |
| `openshift-cluster-csi-drivers` | `ebs-cloud-credentials` | `storageARN` |
| `openshift-cloud-network-config-controller` | `cloud-network-config-controller-cloud-credentials` | `imageRegistryARN` |
| `kube-system` | `kube-controller-manager` | `kubeCloudControllerARN` |
| `kube-system` | `capa-controller-manager` | `nodePoolManagementARN` |
| `kube-system` | `control-plane-operator` | `controlPlaneOperatorARN` |
| `openshift-ingress-operator` | `ingress-operator-cloud-credentials` | `ingressARN` |

The `kms-provider` role (namespace: `kube-system`) is optional and used for encryption configuration.

## Error Handling

New error code for transformation failures:

```go
cluster, err := c.CreateCluster(ctx, req)
if err != nil {
    if apiErr, ok := err.(*types.Error); ok {
        if apiErr.Code == "CLUSTERS-MGMT-CREATE-008" {
            // Transformation error - check operator_iam_roles and subnet_ids
            fmt.Printf("Invalid cluster spec: %s\n", apiErr.Reason)
        }
    }
}
```

Possible transformation errors:
- Missing required operator roles
- Invalid role ARN format
- Invalid subnet ID format
- Subnet not found in AWS
- Duplicate operator roles

## Examples

### Complete Example

See the full working example in [`sdk/examples/create_rosa_cluster/main.go`](sdk/examples/create_rosa_cluster/main.go):

```bash
# Set required environment variables
export ROSA_API_URL="https://api.us-east-1.rosa.example.com"
export AWS_REGION="us-east-1"
export AWS_ACCOUNT_ID="123456789012"
export SUBNET_IDS="subnet-0a1b2c3d4e5f6789,subnet-9k8l7m6n5o4p3q2r"

# Run the example
go run sdk/examples/create_rosa_cluster/main.go
```

### OpenAPI Examples

The OpenAPI specification now includes two complete examples:

1. **rosa-cli-format**: Shows the ROSA CLI input format
2. **hostedcluster-format**: Shows the transformed HostedCluster format

These examples are visible in:
- Swagger UI at `/openapi/swagger-ui.html`
- OpenAPI specification at `/openapi/openapi.yaml`
- SDK documentation

## Backward Compatibility

✅ **Fully backward compatible** - existing code continues to work without changes.

The generic format using `AdditionalProperties` still works as before. The new ROSA CLI format is an additional way to create clusters with type-safe fields.

## AWS Permissions

The API server requires the following IAM permission to query subnet metadata:

```json
{
  "Effect": "Allow",
  "Action": "ec2:DescribeSubnets",
  "Resource": "*"
}
```

Client applications do **not** need AWS permissions - the transformation happens server-side.

## Documentation

Updated documentation:
- ✅ SDK README with ROSA CLI examples
- ✅ OpenAPI spec with detailed field descriptions
- ✅ Complete working example in `sdk/examples/create_rosa_cluster/`
- ✅ Type documentation in generated SDK types

## Next Steps

### For SDK Users

1. **Update SDK**: `go get github.com/openshift-online/rosa-hyperfleet-api-sdk@latest`
2. **Review examples**: See `sdk/examples/create_rosa_cluster/` for complete usage
3. **Test creation**: Try creating a ROSA HCP cluster with the new format
4. **Check errors**: Handle the new `CLUSTERS-MGMT-CREATE-008` error code

### For ROSA CLI Integration

The ROSA CLI can now use this API to create clusters without needing to:
- Query AWS EC2 for VPC/AZ information
- Transform operator roles to HostedCluster format
- Implement complex mapping logic

The API handles all transformation server-side.

## Support

For questions or issues:
- **GitHub Issues**: https://github.com/openshift-online/rosa-hyperfleet-api/issues
- **Documentation**: See SDK README and OpenAPI spec
- **Examples**: Complete working examples in `sdk/examples/`

## Release Commits

- **Mapper Implementation**: `9ee6dec` - Server-side transformation logic
- **SDK Update**: `9e1433b` - OpenAPI spec and SDK generation

---

**Ready for use!** The SDK is available now with full ROSA CLI format support.
