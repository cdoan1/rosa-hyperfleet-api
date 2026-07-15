#!/usr/bin/env bash
#
# Merges generated OpenAPI schemas from openapi-gen into the existing openapi.yaml.
#
# Usage: hack/merge-openapi.sh <generated-schemas.json> <openapi.yaml>
#
# The generated JSON contains Swagger 2.0 definitions for the visible API types
# (hidden fields excluded by +k8s:openapi-gen=false markers). This script extracts
# the ClusterSpec and NodePoolSpec definitions and patches them into the
# corresponding spec: properties in the existing OpenAPI 3.0 YAML.

set -euo pipefail

GENERATED="${1:?Usage: $0 <generated-schemas.json> <openapi.yaml>}"
OPENAPI="${2:?Usage: $0 <generated-schemas.json> <openapi.yaml>}"

if ! command -v yq &>/dev/null; then
    echo "Error: yq is required. Install with: brew install yq" >&2
    exit 1
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Strip marker lines (+hyperfleet:..., +kubebuilder:..., +openshift:...) from JSON description strings
clean_markers() {
    sed -E 's/\\n\+[^"]+//g'
}

# Extract ClusterSpec visible properties (excluding hostedCluster which is all hidden)
yq eval -o=json '
  .definitions.ClusterSpec.properties
  | to_entries
  | map(select(.key != "hostedCluster"))
  | from_entries
' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/cluster-spec.yaml"

# Extract NodePoolSpec visible properties (excluding nodePool which is all hidden)
yq eval -o=json '
  .definitions.NodePoolSpec.properties
  | to_entries
  | map(select(.key != "nodePool"))
  | from_entries
' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/nodepool-spec.yaml"

# Extract ClusterReference for inline use
yq eval -o=json '.definitions.ClusterReference' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/cluster-ref.yaml"

# --- Patch Cluster.spec ---
yq eval -i '
  .components.schemas.Cluster.properties.spec = {
    "type": "object",
    "description": "Cluster specification",
    "additionalProperties": true
  }
' "$OPENAPI"

yq eval -i "
  .components.schemas.Cluster.properties.spec.properties = load(\"$TMPDIR/cluster-spec.yaml\")
" "$OPENAPI"

yq eval -i '
  .components.schemas.Cluster.properties.spec.properties.cloudUrl = {
    "type": "string",
    "readOnly": true,
    "description": "CloudFront URL with cluster ID (auto-populated by server)",
    "example": "https://doku78iof5s87.cloudfront.net/cluster-123"
  }
  | .components.schemas.Cluster.properties.spec.properties.placement = {
    "type": "string",
    "description": "Management cluster name (auto-populated if not provided)",
    "example": "management-cluster-us-east-1"
  }
' "$OPENAPI"

# --- Patch ClusterCreateRequest.spec ---
yq eval -i '
  .components.schemas.ClusterCreateRequest.properties.spec = {
    "type": "object",
    "description": "Cluster specification",
    "additionalProperties": true
  }
' "$OPENAPI"

yq eval -i "
  .components.schemas.ClusterCreateRequest.properties.spec.properties = load(\"$TMPDIR/cluster-spec.yaml\")
" "$OPENAPI"

yq eval -i '
  .components.schemas.ClusterCreateRequest.properties.spec.properties.placement = {
    "type": "string",
    "description": "Management cluster name (auto-populated if not provided)",
    "example": "management-cluster-us-east-1"
  }
' "$OPENAPI"

# --- Patch ClusterUpdateRequest.spec ---
yq eval -i '
  .components.schemas.ClusterUpdateRequest.properties.spec = {
    "type": "object",
    "description": "Cluster specification (mutable fields only)",
    "additionalProperties": true
  }
' "$OPENAPI"

yq eval -i "
  .components.schemas.ClusterUpdateRequest.properties.spec.properties = load(\"$TMPDIR/cluster-spec.yaml\")
" "$OPENAPI"

# --- Patch NodePoolSpec ---
yq eval -i '
  .components.schemas.NodePoolSpec = {
    "type": "object",
    "description": "NodePool specification defining desired state",
    "additionalProperties": true
  }
' "$OPENAPI"

yq eval -i "
  .components.schemas.NodePoolSpec.properties = load(\"$TMPDIR/nodepool-spec.yaml\")
" "$OPENAPI"

# Inline ClusterReference into clusterRef (avoid $ref to external type)
yq eval -i "
  .components.schemas.NodePoolSpec.properties.clusterRef = load(\"$TMPDIR/cluster-ref.yaml\")
" "$OPENAPI"

echo "Merged generated schemas into $OPENAPI"
