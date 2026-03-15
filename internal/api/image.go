package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ImageGenerateRequest struct {
	Model       string           `json:"model,omitempty"`
	Prompt      string           `json:"prompt"`
	Images      []*ReferenceImage `json:"images,omitempty"`
	AspectRatio string           `json:"aspectRatio,omitempty"`
	ImageSize   string           `json:"imageSize,omitempty"`
}

type ReferenceImage struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type ImageGenerateResponse struct {
	Image    string `json:"image"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
}

func (c *Client) GenerateImage(ctx context.Context, req *ImageGenerateRequest) (*ImageGenerateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/images/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result ImageGenerateResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding image generate response: %w", err)
	}
	return &result, nil
}
