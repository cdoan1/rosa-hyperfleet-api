#!/bin/bash

# Cleanup script to delete all OIDC, VPC, and IAM resources created by rosactl
# Usage: ./cleanup-all-resources.sh [region]
#
# Environment variables:
#   ROSACTL_BIN - Path to rosactl binary (default: rosactl)
#   AWS_REGION - AWS region (default: us-east-1)
#   CUSTOMER_AWS_PROFILE - AWS profile to use (default: rrp-customer)

set -e

ROSACTL_BIN="${ROSACTL_BIN:-rosactl}"
AWS_REGION="${AWS_REGION:-${1:-us-east-1}}"
CUSTOMER_AWS_PROFILE="${CUSTOMER_AWS_PROFILE:-rrp-customer}"

export AWS_PROFILE="${CUSTOMER_AWS_PROFILE}"

echo "=== Cleanup Resources ==="
echo "Region: ${AWS_REGION}"
echo "AWS Profile: ${CUSTOMER_AWS_PROFILE}"
echo "ROSACTL: ${ROSACTL_BIN}"
echo ""

# Function to delete OIDC configs
delete_oidc_configs() {
    echo "=== Deleting OIDC Configs ==="

    # List OIDC configs and extract cluster names
    OIDC_LIST=$("${ROSACTL_BIN}" cluster-oidc list --region "${AWS_REGION}" 2>&1 || true)

    if [ -z "${OIDC_LIST}" ]; then
        echo "No OIDC configs found or error listing"
        return
    fi

    # Parse cluster names from the list (assuming format similar to vpc/iam list)
    # This is a simple grep-based extraction - adjust based on actual output format
    CLUSTER_NAMES=$(echo "${OIDC_LIST}" | grep -v "^NAME\|^---\|^$" | awk '{print $1}' || true)

    if [ -z "${CLUSTER_NAMES}" ]; then
        echo "No OIDC configs to delete"
        return
    fi

    for cluster_name in ${CLUSTER_NAMES}; do
        echo "Deleting OIDC config for: ${cluster_name}"
        "${ROSACTL_BIN}" cluster-oidc delete "${cluster_name}" --region "${AWS_REGION}" || echo "Failed to delete OIDC for ${cluster_name}"
    done
    echo ""
}

# Function to delete VPCs
delete_vpcs() {
    echo "=== Deleting VPCs ==="

    # List VPCs and extract cluster names
    VPC_LIST=$("${ROSACTL_BIN}" cluster-vpc list --region "${AWS_REGION}" 2>&1 || true)

    if [ -z "${VPC_LIST}" ]; then
        echo "No VPCs found or error listing"
        return
    fi

    # Parse cluster names from the list
    CLUSTER_NAMES=$(echo "${VPC_LIST}" | grep -v "^NAME\|^---\|^$" | awk '{print $1}' || true)

    if [ -z "${CLUSTER_NAMES}" ]; then
        echo "No VPCs to delete"
        return
    fi

    for cluster_name in ${CLUSTER_NAMES}; do
        echo "Deleting VPC for: ${cluster_name}"
        "${ROSACTL_BIN}" cluster-vpc delete "${cluster_name}" --region "${AWS_REGION}" || echo "Failed to delete VPC for ${cluster_name}"
    done
    echo ""
}

# Function to delete IAM resources
delete_iam_resources() {
    echo "=== Deleting IAM Resources ==="

    # List IAM resources and extract cluster names
    IAM_LIST=$("${ROSACTL_BIN}" cluster-iam list --region "${AWS_REGION}" 2>&1 || true)

    if [ -z "${IAM_LIST}" ]; then
        echo "No IAM resources found or error listing"
        return
    fi

    # Parse cluster names from the list
    CLUSTER_NAMES=$(echo "${IAM_LIST}" | grep -v "^NAME\|^---\|^$" | awk '{print $1}' || true)

    if [ -z "${CLUSTER_NAMES}" ]; then
        echo "No IAM resources to delete"
        return
    fi

    for cluster_name in ${CLUSTER_NAMES}; do
        echo "Deleting IAM for: ${cluster_name}"
        "${ROSACTL_BIN}" cluster-iam delete "${cluster_name}" --region "${AWS_REGION}" || echo "Failed to delete IAM for ${cluster_name}"
    done
    echo ""
}

# Delete in reverse order of creation: OIDC -> VPC -> IAM
# Note: In practice, you might need to adjust this order based on dependencies
delete_oidc_configs
sleep 2
delete_vpcs
sleep 2
delete_iam_resources

echo "=== Cleanup Complete ==="
