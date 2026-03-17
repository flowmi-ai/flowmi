package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VideoGenerateRequest struct {
	Model       string          `json:"model,omitempty"`
	Prompt      string          `json:"prompt"`
	Duration    int             `json:"duration,omitempty"`
	AspectRatio string          `json:"aspectRatio,omitempty"`
	Resolution  string          `json:"resolution,omitempty"`
	Image       *ReferenceImage `json:"image,omitempty"`
	VideoURL    string          `json:"videoUrl,omitempty"`
}

type VideoGenerateResponse struct {
	RequestID string `json:"requestId"`
}

type VideoStatusResponse struct {
	Status string      `json:"status"`
	Video  *VideoReady `json:"video,omitempty"`
}

type VideoReady struct {
	URL      string `json:"url"`
	Duration int    `json:"duration"`
}

func (c *Client) GenerateVideo(ctx context.Context, req *VideoGenerateRequest) (*VideoGenerateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/videos/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result VideoGenerateResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding video generate response: %w", err)
	}
	return &result, nil
}

func (c *Client) GetVideoStatus(ctx context.Context, requestID string) (*VideoStatusResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/videos/"+requestID, nil)
	if err != nil {
		return nil, err
	}

	var result VideoStatusResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding video status response: %w", err)
	}
	return &result, nil
}
