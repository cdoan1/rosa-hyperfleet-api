# ROSA Regional Platform API SDK Changelog

## [0.2.0] - 2026-07-01

### CRITICAL FIX
- **Fixed incorrect operator role mappings** that caused ROSA HCP cluster creation to fail
  - `networkARN`: Now correctly maps from `openshift-cloud-network-config-controller/cloud-network-config-controller-cloud-credentials`
  - `imageRegistryARN`: Now correctly maps from `openshift-image-registry/installer-cloud-credentials`  
  - `ingressARN`: Now correctly maps from `openshift-ingress-operator/cloud-credentials`

### Added
- Comprehensive ROSA CLI format support with type-safe fields:
  - `OperatorIamRoles` field for operator IAM role configuration
  - `InstallerRoleArn`, `SupportRoleArn`, `OidcConfigId` fields
  - Full `Platform.Aws.RolesRef` structure for HostedCluster format
  - `Platform.Aws.CloudProviderConfig` structure with VPC/subnet/zone
- Complete working example in `examples/create_rosa_cluster/`
- Detailed documentation of server-side transformation behavior
- OpenAPI examples showing both ROSA CLI and HostedCluster formats

### Changed
- Updated OpenAPI spec with detailed field descriptions and validation patterns
- Enhanced README with ROSA HCP cluster creation guide
- Regenerated types from corrected OpenAPI specification

### Migration Guide
If you were using v0.1.0 and sending operator_iam_roles manually, you need to update the role names:

**Before (v0.1.0 - WRONG):**
```go
{Name: "cloud-credentials", Namespace: "openshift-cloud-network"}  // networkARN
{Name: "ingress-operator-cloud-credentials", Namespace: "openshift-ingress-operator"}  // ingressARN
```

**After (v0.2.0 - CORRECT):**
```go
{Name: "cloud-network-config-controller-cloud-credentials", Namespace: "openshift-cloud-network-config-controller"}  // networkARN
{Name: "installer-cloud-credentials", Namespace: "openshift-image-registry"}  // imageRegistryARN
{Name: "cloud-credentials", Namespace: "openshift-ingress-operator"}  // ingressARN
```

See the [required operator roles table](README.md#required-operator-roles) for the complete mapping.

## [0.1.0] - 2026-06-29

### Added
- Initial SDK release with type-safe client for ROSA Regional Platform API
- Support for clusters, nodepools, management clusters, authorization, accounts, and ZOA operations
- AWS SigV4 authentication
- Comprehensive error handling with helper functions
- Generated types from OpenAPI specification
- Working examples for common operations

---

**Note:** The SDK uses independent versioning with the `sdk/vX.Y.Z` tag format to allow SDK releases independent of the main API server.
