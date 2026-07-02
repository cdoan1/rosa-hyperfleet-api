package client

import (
	"context"
	"fmt"

	"github.com/openshift-online/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListAccounts lists all accounts
func (c *Client) ListAccounts(ctx context.Context) ([]types.Account, error) {
	var result struct {
		Items []types.Account `json:"items"`
	}

	err := c.doRequest(ctx, "GET", "/api/v0/accounts", nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

// GetAccount gets an account by ID
func (c *Client) GetAccount(ctx context.Context, accountID string) (*types.Account, error) {
	var account types.Account

	path := fmt.Sprintf("/api/v0/accounts/%s", accountID)
	err := c.doRequest(ctx, "GET", path, nil, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// CreateAccount creates a new account
func (c *Client) CreateAccount(ctx context.Context, req *types.EnableAccountRequest) (*types.Account, error) {
	var account types.Account

	err := c.doRequest(ctx, "POST", "/api/v0/accounts", req, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// DeleteAccount deletes an account
func (c *Client) DeleteAccount(ctx context.Context, accountID string) error {
	path := fmt.Sprintf("/api/v0/accounts/%s", accountID)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}
