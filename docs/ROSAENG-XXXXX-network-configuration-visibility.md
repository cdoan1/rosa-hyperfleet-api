# Implementation Spec: Expose NetworkConfiguration Fields in REST API

**Status:** Draft  
**Target Release:** TBD  
**Related Issues:** ROSAENG-66150

## Problem Statement

The `NetworkConfiguration` type fields (e.g., `clusterNetwork`, `serviceNetwork`, `networkType`) are defined in the CRD types with `+k8s:openapi-gen=true` markers indicating they should be visible in the REST API, but they are not being returned to clients. The clientset receives empty values for these fields even when they are set in the backend.

## Root Cause Analysis

### Current Architecture

The ROSA HyperFleet API uses a multi-layer type system:

1. **CRD Types** (`api/v1alpha1/`) - Internal Kubernetes types with full field set
2. **REST Types** (`api/v1alpha1/public/`) - Public API types with service-set fields filtered out
3. **Conversion Functions** (`platform-api/pkg/conversion/v1alpha1/`) - Convert between CRD and REST

The conversion uses JSON roundtrip for field projection:
```go
// ProjectCluster filters CRD → REST (visible fields only)
func projectClusterSpec(crd v1alpha1.ClusterSpec) rest.ClusterSpec {
    data, _ := json.Marshal(crd)
    var out rest.ClusterSpec
    _ = json.Unmarshal(data, &out)
    return out
}
```

### The Bug

The issue is in the generated REST types at `api/v1alpha1/public/hostedclusterspecpassthrough_types.go`:

**CRD Type** (correct):
```go
// api/v1alpha1/zz_generated.passthrough.go
type HostedClusterSpecPassthrough struct {
    Configuration *v1alpha1.ClusterConfiguration `json:"configuration,omitempty"`
    // ^^^ Uses our custom type with Network field
}

// api/v1alpha1/configuration.go
type ClusterConfiguration struct {
    Network *NetworkConfiguration `json:"network,omitempty"`  // ✅ Has Network field
    Kubelet *KubeletConfig `json:"kubelet,omitempty"`
}
```

**REST Type** (wrong):
```go
// api/v1alpha1/public/hostedclusterspecpassthrough_types.go
type HostedClusterSpecPassthrough struct {
    Configuration *hypershiftv1beta1.ClusterConfiguration `json:"configuration,omitempty"`
    // ^^^ Uses upstream HyperShift type WITHOUT Network field ❌
}
```

When JSON roundtrip conversion happens:
1. Marshal CRD with `Configuration.Network` populated → JSON includes `"configuration":{"network":{...}}`
2. Unmarshal into REST type with `hypershiftv1beta1.ClusterConfiguration` → Network data is **silently dropped** because upstream type doesn't have that field
3. Client receives empty/null network configuration

### Why This Happens

The `conversion-gen` tool recognizes the `+hyperfleet:upstream-reduced-object` marker on `ClusterConfiguration`:

```go
// api/v1alpha1/configuration.go
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.ClusterConfiguration
type ClusterConfiguration struct {
    Network *NetworkConfiguration `json:"network,omitempty"`
    // ... additional fields with custom markers
}
```

This marker indicates that `v1alpha1.ClusterConfiguration` is a **reduced/modified copy** of the upstream `hypershiftv1beta1.ClusterConfiguration` type, allowing us to add custom markers (like `+hyperfleet:write-mode=mutable` on individual fields).

**However**, the conversion-gen currently:
- ✅ Generates mirror types for **nested** types like `KubeletConfig` and `MachineConfigSpec` (they appear as separate `_types.go` files in `api/v1alpha1/public/`)
- ❌ Does NOT generate a REST type for `ClusterConfiguration` itself
- ❌ Leaves the upstream `hypershiftv1beta1.ClusterConfiguration` reference in the REST passthrough types

## Current Workarounds

### Registry Shows Fields Correctly

The field metadata registry (`hack/api-codegen/pkg/registry/field_metadata.go`) correctly tracks the Network fields:

```go
"spec.hostedCluster.configuration.network": {
    FieldPath: "spec.hostedCluster.configuration.network",
    WriteMode: ServiceSet,
    OwnerType: "Cluster",
},
"spec.hostedCluster.configuration.network.clusterNetwork": {
    WriteMode: Immutable,
},
// ... etc
```

### OpenAPI Spec Includes Fields

The OpenAPI spec (`api/v1alpha1/public/openapi.yaml`) includes the fields after running `make generate-openapi` (after adding `NetworkConfiguration` to the merge list).

### What Was Already Done

