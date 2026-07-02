package client

import (
	"context"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

// CreateWork creates a manifestwork for ZOA trusted actions
func (c *Client) CreateWork(ctx context.Context, req *types.WorkRequest) (*types.Work, error) {
	var work types.Work

	err := c.doRequest(ctx, "POST", "/api/v0/work", req, &work)
	if err != nil {
		return nil, err
	}

	return &work, nil
}
