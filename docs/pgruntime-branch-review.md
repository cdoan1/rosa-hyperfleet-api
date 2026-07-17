# pgruntime Branch Review

**Branch:** `typeid/rosa-hyperfleet-api:pgruntime`
**Diff vs main:** 68 files changed, 3,562 additions, 9,510 deletions (net -5,948 lines)
**Date:** 2026-07-17

---

## Executive Summary

This branch replaces the two-service backend architecture (Maestro + Hyperfleet REST API) with a single PostgreSQL-backed client (`hyperfleetdb`) that implements the Kubernetes `controller-runtime` `client.Client` interface. The result is a simpler system with fewer network hops, strongly-typed CRD specs instead of `map[string]interface{}`, and no dependency on Maestro, OCM, CloudEvents, or gRPC.

The change touches every layer: data types, client code, API handlers, ZOA reconciliation, server wiring, build/deploy infrastructure, and E2E tests.

---

## 1. Architecture Change: Before vs After

### Before (main)

```
API Request
  --> ClusterHandler
        --> hyperfleet.Client (HTTP REST) --> Hyperfleet API service
        --> maestro.Client (REST + gRPC)  --> Maestro service --> ManifestWork (OCM)
```

- Two upstream services with independent failure modes
- Specs stored as `map[string]interface{}` (untyped)
- gRPC used for ManifestWork distribution via OCM/CloudEvents
- AWS identity forwarded via HTTP headers (`X-Amz-Account-Id`)

### After (pgruntime)

```
API Request
  --> ClusterHandler
        --> hyperfleetdb.Client (pgruntime) --> PostgreSQL
```

- Single PostgreSQL backend via `pgruntime` library
- Specs are strongly-typed CRD structs (`hyperfleetv1alpha1.ClusterSpec`, `NodePoolSpec`)
- Account isolation enforced via label matching (`hyperfleet.io/account-id`)
- No gRPC, no CloudEvents, no intermediate services

---

## 2. New Client Layer: `pkg/clients/hyperfleetdb/`

### Core Client (`client.go`)

Uses `pgruntime.NewClient` from `github.com/jmelis/postgres-controller-backend/pkg/pgruntime`, which provides a Kubernetes `client.Client` interface backed by PostgreSQL instead of etcd. The client:

- Registers `hyperfleetv1alpha1` types (from `github.com/typeid/hyperfleet-operator/api/v1alpha1`) into a `runtime.Scheme`
- Supports **bucket sharding** via `bucket.All()` / `bucket.Assigner()` to match the operator's `BUCKET_COUNT` for watch compatibility
- Marks `ManagementCluster` as an **unsharded GVK** (global scope)
- Scopes all queries by `hyperfleet.io/account-id` label via `client.MatchingLabels`
- Maps cluster UUIDs to Kubernetes namespaces: `clusterNamespace(id)` returns `"cluster-" + id`
- Provides CRUD for: `Cluster`, `NodePool`, `Manifest`, `ManagementCluster`
- Offers `NewClientFrom(c client.Client, logger)` for injecting fakes in unit tests

### Type Conversions (`convert.go`)

Bidirectional conversion between `hyperfleetv1alpha1` CRs and `types.Cluster` / `types.NodePool`:

| Function | Direction |
|---|---|
| `ClusterCRToPlatform` | CR --> platform type |
| `PlatformCreateToClusterCR` | create request --> CR |
| `ApplyPlatformUpdateToClusterCR` | update request --> mutate existing CR |
| `NodePoolCRToPlatform` | CR --> platform type |
| `PlatformCreateToNodePoolCR` | create request --> CR |
| `ApplyPlatformUpdateToNodePoolCR` | update request --> mutate existing CR |
| `ClusterStatusFromCR` / `NodePoolStatusFromCR` | CR --> status response |