1. ✅ Added `NetworkConfiguration` type with fields to `api/v1alpha1/configuration.go`
2. ✅ Changed `Network` field marker from `+k8s:openapi-gen=false` → `+k8s:openapi-gen=true`
3. ✅ Added `NetworkConfiguration` to `typeToRegistryPrefix` in `hack/api-codegen/pkg/openapi/generator.go`
4. ✅ Added `NetworkConfiguration,ClusterNetworkEntry` to OpenAPI merge list in `Makefile`
5. ✅ Added debug logging to `platform-api/pkg/conversion/v1alpha1/cluster.go`

## Proposed Solution

### Option A: Generate REST ClusterConfiguration Type (Recommended)

Extend `conversion-gen` to handle `+hyperfleet:upstream-reduced-object` types properly:

1. Detect types marked with `+hyperfleet:upstream-reduced-object`
2. Generate a corresponding REST type (e.g., `api/v1alpha1/public/clusterconfiguration_types.go`)
3. Update references in generated passthrough types to use the local REST type instead of upstream

**Result:**
```go
// api/v1alpha1/public/clusterconfiguration_types.go (NEW FILE - GENERATED)
type ClusterConfiguration struct {
    Network *NetworkConfiguration `json:"network,omitempty"`  // ✅ Visible fields only
    Kubelet *KubeletConfig `json:"kubelet,omitempty"`
    MachineConfig *MachineConfigSpec `json:"machineConfig,omitempty"`
    // Hidden fields excluded: APIServer, Authentication, FeatureGate, etc.
}

// api/v1alpha1/public/hostedclusterspecpassthrough_types.go (UPDATED)
type HostedClusterSpecPassthrough struct {
    Configuration *ClusterConfiguration `json:"configuration,omitempty"`
    // ^^^ Now uses our local REST type ✅
}
```

### Option B: Manual Override (Workaround)

Manually maintain a REST `ClusterConfiguration` type and prevent regeneration:

1. Manually write `api/v1alpha1/public/clusterconfiguration_types.go`
2. Add to `.codegen-ignore` or similar mechanism
3. Manually keep it in sync with CRD changes

**Downside:** Requires manual maintenance, error-prone, defeats purpose of codegen.

## Implementation Plan

### Phase 1: Update conversion-gen to handle upstream-reduced types

**Files to modify:**

1. **`hack/api-codegen/pkg/conversion/generator.go`**
   - Add logic to detect `+hyperfleet:upstream-reduced-object` marker
   - Generate REST types for upstream-reduced objects
   - Track these types in the REST type registry

2. **`hack/api-codegen/pkg/conversion/mirror_types.go`**
   - Extend `MirrorTypeMapping` to track upstream-reduced objects
   - Add registry of types requiring REST generation

3. **`hack/api-codegen/pkg/markers/scanner.go`**
   - Parse `+hyperfleet:upstream-reduced-object=<upstream-type>` marker
   - Store in type metadata for generator consumption

**Implementation steps:**

#### Step 1: Extend marker scanning

```go
// hack/api-codegen/pkg/markers/scanner.go

var upstreamReducedPattern = regexp.MustCompile(`\+hyperfleet:upstream-reduced-object=([a-zA-Z0-9_.]+)`)

type TypeMeta struct {
    Name              string
    UpstreamType      string  // NEW: populated from +hyperfleet:upstream-reduced-object marker
    NeedsRESTMirror   bool    // NEW: true if this type needs REST generation
}

func (s *MarkerScanner) scanTypeMarkers(node *ast.TypeSpec) *TypeMeta {
    // ... existing code ...
    
    // Check for upstream-reduced-object marker
    if matches := upstreamReducedPattern.FindStringSubmatch(comments); len(matches) > 1 {
        meta.UpstreamType = matches[1]
        meta.NeedsRESTMirror = true
    }
    
    return meta
}
```

#### Step 2: Track upstream-reduced types in generator

```go
// hack/api-codegen/pkg/conversion/generator.go

type typeInfo struct {
    Name            string
    StructType      *ast.StructType
    Doc             *ast.CommentGroup
    Fields          []*fieldInfo
    Markers         []string
    Embeds          []string
    UpstreamType    string  // NEW: from +hyperfleet:upstream-reduced-object
    NeedsRESTMirror bool    // NEW: true if REST type should be generated
}

func (g *Generator) scanTypes() error {
    // ... existing code ...
    
    for _, file := range pkgFiles {
        ast.Inspect(file, func(n ast.Node) bool {
            if typeSpec, ok := n.(*ast.TypeSpec); ok {
                // Check for upstream-reduced-object marker in comments
                comments := typeSpec.Doc.Text()
                if matches := upstreamReducedPattern.FindStringSubmatch(comments); len(matches) > 1 {
                    ti.UpstreamType = matches[1]
                    ti.NeedsRESTMirror = true
                }
            }
        })
    }
}
```

