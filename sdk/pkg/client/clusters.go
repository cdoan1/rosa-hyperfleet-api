package client

import (
	"context"
	"fmt"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListClusters lists all clusters
func (c *Client) ListClusters(ctx context.Context) ([]types.Cluster, error) {
	var result struct {
		Items []types.Cluster `json:"items"`
	}

	err := c.doRequest(ctx, "GET", "/api/v0/clusters", nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

// GetCluster gets a cluster by ID
func (c *Client) GetCluster(ctx context.Context, clusterID string) (*types.Cluster, error) {
	var cluster types.Cluster

	path := fmt.Sprintf("/api/v0/clusters/%s", clusterID)
	err := c.doRequest(ctx, "GET", path, nil, &cluster)
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}

// CreateCluster creates a new cluster
func (c *Client) CreateCluster(ctx context.Context, req *types.ClusterCreateRequest) (*types.Cluster, error) {
	var cluster types.Cluster

	err := c.doRequest(ctx, "POST", "/api/v0/clusters", req, &cluster)
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}

// UpdateCluster updates a cluster (PATCH)
func (c *Client) UpdateCluster(ctx context.Context, clusterID string, req *types.ClusterUpdateRequest) (*types.Cluster, error) {
	var cluster types.Cluster

	path := fmt.Sprintf("/api/v0/clusters/%s", clusterID)
	err := c.doRequest(ctx, "PATCH", path, req, &cluster)
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}

// DeleteCluster deletes a cluster
func (c *Client) DeleteCluster(ctx context.Context, clusterID string) error {
	path := fmt.Sprintf("/api/v0/clusters/%s", clusterID)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// GetClusterStatus gets a cluster's status
func (c *Client) GetClusterStatus(ctx context.Context, clusterID string) (*types.ClusterStatusResponse, error) {
	var status types.ClusterStatusResponse

	path := fmt.Sprintf("/api/v0/clusters/%s/statuses", clusterID)
	err := c.doRequest(ctx, "GET", path, nil, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}
