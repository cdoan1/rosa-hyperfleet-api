# SDK v0.2.0 Release - CRITICAL FIX

**Release Date:** 2026-07-01  
**Tag:** `sdk/v0.2.0`  
**Status:** 🔴 **CRITICAL - IMMEDIATE UPDATE RECOMMENDED**

---

## ⚠️ CRITICAL FIX: Operator Role Mappings Corrected

SDK v0.2.0 fixes **critical errors** in the operator role to HostedCluster mapping that were causing ROSA HCP cluster creation to fail.

### What Was Wrong

The v0.1.0 SDK contained incorrect mappings based on assumptions that didn't match the actual operator roles created by `rosa create operator-roles --hosted-cp`. This caused clusters to fail with:

```
Error: missing required operator role: network
```

### What's Fixed

Three critical operator role mappings have been corrected:

| Field | v0.1.0 (WRONG) | v0.2.0 (CORRECT) |
|-------|---------------|------------------|
| **networkARN** | `openshift-cloud-network`<br>`cloud-credentials` | `openshift-cloud-network-config-controller`<br>`cloud-network-config-controller-cloud-credentials` |
| **imageRegistryARN** | `openshift-cloud-network-config-controller`<br>`cloud-network-config-controller-cloud-credentials` | `openshift-image-registry`<br>`installer-cloud-credentials` |
| **ingressARN** | `openshift-ingress-operator`<br>`ingress-operator-cloud-credentials` | `openshift-ingress-operator`<br>`cloud-credentials` |

---

## Installation

### For New Projects

```bash
go get github.com/cdoan1/rosa-hyperfleet-api/sdk@sdk/v0.2.0
```

### For Existing Projects (Upgrade from v0.1.0)

```bash
go get github.com/cdoan1/rosa-hyperfleet-api/sdk@sdk/v0.2.0
go mod tidy
```

---

## Breaking Changes

### If You're Using ROSA CLI Format (Most Users)

**No code changes needed!** The ROSA CLI sends the correct role names, and the server-side transformation now correctly maps them.

### If You're Manually Constructing Operator Roles

You **MUST** update your role configurations:

**Before (v0.1.0):**
```go
operatorIamRoles := []struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    RoleArn   string `json:"role_arn"`
}{
    {
        Name:      "cloud-credentials",              // ❌ WRONG
        Namespace: "openshift-cloud-network",        // ❌ WRONG
        RoleArn:   "arn:aws:iam::...",
    },
    // ...
}
```

**After (v0.2.0):**
```go
operatorIamRoles := []struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    RoleArn   string `json:"role_arn"`
}{
    {
        Name:      "cloud-network-config-controller-cloud-credentials",  // ✅ CORRECT
        Namespace: "openshift-cloud-network-config-controller",          // ✅ CORRECT
        RoleArn:   "arn:aws:iam::...",
    },
    {
        Name:      "installer-cloud-credentials",    // ✅ CORRECT
        Namespace: "openshift-image-registry",       // ✅ CORRECT
        RoleArn:   "arn:aws:iam::...",
    },
    {
        Name:      "cloud-credentials",              // ✅ CORRECT
        Namespace: "openshift-ingress-operator",     // ✅ CORRECT
        RoleArn:   "arn:aws:iam::...",
    },
    // ... other roles
}
```

See the [complete example](sdk/examples/create_rosa_cluster/main.go) for all 8 operator roles.

---

## What's New in v0.2.0

### ROSA CLI Format Support

Full type-safe support for ROSA CLI cluster creation format:

```go
import (
    "github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/client"
    "github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

cluster, err := c.CreateCluster(ctx, &types.ClusterCreateRequest{
    Name: "my-rosa-hcp-cluster",
    Spec: types.ClusterCreateRequest_Spec{
        OperatorIamRoles: &operatorIamRoles,      // Type-safe!
        InstallerRoleArn: &installerRoleArn,
        SupportRoleArn:   &supportRoleArn,
        OidcConfigId:     &oidcConfigID,
        // subnet_ids and region in AdditionalProperties
    },
})
```

### Server-Side Transformation

The API now automatically:
- ✅ Maps `operator_iam_roles` → `platform.aws.rolesRef`
- ✅ Queries AWS EC2 for VPC and availability zone from `subnet_ids`
- ✅ Populates `platform.aws.cloudProviderConfig` with network metadata

### Documentation

- ✅ Complete [SDK README](sdk/README.md) with ROSA HCP examples
- ✅ Working [example code](sdk/examples/create_rosa_cluster/)
- ✅ [CHANGELOG](sdk/CHANGELOG.md) with migration guide
- ✅ Updated OpenAPI specification with detailed examples

---

## Verification

After upgrading, verify cluster creation works:

```bash
# Create operator roles (actual ROSA CLI command)
rosa create operator-roles \
  --prefix mytest \
  --oidc-config-id <YOUR_OIDC_ID> \
  --hosted-cp \
  --mode auto

# Create cluster using SDK v0.2.0
# Your Go code using the SDK...

# Verify HostedCluster CR has all fields populated
kubectl get hostedcluster <name> -o jsonpath='{.spec.platform.aws.rolesRef}' | jq
```

Expected output (all 7 fields populated):
```json
{
  "networkARN": "arn:aws:iam::...:role/.../mytest-openshift-cloud-network-config-controller-...",
  "storageARN": "arn:aws:iam::...:role/.../mytest-openshift-cluster-csi-drivers-...",
  "imageRegistryARN": "arn:aws:iam::...:role/.../mytest-openshift-image-registry-...",
  "kubeCloudControllerARN": "arn:aws:iam::...:role/.../mytest-kube-system-kube-controller-manager",
  "nodePoolManagementARN": "arn:aws:iam::...:role/.../mytest-kube-system-capa-controller-manager",
  "controlPlaneOperatorARN": "arn:aws:iam::...:role/.../mytest-kube-system-control-plane-operator",
  "ingressARN": "arn:aws:iam::...:role/.../mytest-openshift-ingress-operator-cloud-credentials"
}
```

---

## Complete Changelog

See [CHANGELOG.md](sdk/CHANGELOG.md) for full details.

---

## Support

### Issues
Report issues at: https://github.com/openshift-online/rosa-hyperfleet-api/issues

### Documentation
- [SDK README](sdk/README.md)
- [OpenAPI Specification](openapi/openapi.yaml)
- [Examples](sdk/examples/)

### Migration Help
If you need help upgrading from v0.1.0, see the [migration guide in CHANGELOG.md](sdk/CHANGELOG.md#migration-guide).

---

## Commits in This Release

- `9ee6dec` - Add ROSA CLI to HostedCluster CR mapping for cluster creation
- `9e1433b` - Document ROSA CLI format in OpenAPI spec and update SDK
- `0fc3c1d` - Add SDK release notes for ROSA CLI format support
- `7e92411` - **CRITICAL FIX:** Correct operator role mappings for ROSA HCP clusters
- `767071a` - Release SDK v0.2.0 with corrected operator role mappings

---

**Action Required:** Update to SDK v0.2.0 immediately if you are creating ROSA HCP clusters.
