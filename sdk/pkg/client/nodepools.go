package client

import (
	"context"
	"fmt"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListNodePools lists all nodepools
func (c *Client) ListNodePools(ctx context.Context) ([]types.NodePool, error) {
	var result struct {
		Items []types.NodePool `json:"items"`
	}

	err := c.doRequest(ctx, "GET", "/api/v0/nodepools", nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

// GetNodePool gets a nodepool by ID
func (c *Client) GetNodePool(ctx context.Context, nodepoolID string) (*types.NodePool, error) {
	var nodepool types.NodePool

	path := fmt.Sprintf("/api/v0/nodepools/%s", nodepoolID)
	err := c.doRequest(ctx, "GET", path, nil, &nodepool)
	if err != nil {
		return nil, err
	}

	return &nodepool, nil
}

// CreateNodePool creates a new nodepool
func (c *Client) CreateNodePool(ctx context.Context, req *types.NodePoolCreateRequest) (*types.NodePool, error) {
	var nodepool types.NodePool

	err := c.doRequest(ctx, "POST", "/api/v0/nodepools", req, &nodepool)
	if err != nil {
		return nil, err
	}

	return &nodepool, nil
}

// UpdateNodePool updates a nodepool
func (c *Client) UpdateNodePool(ctx context.Context, nodepoolID string, req *types.NodePoolUpdateRequest) (*types.NodePool, error) {
	var nodepool types.NodePool

	path := fmt.Sprintf("/api/v0/nodepools/%s", nodepoolID)
	err := c.doRequest(ctx, "PUT", path, req, &nodepool)
	if err != nil {
		return nil, err
	}

	return &nodepool, nil
}

// DeleteNodePool deletes a nodepool
func (c *Client) DeleteNodePool(ctx context.Context, nodepoolID string) error {
	path := fmt.Sprintf("/api/v0/nodepools/%s", nodepoolID)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// GetNodePoolStatus gets a nodepool's status
func (c *Client) GetNodePoolStatus(ctx context.Context, nodepoolID string) (*types.NodePoolStatusResponse, error) {
	var status types.NodePoolStatusResponse

	path := fmt.Sprintf("/api/v0/nodepools/%s/status", nodepoolID)
	err := c.doRequest(ctx, "GET", path, nil, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}