Key mapping conventions:
- `cr.Namespace` minus `"cluster-"` prefix = `cluster.ID`
- `cr.Spec.HostedCluster.IssuerURL` = `cluster.OIDCIssuerURL`
- `cr.Status.PlacementRef` / `cr.Status.ControlPlaneEndpoint` mapped to new platform types

### Removed Clients

| Client | Lines Removed | What It Did |
|---|---|---|
| `pkg/clients/hyperfleet/` | ~1,030 | HTTP REST client to standalone Hyperfleet API service |
| `pkg/clients/maestro/` | ~1,305 | REST + gRPC client to OCM Maestro for ManifestWork distribution |

---

## 3. Type System Changes (`pkg/types/`)

The most impactful design change -- specs are now compile-time checked:

| Field | Before | After |
|---|---|---|
| `Cluster.Spec` | `map[string]interface{}` | `hyperfleetv1alpha1.ClusterSpec` |
| `NodePool.Spec` | local `NodePoolSpec` struct with untyped maps | `hyperfleetv1alpha1.NodePoolSpec` |

New fields added to platform types:
- `Cluster.OIDCIssuerURL string`
- `ClusterStatusInfo.ControlPlaneEndpoint *APIEndpoint`
- `ClusterStatusInfo.Version string`
- `ClusterStatusInfo.PlacementRef *PlacementReference`
- New types: `APIEndpoint{Host, Port}`, `PlacementReference{Name, ManagementCluster}`

---

## 4. Handler Changes

### Modified Handlers

**ClusterHandler** (`pkg/handlers/cluster.go`):
- Dependencies: `hyperfleet.Client` + `maestro.Client` --> `hyperfleetdb.Client` + `oidcIssuerBaseURL` string
- `Create`: generates `uuid.New()` locally, computes OIDC issuer URL as `oidcIssuerBaseURL + "/" + clusterID`, validates name uniqueness by listing existing clusters
- `Update`: read-modify-write pattern (fetch CR, apply changes via `ApplyPlatformUpdateToClusterCR`, preserve existing `IssuerURL`, write back)
- `List`: drops `status` query param, fetches all clusters and paginates in-memory
- `Delete`: drops `force` query parameter

**NodePoolHandler** (`pkg/handlers/nodepool.go`):
- Validates parent cluster exists via `db.GetCluster()` before creating
- Same read-modify-write update pattern as clusters
- In-memory pagination on list

**ManagementClusterHandler** (`pkg/handlers/management_cluster.go`):
- New request/response types: `ManagementClusterCreateRequest{id, region, accountId}` and `ManagementClusterResponse`
- `Create`: builds a `hyperfleetv1alpha1.ManagementCluster` CR directly (no longer proxies to Maestro `CreateConsumer`)
- Returns `409 Conflict` for duplicates (new behavior)

### Removed Handlers

| Handler | Route | Why Removed |
|---|---|---|
| `resource_bundle.go` | `GET/DELETE /api/v0/resource_bundles` | Maestro-specific; no equivalent in pgruntime model |
| `work.go` | `POST /api/v0/work` | Forwarded ManifestWork to Maestro via gRPC; replaced by direct Manifest CR creation |

---

## 5. ZOA (Trusted Actions) Changes

### Reconciler (`pkg/zoa/reconciler.go`)
- Replaces `maestro.ClientInterface` with `hyperfleetdb.Client`
- Calls `db.GetManifest()` / `db.DeleteManifest()` instead of Maestro gRPC
- **New status parsing**: `parseManifestStatus()` replaces `parseManifestWorkStatus()` -- iterates `Manifest.Status.ResourceStatuses[]`, unmarshals each watched resource's `.Status.Raw` into a `partialJobStatus` struct, matches by resource name (`zoa-{execID}`)
- **Terminal failure detection**: uses Kubernetes Job conditions (`type=Failed, status=True`) instead of raw `.status.failed` counter -- correctly distinguishes retries from terminal backoff-limit failures
- **Ordering fix**: the `applied` status update now returns early before checking `fullyCompleted()`, preventing the completion handler from firing on the same reconcile tick