#### Step 3: Generate REST types for upstream-reduced objects

```go
// hack/api-codegen/pkg/conversion/generator.go

func (g *Generator) generateRESTTypes() error {
    // ... existing code for root types (Cluster, NodePool, etc.) ...
    
    // NEW: Generate REST types for upstream-reduced objects
    for typeName, ti := range g.typeInfos {
        if !ti.NeedsRESTMirror {
            continue
        }
        
        filename := filepath.Join(g.restOutputDir(), toSnakeCase(typeName)+"_types.go")
        content, err := g.renderRESTType(ti, restTypeSet, false)
        if err != nil {
            return fmt.Errorf("rendering REST type %s: %w", typeName, err)
        }
        
        if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
            return fmt.Errorf("writing %s: %w", filename, err)
        }
    }
    
    return nil
}
```

#### Step 4: Replace upstream type references in passthrough types

```go
// hack/api-codegen/pkg/conversion/generator.go

func (g *Generator) qualifyTypeForREST(goType string, restTypeSet map[string]bool) string {
    base := strings.TrimPrefix(goType, "*")
    base = strings.TrimPrefix(base, "[]")
    
    // Check if this is an upstream-reduced type
    if ti, ok := g.typeInfos[base]; ok && ti.NeedsRESTMirror {
        // Use local REST type instead of upstream
        return strings.ReplaceAll(goType, base, base)  // No qualification needed, same package
    }
    
    // ... existing qualification logic ...
}
```

### Phase 2: Update build process

**Files to modify:**

1. **`Makefile`**
   - Ensure `codegen-conversion` target runs with updated generator
   - Add verification step for upstream-reduced types

2. **`.gitignore`** (if needed)
   - Ensure generated `*_types.go` files are tracked

### Phase 3: Verification

**Test cases:**

1. **Unit test for marker scanning:**
   ```go
   // hack/api-codegen/pkg/markers/scanner_test.go
   func TestUpstreamReducedObjectMarker(t *testing.T) {
       input := `
       // +hyperfleet:upstream-reduced-object=hypershiftv1beta1.ClusterConfiguration
       type ClusterConfiguration struct {
           Network *NetworkConfiguration
       }
       `
       // Assert: UpstreamType = "hypershiftv1beta1.ClusterConfiguration"
       // Assert: NeedsRESTMirror = true
   }
   ```

2. **Integration test for REST type generation:**
   ```bash
   # Verify ClusterConfiguration REST type is generated
   test -f api/v1alpha1/public/clusterconfiguration_types.go
   
   # Verify it contains Network field
   grep -q "Network \*NetworkConfiguration" api/v1alpha1/public/clusterconfiguration_types.go
   
   # Verify it does NOT contain hidden fields
   ! grep -q "APIServer\|Authentication\|FeatureGate" api/v1alpha1/public/clusterconfiguration_types.go
   ```

3. **E2E test with platform-api:**
   ```go
   // Test that Network fields roundtrip correctly
   func TestNetworkConfigurationRoundtrip(t *testing.T) {
       crd := &v1alpha1.Cluster{
           Spec: v1alpha1.ClusterSpec{
               HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
                   Configuration: &v1alpha1.ClusterConfiguration{
                       Network: &v1alpha1.NetworkConfiguration{
                           NetworkType: "OVNKubernetes",
                           ServiceNetwork: []string{"172.30.0.0/16"},
                       },
                   },
               },
           },
       }
       
       // Convert CRD → REST
       rest := v1alpha1conv.ProjectCluster(crd)
       
       // Assert: Network fields are preserved
       assert.NotNil(t, rest.Spec.HostedCluster.Configuration.Network)
       assert.Equal(t, "OVNKubernetes", rest.Spec.HostedCluster.Configuration.Network.NetworkType)
   }
   ```

## Files Affected

### Modified Files

1. `hack/api-codegen/pkg/markers/scanner.go` - Add upstream-reduced-object pattern
2. `hack/api-codegen/pkg/markers/types.go` - Extend TypeMeta
3. `hack/api-codegen/pkg/conversion/generator.go` - Generate REST types for upstream-reduced
4. `hack/api-codegen/pkg/conversion/mirror_types.go` - Track upstream-reduced mappings
5. `Makefile` - Update codegen targets (if needed)
6. `api/v1alpha1/configuration.go` - Already has `+hyperfleet:upstream-reduced-object` marker ✅

