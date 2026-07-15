# Codegen Integration: hyperfleet-api-codegen

**Codegen repo:** `github.com/cdoan1/hyperfleet-api-codegen` (tag: v0.1.7)
**Jira:** [ROSAENG-61801](https://redhat.atlassian.net/browse/ROSAENG-61801)
**Parent:** [ROSAENG-61383](https://redhat.atlassian.net/browse/ROSAENG-61383)

## What was done

Phases 1, 2, and 6 of the [integration spec](https://github.com/cdoan1/hyperfleet-api-codegen/blob/main/docs/integration-rosa-hyperfleet-api.md) are complete. The gateway now imports the codegen repo as a Go library and enforces write-mode (mutable/immutable/service-set) and feature-gate validation on cluster and nodepool mutations.

### Phase 1: Module dependency

Added `github.com/cdoan1/hyperfleet-api-codegen@v0.1.7` as a direct dependency. This required upgrading:

- Go: 1.25.4 → 1.26.0
- k8s.io/apimachinery: v0.34.3 → v0.36.0
- k8s.io/api: v0.34.3 → v0.36.0
- k8s.io/client-go: v0.34.3 → v0.36.0

The k8s.io/client-go upgrade was required because maestro's client-go v0.34.3 references packages removed in k8s.io/api v0.36.0.

**Files changed:** `go.mod`, `go.sum`

### Phase 2: Validation middleware

#### pkg/middleware/field_validation.go (new)

Wraps the codegen's `validation.Validator`. Provides two methods:

- `ValidateCreate(spec, featureSet, enabledGates)` — validates a create request
- `ValidateUpdate(spec, existingSpec, featureSet, enabledGates)` — validates an update request

Key design: the codegen registry uses dotted paths with `spec.` prefix (e.g., `spec.displayName`, `spec.accountId`). The request body spec is a `map[string]interface{}` with keys like `displayName`. The `flattenWithPrefix("spec", spec)` helper recursively flattens nested maps into dotted-path keys to match the registry format.

```go
fv := middleware.NewFieldValidator()
err := fv.ValidateCreate(req.Spec, featuregate.Default, nil)
```

Imports from codegen:
- `github.com/cdoan1/hyperfleet-api-codegen/pkg/validation`
- `github.com/cdoan1/hyperfleet-api-codegen/pkg/featuregate`

#### pkg/handlers/cluster.go (modified)

- Added `fieldValidator *middleware.FieldValidator` to `ClusterHandler` struct
- Updated `NewClusterHandler` signature: added `*middleware.FieldValidator` parameter
- `Create`: after JSON decode and basic field checks, calls `ValidateCreate`. Returns 422 on failure.
- `Update`: fetches existing cluster via `hyperfleetClient.GetCluster()`, then calls `ValidateUpdate` with both new and existing specs. Returns 422 on failure.
- Added `writeValidationError` helper returning structured 422 response

Validation is nil-safe — if `fieldValidator` is nil, validation is skipped. This preserves backward compatibility for tests that pass `nil`.

422 response format:

```json
{
  "kind": "Error",
  "code": "CLUSTERS-MGMT-VALIDATE-001",
  "reason": "Validation failed",
  "details": [
    {"field": "spec.accountId", "reason": "field is platform-managed (service-set) and cannot be set by customers"}
  ]
}
```

#### pkg/handlers/nodepool.go (modified)

Same pattern as cluster handler:
- Added `fieldValidator` field and updated `NewNodePoolHandler` signature
- Validation calls in `Create` and `Update`
- Added `nodePoolSpecToMap` helper to convert `*types.NodePoolSpec` struct to `map[string]interface{}` for the validator

#### pkg/server/server.go (modified)

Creates `middleware.NewFieldValidator()` once and passes it to both `NewClusterHandler` and `NewNodePoolHandler`.

#### pkg/middleware/field_validation_test.go (new, 12 tests)

| Test | What it validates |
|------|-------------------|
| MutableFieldAllowed | `displayName` accepted on create |
| ServiceSetFieldRejected | `accountId` rejected on create |
| MultipleServiceSetFieldsRejected | `accountId`, `creatorARN`, `internalId` all rejected |
| UpdateMutableAllowed | `displayName` change accepted on update |
| UpdateServiceSetRejected | `accountId` rejected on update |
| FeatureGatedFieldRejected | `tags` rejected with Default feature set |
| FeatureGatedFieldAllowedWithTechPreview | `tags` accepted with TechPreviewNoUpgrade |
| UnknownFieldAllowed | Fields not in registry pass through |
| NestedServiceSetFieldRejected | `spec.hostedCluster.pullSecret` rejected |
| MixedFields | Only service-set fields error, mutable fields pass |
| FlattenWithPrefix | Verifies dotted-path key generation |
| FlattenWithPrefix_EmptyMap | Empty input produces empty output |

#### pkg/handlers/cluster_test.go (modified)

- All `NewClusterHandler` calls updated from 3 to 4 args (added `nil` for fieldValidator)
- Added `TestClusterHandler_Create_ValidationRejectsServiceSetField` — sends `accountId` in spec, expects 422
- Added `TestClusterHandler_Create_ValidationAllowsMutableFields` — sends `displayName` only, expects 201

### Phase 6: Makefile targets

Added to `Makefile`:

```makefile
CODEGEN_VERSION ?= v0.1.7

codegen-bump:
    go get github.com/cdoan1/hyperfleet-api-codegen@$(CODEGEN_VERSION)
    go mod tidy

codegen-verify:
    @echo "Verifying codegen dependency compiles..."
    go build ./pkg/middleware/...
    go build ./pkg/handlers/...
```

Usage:

```bash
make codegen-bump CODEGEN_VERSION=v0.1.8   # upgrade codegen dep
make codegen-verify                         # verify codegen packages compile
```

## What remains

| Phase | Description | Effort | Notes |
|-------|-------------|--------|-------|
| 3 | Replace hardcoded service-set injection with codegen conversion functions | Small | Replace manual `req.Spec["cloudUrl"] = ...` in cluster.go with `conversion.UnprojectCluster()` |
| 4 | Migrate to typed specs | Large | Replace `map[string]interface{}` with codegen REST types (`rest.ClusterSpec`). Touches every handler, client, and test that accesses spec fields. |
| 5 | OpenAPI spec alignment | Medium | Replace freeform `spec: object` in openapi.yaml with schemas generated from codegen |

Recommended order: 3 → 5 → 4. Phases 2 and 3 work with existing `map[string]interface{}` types. Phase 4 is the largest change and can be deferred.

## How to recreate this work

Given a clean `main` branch, an AI agent can reproduce Phases 1, 2, and 6 with these instructions:

1. **Add the codegen dependency:**
   ```bash
   go get github.com/cdoan1/hyperfleet-api-codegen@v0.1.7
   ```
   If `go mod tidy` fails on k8s dependency conflicts, upgrade k8s.io/client-go to match the k8s.io/apimachinery version pulled in by the codegen:
   ```bash
   go get k8s.io/client-go@v0.36.0
   go mod tidy
   ```

2. **Create `pkg/middleware/field_validation.go`** — a `FieldValidator` struct wrapping `validation.NewValidator()` with `ValidateCreate` and `ValidateUpdate` methods. Include `flattenWithPrefix` to convert `map[string]interface{}` keys to dotted `spec.*` paths matching the codegen registry.

3. **Wire into handlers** — add `fieldValidator *middleware.FieldValidator` to `ClusterHandler` and `NodePoolHandler` structs. Call `ValidateCreate`/`ValidateUpdate` after JSON decode, before business logic. Return 422 with field-level details on failure. Use nil-checks so tests can pass `nil` to skip validation.

4. **Update `pkg/server/server.go`** — create `middleware.NewFieldValidator()` and pass to both handler constructors.

5. **Add tests** — middleware unit tests covering mutable/service-set/immutable/feature-gated/unknown/nested fields. Handler integration tests for 422 on service-set fields and 201 on valid fields.

6. **Add Makefile targets** — `codegen-bump` and `codegen-verify`.

7. **Verify** — `go build ./...`, `make test`, `make lint` should all pass.

## Pending design decision: where should the codegen code live?

**Status:** Needs team input

The codegen repo (`github.com/cdoan1/hyperfleet-api-codegen`) is currently only used by this API project. Should we merge it into this repo, keep it separate, or do a partial merge?

### The two halves of the codegen repo

The codegen repo contains two distinct categories of code:

| Category | Packages | Dependencies | Used when |
|----------|----------|-------------|-----------|
| **Runtime libraries** | `pkg/registry/`, `pkg/validation/`, `pkg/featuregate/`, `pkg/conversion/` | Lightweight (k8s.io/apimachinery only) | Every API request — imported by handlers and middleware |
| **Generator tools** | `cmd/passthrough-gen`, `cmd/marker-scanner`, `cmd/openapi-gen`, `cmd/conversion-gen`, `cmd/crd-variants`, `cmd/verify-configuration` | Heavy (openshift/hypershift/api, AST parsing, controller-runtime, code-generator) | Only when HyperShift CRDs change — run offline to regenerate types |

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A. Keep separate** (current) | Codegen stays in its own repo, imported as `go get` dependency | Clean separation; heavy generator deps stay out of gateway go.mod; independent release cycle | Two repos to maintain; version coordination on every bump; k8s dependency alignment issues (already hit in Phase 1) |
| **B. Merge everything** | Move entire codegen repo into this repo (e.g., `pkg/codegen/` or `internal/codegen/`) | Single repo; no version coordination; easier cross-cutting changes | Gateway go.mod inherits all generator deps (HyperShift API, controller-runtime, etc.) even though they're only needed at generation time; heavier builds; worse k8s version conflicts |
| **C. Partial merge** (recommended) | Move runtime libraries into this repo (e.g., `internal/codegen/registry/`, `internal/codegen/validation/`). Keep generator tools in the separate repo or as a Go sub-module. | Runtime code is local — no external dep for request-path code; generator deps stay isolated; simpler day-to-day development | Still two places to look for codegen-related code; generator tool changes still require a separate workflow |
| **D. Go workspace** | Use a Go workspace (`go.work`) with both repos checked out side-by-side | Develop across both repos without publishing versions; each repo keeps its own go.mod | Requires both repos checked out locally; CI needs workspace setup; go.work files shouldn't be committed |

### Key considerations

- **Dependency weight:** The generator tools pull in `openshift/hypershift/api`, `controller-runtime`, and k8s code-generator packages. Merging these into the gateway caused k8s.io/api version conflicts during Phase 1 (removed packages in v0.36.0 broke maestro's client-go v0.34.3). This problem gets worse if generators and API share a go.mod.
- **Change frequency:** Generator tools only run when upstream HyperShift CRDs change. Runtime libraries change when validation rules or field metadata evolve. The API server changes frequently. Different cadences favor separation.
- **Team workflow:** A single repo is simpler for code review and CI. Two repos mean PRs that span both are harder to coordinate.
- **Future consumers:** If another project ever needs the codegen output, a separate repo is easier to share. If this API is the only consumer, the separate repo is overhead.

### Decision needed

Team should weigh in on which option to pursue before Phase 3 work begins, since Phase 3 (conversion functions) and Phase 4 (typed specs) will significantly increase the coupling between the two codebases.

## Codegen version bump workflow

When the codegen repo releases a new version:

```bash
make codegen-bump CODEGEN_VERSION=v0.1.8
go build ./...        # compile errors surface breaking changes
make test             # run full test suite
make codegen-verify   # verify codegen-dependent packages
```
