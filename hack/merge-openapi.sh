#!/usr/bin/env bash
#
# Merges generated OpenAPI schemas from openapi-gen into the existing openapi.yaml.
#
# Usage: hack/merge-openapi.sh [--keep-markers] <generated-schemas.json> <openapi.yaml>
#
# Flags:
#   --keep-markers  Preserve Go marker annotations (+k8s:, +hyperfleet:, etc.) in descriptions
#                   and include hidden passthrough objects (hostedCluster, nodePool)
#
# The generated JSON contains Swagger 2.0 definitions for the visible API types
# (hidden fields excluded by +k8s:openapi-gen=false markers). This script extracts
# the ClusterSpec and NodePoolSpec definitions and patches them into the
# corresponding spec: properties in the existing OpenAPI 3.0 YAML.
#
# ClusterSpec and NodePoolSpec are created as standalone schemas under
# components.schemas so they appear in Swagger UI. The Cluster,
# ClusterCreateRequest, and ClusterUpdateRequest schemas reference them via $ref.

set -euo pipefail

KEEP_MARKERS=false
if [[ "${1:-}" == "--keep-markers" ]]; then
    KEEP_MARKERS=true
    shift
fi

GENERATED="${1:?Usage: $0 [--keep-markers] <generated-schemas.json> <openapi.yaml>}"
OPENAPI="${2:?Usage: $0 [--keep-markers] <generated-schemas.json> <openapi.yaml>}"

if ! command -v yq &>/dev/null; then
    echo "Error: yq is required. Install with: brew install yq" >&2
    exit 1
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Strip marker lines (+hyperfleet:..., +kubebuilder:..., +openshift:...) from JSON description strings
clean_markers() {
    if [[ "$KEEP_MARKERS" == "true" ]]; then
        cat
    else
        sed -E 's/\\n\+[^"]+//g'
    fi
}

# Extract ClusterSpec properties
if [[ "$KEEP_MARKERS" == "true" ]]; then
    yq eval -o=json '.definitions.ClusterSpec.properties' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/cluster-spec.yaml"
else
    yq eval -o=json '
      .definitions.ClusterSpec.properties
      | to_entries
      | map(select(.key != "hostedCluster"))
      | from_entries
    ' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/cluster-spec.yaml"
fi

# Extract NodePoolSpec properties
if [[ "$KEEP_MARKERS" == "true" ]]; then
    yq eval -o=json '.definitions.NodePoolSpec.properties' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/nodepool-spec.yaml"
else
    yq eval -o=json '
      .definitions.NodePoolSpec.properties
      | to_entries
      | map(select(.key != "nodePool"))
      | from_entries
    ' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/nodepool-spec.yaml"
fi

# Extract ClusterReference for inline use
yq eval -o=json '.definitions.ClusterReference' "$GENERATED" | clean_markers | yq eval -P '.' > "$TMPDIR/cluster-ref.yaml"

# --- Import all generated definitions as standalone schemas (dev mode) ---
if [[ "$KEEP_MARKERS" == "true" ]]; then
    # Extract every definition from the generated Swagger 2.0 JSON, excluding
    # top-level types already handled (ClusterSpec, NodePoolSpec, ClusterReference)
    SKIP_DEFS="ClusterSpec NodePoolSpec ClusterReference"
    for def in $(yq eval -r '.definitions | keys | .[]' "$GENERATED"); do
        skip=false
        for s in $SKIP_DEFS; do
            if [[ "$def" == "$s" ]]; then skip=true; break; fi
        done
        if [[ "$skip" == "true" ]]; then continue; fi

        yq eval -o=json ".definitions.\"${def}\"" "$GENERATED" \
            | clean_markers \
            | yq eval -P '.' > "$TMPDIR/def-${def}.yaml"

        yq eval -i "
          .components.schemas.\"${def}\" = load(\"$TMPDIR/def-${def}.yaml\")
        " "$OPENAPI"
    done

fi

# --- Create standalone ClusterSpec schema ---
yq eval -i '
  .components.schemas.ClusterSpec = {
    "type": "object",
    "description": "Cluster specification defining desired state",
    "additionalProperties": true
  }
' "$OPENAPI"

yq eval -i "
  .components.schemas.ClusterSpec.properties = load(\"$TMPDIR/cluster-spec.yaml\")
" "$OPENAPI"

yq eval -i '
  .components.schemas.ClusterSpec.properties.cloudUrl = {
    "type": "string",
    "readOnly": true,
    "description": "CloudFront URL with cluster ID (auto-populated by server)",
    "example": "https://doku78iof5s87.cloudfront.net/cluster-123"
  }
  | .components.schemas.ClusterSpec.properties.placement = {
    "type": "string",
    "description": "Management cluster name (auto-populated if not provided)",
    "example": "management-cluster-us-east-1"
  }
' "$OPENAPI"

# --- Patch Cluster.spec to $ref ClusterSpec ---
yq eval -i '
  .components.schemas.Cluster.properties.spec = {
    "$ref": "#/components/schemas/ClusterSpec"
  }
' "$OPENAPI"

# --- Patch ClusterCreateRequest.spec to allOf ClusterSpec (without cloudUrl) ---
yq eval -i '
  .components.schemas.ClusterCreateRequest.properties.spec = {
    "allOf": [
      {"$ref": "#/components/schemas/ClusterSpec"}
    ],
    "description": "Cluster specification (cloudUrl is server-populated and ignored on create)"
  }
' "$OPENAPI"

# --- Patch ClusterUpdateRequest.spec to allOf ClusterSpec ---
yq eval -i '
  .components.schemas.ClusterUpdateRequest.properties.spec = {
    "allOf": [
      {"$ref": "#/components/schemas/ClusterSpec"}
    ],
    "description": "Cluster specification (mutable fields only)"
  }
' "$OPENAPI"

# --- Patch NodePoolSpec as standalone schema ---
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

# Rewrite $ref paths from Swagger 2.0 (#/definitions/X) to OpenAPI 3.0 (#/components/schemas/X)
if [[ "$KEEP_MARKERS" == "true" ]]; then
    sed -i '' "s|#/definitions/|#/components/schemas/|g" "$OPENAPI"
fi

echo "Merged generated schemas into $OPENAPI"