### Generated Files (will be created/updated)

1. `api/v1alpha1/public/clusterconfiguration_types.go` - **NEW** (will be generated)
2. `api/v1alpha1/public/networkconfiguration_types.go` - **NEW** (will be generated)
3. `api/v1alpha1/public/clusternetworkentry_types.go` - **NEW** (will be generated)
4. `api/v1alpha1/public/hostedclusterspecpassthrough_types.go` - **UPDATED** (Configuration field type changes)
5. `platform-api/pkg/conversion/v1alpha1/cluster.go` - **UPDATED** (conversion helpers)
6. `hack/api-codegen/pkg/registry/field_metadata.go` - **UPDATED** (already correct)

## Dependencies

### Existing Infrastructure

- ✅ Field metadata registry system (`hack/api-codegen/pkg/registry/`)
- ✅ Marker scanning system (`hack/api-codegen/pkg/markers/`)
- ✅ Conversion generator (`hack/api-codegen/pkg/conversion/`)
- ✅ OpenAPI generator integration

### New Dependencies

- None - uses existing codegen infrastructure

## Migration Strategy

### Backward Compatibility

This change is **backward compatible** because:

1. **Wire format unchanged:** JSON field names remain the same
2. **Additive change:** Previously null fields now return data
3. **No breaking changes:** Existing clients ignoring these fields continue to work

### Rollout Plan

1. **Phase 1:** Implement conversion-gen changes (this spec)
2. **Phase 2:** Run `make generate` to regenerate types
3. **Phase 3:** Verify with integration tests
4. **Phase 4:** Deploy to staging environment
5. **Phase 5:** Verify clients can read Network fields
6. **Phase 6:** Deploy to production

### Rollback Plan

If issues arise:
1. Revert the `+k8s:openapi-gen=true` marker on `Network` field in `api/v1alpha1/configuration.go`
2. Run `make generate` to hide the field again
3. The field becomes service-set only (invisible to API)

## Testing Strategy

### Unit Tests

- `hack/api-codegen/pkg/markers/scanner_test.go` - Marker parsing
- `hack/api-codegen/pkg/conversion/generator_test.go` - REST type generation

### Integration Tests

- `make test-api-codegen` - Codegen tool tests
- `make verify-codegen` - Verify generated files are up to date

### E2E Tests

- Create cluster with Network configuration
- Verify GET /clusters/{id} returns Network fields
- Verify clientset can read Network fields

## Success Criteria

1. ✅ `make generate` produces `api/v1alpha1/public/clusterconfiguration_types.go`
2. ✅ Generated REST type includes `Network *NetworkConfiguration` field
3. ✅ Generated REST type excludes hidden fields (APIServer, Authentication, etc.)
4. ✅ `make verify` passes without diffs
5. ✅ Integration tests verify Network fields roundtrip correctly
6. ✅ OpenAPI spec includes NetworkConfiguration schema
7. ✅ Clients can read clusterNetwork, serviceNetwork, networkType

## Future Enhancements

### Additional Upstream-Reduced Types

Other configuration types may benefit from the same pattern:

- `APIServerNetworkConfiguration`
- `IngressConfiguration`
- `OAuthConfiguration`
- `ProxyConfiguration`

### Automated Detection

The conversion-gen could automatically detect field mismatches:

```go
// Warn if CRD field type != REST field type for same JSON name
if crdField.Type != restField.Type {
    log.Warnf("Type mismatch: %s.%s (CRD: %s, REST: %s)",
        typeName, fieldName, crdField.Type, restField.Type)
}
```

## Open Questions

1. **Should we generate REST types for ALL upstream-reduced objects, or only those with visible fields?**
   - **Recommendation:** Only generate if at least one field is visible (has `+k8s:openapi-gen=true`)

2. **Should the REST type name match the CRD type name exactly?**
   - **Recommendation:** Yes, keep names identical for clarity (both `ClusterConfiguration`)

3. **How to handle deeply nested upstream-reduced types?**
   - **Recommendation:** Generate REST types for all levels, maintain same nesting structure

## References

- ROSAENG-66150 - Original ticket for exposing NetworkConfiguration
- ROSAENG-61799 - Codegen phase dependency chain
- ROSAENG-62606 - Conversion-gen and openapi-gen refactor
- `CLAUDE.md` - Project build conventions
- `docs/codegen-pipeline.md` - Codegen architecture (if exists)

## Implementation Checklist

