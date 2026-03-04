package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

func (c *Client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/credits/balance", nil)
	if err != nil {
		return nil, err
	}

	var balance BalanceResponse
	if err := json.Unmarshal(resp.Data, &balance); err != nil {
		return nil, fmt.Errorf("decoding balance: %w", err)
	}
	return &balance, nil
}
