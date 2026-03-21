package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Ticket struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type CreateTicketRequest struct {
	Type    string `json:"type"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (c *Client) CreateTicket(ctx context.Context, ticketType, subject, message string) (*Ticket, error) {
	body, err := json.Marshal(&CreateTicketRequest{
		Type:    ticketType,
		Subject: subject,
		Message: message,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tickets", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result Ticket
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding ticket response: %w", err)
	}
	return &result, nil
}