### Job Builder (`pkg/zoa/jobbuilder.go`)
- `BuildManifestWork()` renamed to `BuildManifest()`, returns `*hyperfleetv1alpha1.Manifest`
- Resources wrapped as `hyperfleetv1alpha1.ResourceTemplate{Resource, Content, Watch}` instead of ManifestWork configs
- Complex `ManifestConfigs` with JSONPath feedback rules replaced by simple `Watch: true` on Job resources
- Job `backoffLimit` changed from 0 to 5 (allows retries)
- Manifest targets a cluster via `Spec.ManagementCluster` field instead of namespace convention

---

## 6. Server & Startup Changes

### `main.go`
- **Removed flags**: `--maestro-url`, `--maestro-grpc-url`, `--hyperfleet-url`
- **Added flags**: `--postgres-dsn` (required, fallback to `POSTGRES_DSN` env), `--oidc-issuer-base-url`
- AWS region auto-detected via SDK (`awsconfig.LoadDefaultConfig`)
- Creates `hyperfleetdb.NewClient(ctx, dsn, bucketCount, logger)` with `BUCKET_COUNT` env var (default 1)

### `server.go`
- `server.New()` accepts `dbClient *hyperfleetdb.Client` instead of separate Maestro/Hyperfleet clients
- Routes for `/api/v0/resource_bundles` and `/api/v0/work` removed entirely

---

## 7. Config Changes (`pkg/config/config.go`)

| Removed | Added |
|---|---|
| `MaestroConfig{BaseURL, GRPCBaseURL, Timeout}` | `DBConfig{DSN string}` |
| `HyperfleetConfig{BaseURL, Timeout}` | `RegionalConfig{OIDCIssuerBaseURL string}` |

---

## 8. Build & Infrastructure Changes

### Dockerfile
- **Builder**: `ubi9/go-toolset:1.26.4` --> `golang:1.26-alpine`
- **Runtime**: `ubi9/ubi-minimal:9.8` --> `gcr.io/distroless/static-debian12:nonroot`
- Port: 8081 --> 8080 (Envoy sidecar no longer owns 8080)
- Red Hat container labels and compliance metadata removed

### Removed Infrastructure (2,047 lines)
- Both Tekton/Konflux CI pipelines (`.tekton/`)
- Full Helm chart (`deployment/helm/rosa-regional-frontend/`)
- ArgoCD Application manifests (`deployment/argocd/`)
- Standalone Kubernetes manifests with Envoy sidecar (`deployment/manifests/api.yaml`)

No replacement deployment manifests are included -- the service is expected to be deployed via a different mechanism.

### Makefile
- Container engine hardcoded to `docker` (was auto-detecting `podman` first)

### go.mod
- Go version: 1.25.4 --> 1.26.4
- **Added**: `postgres-controller-backend`, `hyperfleet-operator/api`, `hypershift/api`, `controller-runtime`, `pgx/v5`
- **Removed**: `maestro`, `open-cluster-management.io/*`, `cloudevents/sdk-go`, `grpc`, `sentry-go`, `opentelemetry`, `zap`

---

## 9. Things to Watch

1. **In-memory pagination**: All `List` operations fetch everything from the DB then slice in Go. This works for small datasets but will need server-side pagination at scale.
2. **No deployment manifests**: The branch strips all Helm/ArgoCD/Tekton infrastructure with no replacement -- deployment method needs to be established.
3. **No observability stack**: OpenTelemetry, Sentry, and Zap are removed. Logging relies on `slog` only.
4. **Non-UBI images**: The shift from Red Hat UBI to Alpine/distroless drops FedRAMP-relevant container provenance. This will need to be reconciled for production.
5. **Bucket sharding**: The `BUCKET_COUNT` default is 1. This must match the operator's configuration to ensure watch compatibility.
6. **OIDC issuer URL**: Now computed locally from a base URL flag rather than discovered from management cluster state. The flag must be correctly configured per region.
