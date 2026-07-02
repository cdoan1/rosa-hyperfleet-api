package client

import (
	"context"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

// CheckAuthorization checks if an action is authorized
func (c *Client) CheckAuthorization(ctx context.Context, req *types.CheckAuthorizationRequest) (*types.CheckAuthorizationResponse, error) {
	var response types.CheckAuthorizationResponse

	err := c.doRequest(ctx, "POST", "/api/v0/authz/check", req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
