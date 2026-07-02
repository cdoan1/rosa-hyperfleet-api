package client

import (
	"context"
	"fmt"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListManagementClusters lists all management clusters
func (c *Client) ListManagementClusters(ctx context.Context) ([]types.ManagementCluster, error) {
	var result struct {
		Items []types.ManagementCluster `json:"items"`
	}

	err := c.doRequest(ctx, "GET", "/api/v0/management_clusters", nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

// GetManagementCluster gets a management cluster by ID
func (c *Client) GetManagementCluster(ctx context.Context, clusterID string) (*types.ManagementCluster, error) {
	var cluster types.ManagementCluster

	path := fmt.Sprintf("/api/v0/management_clusters/%s", clusterID)
	err := c.doRequest(ctx, "GET", path, nil, &cluster)
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}

// CreateManagementCluster creates a new management cluster
func (c *Client) CreateManagementCluster(ctx context.Context, req *types.ManagementClusterRequest) (*types.ManagementCluster, error) {
	var cluster types.ManagementCluster

	err := c.doRequest(ctx, "POST", "/api/v0/management_clusters", req, &cluster)
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}
