# Codegen Field Validation on pgruntime Branch Review

**Source:** PR #150 (ROSAENG-61801) — codegen field validation for cluster/nodepool handlers
**Target base:** `pgruntime` branch (PostgreSQL-backed hyperfleetdb architecture)
**Branch:** `pgruntime-codegen`
**Date:** 2026-07-17
**Jira:** [ROSAENG-61801](https://redhat.atlassian.net/browse/ROSAENG-61801), [ROSAENG-62067](https://redhat.atlassian.net/browse/ROSAENG-62067)

---

## API Version Reference

This table lists every version identifier in the codebase and what it means. These are **not** sequential versions of the same API — they are distinct systems with different owners and purposes.

| Version | Full Identifier | Owner | Where It Lives | What It Is | Used At Runtime? |
|---|---|---|---|---|---|
| **`/api/v0/`** | REST route prefix | rosa-hyperfleet-api | `pkg/server/server.go` routes | The HTTP REST API that external clients (rosactl, UI) call. Routes like `POST /api/v0/clusters`, `GET /api/v0/nodepools`. This is the only version customers see. | Yes — all HTTP traffic |
| **`hyperfleetv1alpha1`** | `github.com/typeid/hyperfleet-operator/api/v1alpha1` | hyperfleet-operator repo | `pkg/clients/hyperfleetdb/` | Kubernetes CRD types (`Cluster`, `NodePool`, `Manifest`, `ManagementCluster`) stored in PostgreSQL via pgruntime. The Go structs that define the actual data schema. GroupVersion: `hyperfleet.io/v1alpha1`. | Yes — all DB reads/writes |
| **`v2alpha1`** | `api/v2alpha1/` (local package) | rosa-hyperfleet-api codegen | `api/v2alpha1/` | Annotated type definitions with `+hyperfleet:write-mode` and `+openshift:enable:FeatureGate` markers. Fed into codegen tools (`marker-scanner`) to generate the field validation registry. These types mirror HyperShift fields but are **not** used for storage or serialization. | **No** — codegen input only |
| **`hypershiftv1beta1`** | `github.com/openshift/hypershift/api/hypershift/v1beta1` | openshift/hypershift repo | `api/v2alpha1/hostedclusterspec.passthrough.go`, `hyperfleetv1alpha1` types | Upstream HyperShift CRD types (`HostedClusterSpec`, `NodePoolSpec`, `Release`, `PlatformSpec`, etc.). Both `hyperfleetv1alpha1` and `v2alpha1` embed or reference these. | Indirectly — via embedded fields |
| **`configv1`** | `github.com/openshift/api/config/v1` | openshift/api repo | `api/v2alpha1/hostedclusterspec.passthrough.go` | OpenShift config types (e.g., `configv1.URL`). Used by the passthrough types. | Indirectly — via embedded fields |
| **`/api/v1/`** | Thanos/Prometheus route prefix | Thanos (external) | `internal/test/thanos/helpers.go` | Prometheus-compatible query API used in E2E test helpers for metrics validation. Not part of this service's API. | No — test helpers only |

### Key Relationships

```
Customer (rosactl / UI)
  │
  ▼
/api/v0/clusters (REST)                    ← rosa-hyperfleet-api HTTP routes
  │
  ▼
pkg/handlers/cluster.go
  │
  ├── validates via ──► field registry     ← generated FROM api/v2alpha1/ markers
  │                      (map[string]FieldMeta)      (codegen input, not runtime types)
  │
  ▼
pkg/clients/hyperfleetdb/
  │
  ├── uses ──► hyperfleetv1alpha1.Cluster  ← actual Go types for DB storage
  │            (embeds hypershiftv1beta1     (from hyperfleet-operator repo)
  │             fields like HostedCluster)
  │
  ▼
PostgreSQL (via pgruntime)
```

### Why v2alpha1 and hyperfleetv1alpha1 Both Exist

| | `hyperfleetv1alpha1` | `api/v2alpha1` |
|---|---|---|
| **Purpose** | Data storage schema (the actual CRD) | Field-level policy annotations for codegen |
| **Consumed by** | `hyperfleetdb.Client`, pgruntime, Kubernetes | `marker-scanner` tool at build time |
| **Contains** | `Cluster.Spec.HostedCluster` (typed struct) | `ClusterSpec.HostedCluster` (annotated mirror) |
| **Runtime usage** | Every request | None — only the generated registry map is used |
| **Maintained in** | `typeid/hyperfleet-operator` repo | `rosa-hyperfleet-api` repo (local) |

These two will likely be unified in the future — the markers could live directly on the operator types, eliminating the local `api/v2alpha1/` package entirely.

---

## Executive Summary

PR #150 adds a field validation system (write-mode enforcement, feature-gate gating, immutability checks) to cluster and nodepool mutation endpoints. It was originally built against `main`, which uses `map[string]interface{}` specs and the old Maestro/Hyperfleet REST client architecture.

This branch ports those changes on top of `pgruntime`, which uses strongly-typed CRD specs (`hyperfleetv1alpha1.ClusterSpec` / `NodePoolSpec`) and a single PostgreSQL-backed `hyperfleetdb.Client`. The port required one key adaptation: a `specToMap` bridge function that serializes typed specs to `map[string]interface{}` via JSON round-trip, allowing the existing validator (which operates on maps) to work with strongly-typed specs.

**Result:** Build passes, all unit tests pass, lint clean, `codegen-verify` target passes.

---

## What Was Ported (New Files — No Conflicts)

These files were copied directly from PR #150 with zero modifications. They have no dependencies on the old Maestro/Hyperfleet client code.

### API Types (`api/v2alpha1/`)

| File | Purpose |
|---|---|
| `cluster_types.go` | `Cluster` / `ClusterSpec` CRD with `+hyperfleet:write-mode` markers |
| `nodepool_types.go` | `NodePool` / `NodePoolSpec` CRD with markers |
| `configuration.go` | `ClusterConfiguration`, `KubeletConfig`, `MachineConfigSpec` with granular per-field markers |
| `hostedclusterspec.passthrough.go` | Generated passthrough of `HostedClusterSpec` / `NodePoolSpec` from upstream HyperShift |
| `zz_generated.passthrough.go.raw` | Raw generated file (pre-curation snapshot) |

These types define the v2alpha1 API surface with field-level annotations:
- `+hyperfleet:write-mode=mutable` — customer can set/update
- `+hyperfleet:write-mode=immutable` — set at creation only
- `+hyperfleet:write-mode=service-set` — platform-managed, rejected if customer attempts to set
- `+openshift:enable:FeatureGate=<gate>` — requires specific feature gate to be enabled

### Codegen Runtime (`internal/codegen/`)

| Package | Files | Purpose |
|---|---|---|
| `registry/` | `field_metadata.go`, `field_metadata.json` | Generated field registry (~120 entries mapping dotted paths to `FieldMeta{WriteMode, FeatureGates}`) |
| `featuregate/` | `types.go`, `registry.go` | `FeatureSet` (Default/TechPreview/DevPreview), `FeatureStage` (GA/TechPreview/DevPreview), gate registry |
| `validation/` | `validator.go`, `validator_test.go`, `example_test.go`, `gated_writemode_test.go` | `Validator` that checks each request field against the registry for write-mode and feature-gate compliance |
| `conversion/` | `cluster.go` | `InjectClusterServiceSet` / `RewriteCloudURLWithID` helpers (not used on pgruntime — cloudUrl/placement injection doesn't apply) |

### Field Validation Middleware (`pkg/middleware/`)

| File | Purpose |
|---|---|
| `field_validation.go` | `FieldValidator` wrapping the codegen `Validator` — provides `ValidateCreate` and `ValidateUpdate` methods |
| `field_validation_test.go` | Tests for the middleware |

### Documentation and Build

| File | Purpose |
|---|---|
| `docs/codegen.md` | Implementation guide: architecture, pending decisions, codegen workflow |
| `Makefile` (modified) | New targets: `codegen-install-tools`, `codegen-passthrough`, `codegen-registry`, `codegen-verify`, `get-hypershift-version` |

---

## What Was Adapted (Modified Files)

### The Core Adaptation: `specToMap` Bridge

The original PR #150 (on `main`) works with `map[string]interface{}` specs natively — the validator consumes them directly. On `pgruntime`, specs are strongly-typed CRD structs.

**Solution:** A `specToMap` helper function added to `pkg/handlers/cluster.go`:

```go
func specToMap(v interface{}) (map[string]interface{}, error) {
    data, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }
    var m map[string]interface{}
    if err := json.Unmarshal(data, &m); err != nil {
        return nil, err
    }
    return m, nil
}
```

This JSON round-trip preserves the field names from struct tags (e.g., `json:"displayName"`) which match the registry's dotted-path keys (e.g., `spec.displayName`). Both handlers use this same function.

### `pkg/handlers/cluster.go`

- Added `fieldValidator *middleware.FieldValidator` to `ClusterHandler` struct
- Changed constructor: `NewClusterHandler(db, oidcIssuerBaseURL, fieldValidator, logger)`
- **Create flow:** After basic field checks (`req.Name == "" || req.Spec == nil`), before name-uniqueness check:
  - Converts `req.Spec` (typed `*hyperfleetv1alpha1.ClusterSpec`) to map via `specToMap`
  - Calls `h.fieldValidator.ValidateCreate(specMap, featuregate.Default, nil)`
  - Returns HTTP 422 with field-level details on validation failure
- **Update flow:** After fetching existing cluster (needed for immutability comparison):
  - Converts both `req.Spec` and `cr.Spec.HostedCluster` (existing) to maps
  - Calls `h.fieldValidator.ValidateUpdate(specMap, existingMap, featuregate.Default, nil)`
- Added `writeValidationError` method returning structured 422 responses with error code `CLUSTERS-MGMT-VALIDATE-001`

### `pkg/handlers/nodepool.go`

- Same pattern as cluster handler:
  - `fieldValidator *middleware.FieldValidator` added to struct
  - Constructor: `NewNodePoolHandler(db, fieldValidator, logger)`
  - Create/Update: `specToMap` + `ValidateCreate`/`ValidateUpdate`
  - `writeValidationError` with code `NODEPOOLS-MGMT-VALIDATE-001`
- Update passes `nil` for existing spec (simpler than cluster — no immutability check against old nodepool data)

### `pkg/server/server.go`

- Creates `fieldValidator := middleware.NewFieldValidator()` once
- Passes to both handler constructors

### `pkg/handlers/cluster_test.go`

- All 18 existing `NewClusterHandler(...)` calls updated to include `nil` as the fieldValidator parameter

### `pkg/handlers/zoa.go` and `test/e2e-cli/cluster_test.go`

- Lint fixes from the pgruntime branch (errcheck: explicit `_ =` for deferred Close and Fprintln calls)

---

## What Was NOT Ported

### `internal/codegen/conversion/cluster.go`

Copied but **not wired into handlers**. On `main`, this package provides:
- `InjectClusterServiceSet(spec, {CloudURL, Placement, CreatorARN})` — injects platform-managed fields into the spec map
- `RewriteCloudURLWithID(spec, baseURL, clusterID)` — rewrites cloudUrl in responses

On `pgruntime`, these operations don't apply because:
- `cloudUrl` / `placement` aren't part of the typed CRD spec
- `creatorARN` is set directly on the typed struct field (`req.Spec.CreatorARN = callerARN`)
- OIDC issuer URL is set directly on the CR (`cr.Spec.HostedCluster.IssuerURL`)

The conversion package is included for completeness and future use but has no call sites.

---

## How Validation Works End-to-End

```
1. HTTP Request (POST /api/v0/clusters)
2. JSON decode into types.ClusterCreateRequest
   req.Spec is *hyperfleetv1alpha1.ClusterSpec (strongly typed)
3. Basic field checks (name, spec not nil)
4. specToMap(req.Spec) → map[string]interface{}
   JSON round-trip preserves struct tag field names
5. FieldValidator.ValidateCreate(specMap, Default, nil)
   → flattenWithPrefix("spec.", specMap)
   → for each "spec.fieldName" key, look up FieldMeta in registry
   → reject service-set fields (e.g., spec.accountId, spec.creatorARN)
   → reject feature-gated fields if gate not enabled
6. On failure: HTTP 422 with details [{field, reason}, ...]
7. On success: continue to create cluster in PostgreSQL
```

---

## Validation Error Response Format

```json
{
  "kind": "Error",
  "code": "CLUSTERS-MGMT-VALIDATE-001",
  "reason": "Validation failed",
  "details": [
    {
      "field": "spec.accountId",
      "reason": "field is service-managed and cannot be set by the caller"
    }
  ]
}
```

HTTP status: **422 Unprocessable Entity**

---

## Decision: Where Should Field Markers Live?

### Background

Today the codebase has two parallel type definitions for the same fields:

| | `hyperfleetv1alpha1` (operator repo) | `api/v2alpha1/` (local) |
|---|---|---|
| Defines | Storage schema (CRD spec/status) | Field access policy (write-mode, feature gates) |
| Used at runtime | Yes — every DB read/write | No — only the generated registry map is used |

One could argue that moving the `+hyperfleet:write-mode` markers onto the operator types would eliminate `api/v2alpha1/` and prevent the two type systems from drifting. However, this benefit is smaller than it appears.

### The Real Maintenance Cost: HyperShift Upstream Bumps

Regardless of where the markers live, the dominant maintenance event is the same: **when `openshift/hypershift` adds, removes, or renames fields in `HostedClusterSpec` or `NodePoolSpec`**, both the operator types and the validation registry need updating. Moving markers from `api/v2alpha1/` to the operator repo doesn't reduce this work — it just moves where you do it.

The drift risk that moving markers would fix — someone adds a field to `hyperfleetv1alpha1` but forgets the corresponding `v2alpha1` entry — is real but minor. An unknown field simply passes through validation unchecked (same as today for any field not in the registry). The `codegen-verify` CI target catches compilation failures from stale generated code, which is the more dangerous class of drift.

### Two Options If We Do Want to Consolidate

#### Option A: Codegen runs in the operator repo

The operator repo owns the markers, runs `marker-scanner`, and publishes the generated registry as a Go package.

```
hyperfleet-operator repo:
  api/v1alpha1/cluster_types.go       ← types WITH markers
  api/v1alpha1/registry/field_meta.go ← generated output (published)

rosa-hyperfleet-api repo:
  import "github.com/typeid/hyperfleet-operator/api/v1alpha1/registry"
  ← consumes published registry, no local codegen
```

| Pros | Cons |
|---|---|
| Single source of truth — types and access policy in one place | Operator repo takes on platform API concerns (write-mode, feature gates) |
| Registry version always matches the type version | Tighter coupling between repos — operator releases gate platform API validation changes |
| Platform API has zero codegen machinery | Operator CI must run `marker-scanner` and verify output |

#### Option B: Codegen runs in the platform API repo, scans operator types

The operator repo adds marker comments to its types but does not run codegen. The platform API runs `marker-scanner` against the imported/vendored operator types at build time.

```
hyperfleet-operator repo:
  api/v1alpha1/cluster_types.go  ← types WITH markers (comments only, no tooling)

rosa-hyperfleet-api repo:
  Makefile: marker-scanner --input-dirs=<vendored operator types>
  internal/codegen/registry/field_metadata.go ← generated locally
```

| Pros | Cons |
|---|---|
| Operator repo change is minimal — just comment annotations, no tools | Platform API controls regeneration timing — could lag behind operator type changes |
| Platform API retains full control over validation policy and release cadence | Must remember to regenerate after bumping the operator dependency |
| No new published packages or cross-repo CI dependencies | Markers on operator types are inert without the platform API's scanner |

### Recommendation: Keep `api/v2alpha1/` For Now

Neither option meaningfully reduces the maintenance burden. The dominant cost — updating types and registry after HyperShift bumps — exists in every scenario. Moving markers to the operator repo is a nice-to-have cleanup if the operator team is amenable, but it's not a priority.

What **is** a priority is the `codegen-verify` CI target, which catches stale generated code at build time. This provides more practical protection against drift than reorganizing where the markers live.

---

## PR #148 Port: OpenAPI Codegen and Swagger UI

### Context

PR #148 ([ROSAENG-61805](https://redhat.atlassian.net/browse/ROSAENG-61805)) builds on PR #150 by adding **Phase 5** of the codegen integration: generating typed OpenAPI schemas from the `api/v2alpha1/` Go types and merging them into the project's `openapi/openapi.yaml`. It also adds a local Swagger UI for browsing the API docs.

PR #148 was built against `main` (Maestro/Hyperfleet REST architecture). This section documents what was ported onto `pgruntime-codegen`, what was adapted, and what was intentionally skipped.

### What This Adds (The Big Picture)

Before this port, the OpenAPI spec (`openapi/openapi.yaml`) described cluster and nodepool specs as opaque `object` types — consumers had no visibility into which fields exist, their types, or their constraints. The codegen OpenAPI integration solves this by:

1. **Running `openapi-gen`** against `api/v2alpha1/` Go types → produces `openapi/generated-schemas.json` (Swagger 2.0 definitions with field names, types, descriptions, and marker annotations)
2. **Running `hack/merge-openapi.sh`** to extract `ClusterSpec` and `NodePoolSpec` from the generated JSON and patch them into `openapi/openapi.yaml` as standalone schemas with `$ref` links
3. **Serving via Swagger UI** (`openapi/swagger-ui/index.html`) for local browsing

This means the OpenAPI spec now documents every visible field on ClusterSpec and NodePoolSpec with proper types, rather than treating specs as freeform objects.

### Files Added

| File | What It Does |
|---|---|
| `hack/merge-openapi.sh` | Shell script that extracts ClusterSpec/NodePoolSpec properties from generated Swagger 2.0 JSON and merges them into the OpenAPI 3.0 YAML. Supports `--keep-markers` flag to preserve Go annotations and include hidden passthrough objects. |
| `openapi/generated-schemas.json` | Generated Swagger 2.0 definitions from `api/v2alpha1/` types. Contains ~20 definitions including ClusterSpec, NodePoolSpec, ClusterStatus, KubeletConfig, MachineConfigSpec, etc. This file is a build artifact — regenerate it with `make codegen-openapi`. |
| `openapi/swagger-ui/index.html` | Single HTML file that loads Swagger UI from CDN and points it at `../openapi.yaml`. Open via `make swagger-ui-serve` + `make swagger-ui-open`. |

### Files Modified

| File | Change |
|---|---|
| `openapi/openapi.yaml` | ClusterSpec and NodePoolSpec extracted as standalone `components/schemas` entries. Cluster, ClusterCreateRequest, and ClusterUpdateRequest now reference ClusterSpec via `$ref` instead of inline definitions. NodePool schemas reference NodePoolSpec similarly. |
| `Makefile` | Added `codegen-openapi` target (runs `openapi-gen` → `merge-openapi.sh`), `swagger-ui-serve` and `swagger-ui-open` targets, `openapi-gen` binary in `codegen-install-tools`, `KEEP_MARKERS` and `VERBOSE` variables, updated help text. |
| `pkg/handlers/cluster_test.go` | Added two new tests: `TestClusterHandler_Create_ValidationRejectsServiceSetField` (verifies service-set fields produce HTTP 422) and `TestClusterHandler_Create_NilValidatorBypasses` (verifies nil validator skips validation). |
| `pkg/handlers/zoa_test.go` | Fixed `resp.Execution.ExecutionID` → `resp.ExecutionID` to match the embedded struct access pattern on `ExecutionResponse`. |

### New Makefile Targets

```bash
make codegen-openapi          # Generate OpenAPI schemas and merge into openapi.yaml
make swagger-ui-serve         # Start local Python HTTP server for Swagger UI
make swagger-ui-open          # Open Swagger UI in browser
make codegen-openapi KEEP_MARKERS=1  # Preserve Go marker annotations in output
make codegen-registry VERBOSE=1      # Run marker-scanner with verbose output
```

### What Was NOT Ported (And Why)

| PR #148 Change | Why It Was Skipped |
|---|---|
| `ci/build-push-image.sh` docker/podman preference | Cosmetic — swaps podman-first to docker-first detection. Not related to codegen. |
| `README.md` docker/podman wording | Same cosmetic change. |
| `pkg/handlers/cluster.go` conversion wiring | PR #148 wires `conversion.InjectClusterServiceSet()` and `conversion.RewriteCloudURLWithID()` into the cluster handler. On pgruntime, these functions operate on `map[string]interface{}` specs and are not applicable — the handler sets `req.Spec.CreatorARN` directly on the typed struct and there is no cloudUrl/placement map injection. |
| `pkg/handlers/zoa_test.go` architecture changes | PR #148 replaces `hyperfleetdb.Client` (fake) with `zoaMockMaestroClient` in ZOA handler tests. This is the reverse of what pgruntime needs — our branch uses `hyperfleetdb` correctly. Only the `resp.ExecutionID` fix was taken. |
| `pkg/zoa/types.go`, `reconciler.go`, `templates_test.go`, `audit_store.go` formatting | Alignment changes against `main`. The pgruntime branch already has correct alignment for these files. |
| `test/e2e-zoa/zoa_test.go` formatting | Same — pgruntime already has correct alignment. |
| `pkg/clients/maestro/client.go`, `client_test.go` | Minor changes to maestro client — not applicable since pgruntime replaces maestro with hyperfleetdb. |
| "Allow mutable fields" handler test | PR #148 tests `{"spec": {"displayName": "My Cluster"}}` to prove mutable fields pass. On pgruntime, `ClusterSpec` is a typed struct without a `displayName` field, so this JSON key is silently dropped during decode. The middleware-level tests (`pkg/middleware/field_validation_test.go`) already validate that mutable map keys pass. |

### Test Adaptation Notes

PR #148's handler tests use the old Hyperfleet REST client (`hyperfleet.NewClient(config.HyperfleetConfig{BaseURL: ...})`). The pgruntime-adapted tests use `fake.NewClientBuilder().WithScheme(scheme).Build()` + `hyperfleetdb.NewClientFrom(fc, logger)` to create an in-memory fake database.

The "reject service-set" test uses `creatorARN` (json:"creatorARN") instead of `accountId` from PR #148. On pgruntime, `accountId` is stored as a Kubernetes label (`hyperfleet.io/account-id`), not a spec field, so it wouldn't appear in `specToMap` output. `creatorARN` is an actual field on `hyperfleetv1alpha1.ClusterSpec` that's marked `service-set` in the codegen registry.

### Limitation: Zero-Value Field Leaking in specToMap

The `specToMap` bridge function (JSON round-trip from typed struct to map) has a subtle interaction with validation on pgruntime: struct fields without `omitempty` tags always appear in the serialized map, even when the customer didn't explicitly set them. For example, `HostedClusterSpec.Platform.Type` has no `omitempty`, so a zero-value `ClusterSpec{}` produces a map containing `spec.hostedCluster.platform.type: ""`. The validator sees this as a customer-provided value and rejects it if it's service-set.

In practice, this means the field validator currently works correctly for **rejecting** explicit service-set fields in customer requests, but enabling it for **all creates** would require one of:

- Filtering zero-value fields out of the map before validation
- Using `json.Decoder` with `DisallowUnknownFields` and a separate "customer input" struct
- Changing the validator to ignore empty/zero values for service-set checks

This is tracked as a follow-up design decision.

---

## Things to Watch

1. **JSON field name alignment:** The validator matches registry keys (e.g., `spec.displayName`) against flattened map keys from `json.Marshal`. This works because the CRD struct tags use the same names as the registry. If a struct tag is renamed, the registry must be regenerated.

2. **Conversion package unused:** `internal/codegen/conversion/cluster.go` has no call sites on pgruntime. It can be removed or adapted if service-set injection is needed in the future.

3. **NodePool update immutability:** NodePool `ValidateUpdate` passes `nil` for existing spec, meaning immutability checks are skipped. If nodepool fields are marked `+hyperfleet:write-mode=immutable`, the existing nodepool spec should be fetched and passed (same pattern as cluster update).

4. **Feature gates default to `Default` (GA only):** All validation calls pass `featuregate.Default`, meaning TechPreview and DevPreview-gated fields are always rejected. To enable them, the feature set would need to come from cluster or account configuration.

5. **No go.mod changes required:** All external dependencies needed by the codegen packages (`openshift/hypershift/api`, `openshift/api`, `k8s.io/api`, `k8s.io/apimachinery`) were already present in the pgruntime go.mod.

6. **Zero-value field leaking in specToMap:** See the "Limitation" subsection under "PR #148 Port" above. The JSON round-trip from typed structs to maps includes zero-value fields without `omitempty`, which can trigger false service-set rejections.

---

## Generation Prompt

The prompt below was used to port PR #148 onto the `pgruntime-codegen` branch. If this work needs to be redone (e.g., after rebasing pgruntime, or when a new codegen PR lands on main), edit and re-run this prompt.

````
Review the changes from PR #148 (https://github.com/openshift-online/rosa-hyperfleet-api/pull/148,
branch ROSAENG-61805) and apply the applicable changes onto the current pgruntime-codegen branch.

### Background

This branch (`pgruntime-codegen`) is based on the `pgruntime` branch, which replaces the old
Maestro + Hyperfleet REST client architecture with a single PostgreSQL-backed `hyperfleetdb.Client`
using typed CRD specs (`hyperfleetv1alpha1.ClusterSpec` / `NodePoolSpec`).

PR #150's codegen field validation (write-mode enforcement, feature gates, immutability checks)
has already been ported onto this branch. That work added:
- `api/v2alpha1/` — annotated type mirrors for codegen input (NOT runtime types)
- `internal/codegen/` — registry, featuregate, validation, conversion packages
- `pkg/middleware/field_validation.go` — FieldValidator wrapping the codegen Validator
- Handler modifications using a `specToMap` JSON round-trip bridge to convert typed specs to maps
- Makefile targets: codegen-install-tools, codegen-passthrough, codegen-registry, codegen-verify

PR #148 adds Phase 5 on top of PR #150: OpenAPI schema generation from Go types, Swagger UI,
and additional Makefile targets. Apply the NEW changes from PR #148 that are not already present.

### Design decisions to respect

1. Keep `api/v2alpha1/` as-is — markers live in the platform API repo, not the operator repo.
2. The `specToMap` bridge pattern is the correct way to connect typed CRD specs to the
   map-based validator on pgruntime.
3. `internal/codegen/conversion/cluster.go` has no call sites on pgruntime because cloudUrl,
   placement, and creatorARN are set directly on typed struct fields.
4. Handler tests use `fake.NewClientBuilder().WithScheme(scheme).Build()` +
   `hyperfleetdb.NewClientFrom(fc, logger)` — NOT the old `hyperfleet.NewClient(config...)`.

### What to apply from PR #148

1. **OpenAPI codegen files** (new, no conflicts):
   - `hack/merge-openapi.sh` — copy from PR #148
   - `openapi/generated-schemas.json` — copy from PR #148
   - `openapi/swagger-ui/index.html` — copy from PR #148
   - `openapi/openapi.yaml` — run `hack/merge-openapi.sh` against the existing pgruntime
     openapi.yaml with the generated-schemas.json (requires `yq`)

2. **Makefile additions**:
   - Add `openapi-gen` to `codegen-install-tools`
   - Add `codegen-openapi` target (runs openapi-gen then merge-openapi.sh)
   - Add `swagger-ui-serve` and `swagger-ui-open` targets
   - Add `KEEP_MARKERS` and `VERBOSE` variables
   - Update `.PHONY` and help text

3. **Handler validation tests** — port to pgruntime:
   - `TestClusterHandler_Create_ValidationRejectsServiceSetField`: use `creatorARN`
     (not `accountId` — it's a label on pgruntime, not a spec field) with typed
     `types.ClusterCreateRequest` and `hyperfleetv1alpha1.ClusterSpec`.
   - `TestClusterHandler_Create_NilValidatorBypasses`: verify nil validator skips validation.
   - Import `"github.com/openshift/rosa-regional-platform-api/pkg/types"` in cluster_test.go.

4. **zoa_test.go fix**: `resp.Execution.ExecutionID` → `resp.ExecutionID` (embedded struct).

### What to skip from PR #148

- `ci/build-push-image.sh`, `README.md` — cosmetic docker/podman preference changes
- `pkg/handlers/cluster.go` conversion wiring — not applicable on pgruntime
- `pkg/handlers/zoa_test.go` architecture changes — these revert pgruntime back to maestro mocks
- `pkg/zoa/types.go`, `reconciler.go`, `reconciler_test.go`, `templates_test.go`,
  `audit_store.go` formatting — pgruntime already has correct alignment
- `test/e2e-zoa/zoa_test.go` formatting — already correct
- `pkg/clients/maestro/` changes — maestro is replaced by hyperfleetdb on pgruntime
- "Allow mutable fields" handler test — `ClusterSpec` has no `displayName` field;
  middleware tests already cover mutable field validation

### Verification

After applying, run:
```
make build          # must pass
make test           # must pass (0 failures)
make lint           # must pass (0 issues)
make codegen-verify # must pass (all packages compile)
```
````