- [ ] Create feature branch: `feature/ROSAENG-XXXXX-network-config-visibility`
- [ ] Extend marker scanner to parse `+hyperfleet:upstream-reduced-object`
- [ ] Update conversion generator to track upstream-reduced types
- [ ] Implement REST type generation for upstream-reduced objects
- [ ] Add unit tests for marker parsing
- [ ] Add integration tests for REST type generation
- [ ] Run `make generate` and verify output
- [ ] Add E2E test for Network field roundtrip
- [ ] Update documentation (if needed)
- [ ] Create PR with detailed testing notes
- [ ] Code review
- [ ] Merge to main
- [ ] Deploy to staging
- [ ] Verify with real clients
- [ ] Deploy to production

## Appendix A: Current Type Hierarchy

```
CRD Types (api/v1alpha1/):
  Cluster
    └── ClusterSpec
        └── HostedClusterSpecPassthrough (from zz_generated.passthrough.go)
            └── Configuration *v1alpha1.ClusterConfiguration ✅
                └── Network *NetworkConfiguration ✅

REST Types (api/v1alpha1/public/):
  Cluster
    └── ClusterSpec
        └── HostedClusterSpecPassthrough
            └── Configuration *hypershiftv1beta1.ClusterConfiguration ❌
                └── (no Network field) ❌
```

## Appendix B: Debug Logging Output (Expected)

With the debug logging added to `platform-api/pkg/conversion/v1alpha1/cluster.go`, you should see:

```
[ProjectCluster] CRD ClusterSpec JSON: {"hostedCluster":{"configuration":{"network":{"networkType":"OVNKubernetes","serviceNetwork":["172.30.0.0/16"]}}}}
[ProjectCluster] REST ClusterSpec JSON: {"hostedCluster":{"configuration":{}}}
[ProjectCluster] CRD Network config: {"networkType":"OVNKubernetes","serviceNetwork":["172.30.0.0/16"]}
[ProjectCluster] REST Network config: null
```

After the fix:

```
[ProjectCluster] CRD ClusterSpec JSON: {"hostedCluster":{"configuration":{"network":{"networkType":"OVNKubernetes","serviceNetwork":["172.30.0.0/16"]}}}}
[ProjectCluster] REST ClusterSpec JSON: {"hostedCluster":{"configuration":{"network":{"networkType":"OVNKubernetes","serviceNetwork":["172.30.0.0/16"]}}}}
[ProjectCluster] CRD Network config: {"networkType":"OVNKubernetes","serviceNetwork":["172.30.0.0/16"]}
[ProjectCluster] REST Network config: {"networkType":"OVNKubernetes","serviceNetwork":["172.30.0.0/16"]}
```

## Appendix C: Alternative Approaches Considered

### Alternative 1: Manual Field-by-Field Conversion

Instead of JSON roundtrip, manually copy visible fields:

```go
func projectClusterConfiguration(crd *v1alpha1.ClusterConfiguration) *rest.ClusterConfiguration {
    if crd == nil {
        return nil
    }
    return &rest.ClusterConfiguration{
        Network:       projectNetworkConfiguration(crd.Network),
        Kubelet:       crd.Kubelet,  // Already filtered
        MachineConfig: crd.MachineConfig,
    }
}
```

**Pros:** Explicit, easy to debug  
**Cons:** Requires manual maintenance, defeats codegen purpose

### Alternative 2: Use Interface Types

Define an interface for Configuration and let runtime determine type:

```go
type Configuration interface {
    GetNetwork() *NetworkConfiguration
}
```

**Pros:** Type-safe  
**Cons:** Breaks JSON marshaling, requires extensive refactoring

### Alternative 3: Post-Processing Enrichment

Keep upstream types, but enrich after unmarshaling:

```go
func projectClusterSpec(crd v1alpha1.ClusterSpec) rest.ClusterSpec {
    data, _ := json.Marshal(crd)
    var out rest.ClusterSpec
    _ = json.Unmarshal(data, &out)
    
    // Manually copy fields not in upstream type
    if crd.HostedCluster.Configuration != nil {
        out.HostedCluster.Configuration.Network = crd.HostedCluster.Configuration.Network
    }
    
    return out
}
```

**Pros:** Quick fix  
**Cons:** Fragile, breaks with type changes, defeats codegen purpose

## Appendix D: Estimated Effort

- **Marker scanning extension:** 2-4 hours
- **Generator logic for REST types:** 4-8 hours
- **Unit tests:** 2-4 hours
- **Integration tests:** 2-4 hours
- **E2E tests:** 2-4 hours
- **Documentation:** 2 hours
- **Code review & iteration:** 4 hours

**Total:** 18-30 hours (2.5-4 days)
