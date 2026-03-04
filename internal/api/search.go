package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type WebSearchRequest struct {
	Q    string `json:"q"`
	GL   string `json:"gl,omitempty"`
	HL   string `json:"hl,omitempty"`
	Num  int    `json:"num,omitempty"`
	Page int    `json:"page,omitempty"`
}

type ImageSearchRequest struct {
	Q    string `json:"q"`
	GL   string `json:"gl,omitempty"`
	HL   string `json:"hl,omitempty"`
	Num  int    `json:"num,omitempty"`
	Page int    `json:"page,omitempty"`
	TBS  string `json:"tbs,omitempty"`
}

type NewsSearchRequest struct {
	Q    string `json:"q"`
	GL   string `json:"gl,omitempty"`
	HL   string `json:"hl,omitempty"`
	Num  int    `json:"num,omitempty"`
	Page int    `json:"page,omitempty"`
	TBS  string `json:"tbs,omitempty"`
}

type ScrapeRequest struct {
	URL             string `json:"url"`
	IncludeMarkdown bool   `json:"includeMarkdown,omitempty"`
}

type SearchParameters struct {
	Q      string `json:"q"`
	GL     string `json:"gl,omitempty"`
	HL     string `json:"hl,omitempty"`
	Num    int    `json:"num,omitempty"`
	Type   string `json:"type"`
	Engine string `json:"engine"`
}

type OrganicResult struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Position int    `json:"position"`
}

type WebSearchResponse struct {
	SearchParameters *SearchParameters `json:"searchParameters"`
	Organic          []*OrganicResult  `json:"organic"`
	Credits          int               `json:"credits"`
}

type ImageResult struct {
	Title        string `json:"title"`
	ImageURL     string `json:"imageUrl"`
	ImageWidth   int    `json:"imageWidth"`
	ImageHeight  int    `json:"imageHeight"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Source       string `json:"source"`
	Domain       string `json:"domain"`
	Link         string `json:"link"`
	Position     int    `json:"position"`
}

type ImageSearchResponse struct {
	SearchParameters *SearchParameters `json:"searchParameters"`
	Images           []*ImageResult    `json:"images"`
	Credits          int               `json:"credits"`
}

type NewsResult struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Date     string `json:"date"`
	Source   string `json:"source"`
	ImageURL string `json:"imageUrl,omitempty"`
	Position int    `json:"position"`
}

type NewsSearchResponse struct {
	SearchParameters *SearchParameters `json:"searchParameters"`
	News             []*NewsResult     `json:"news"`
	Credits          int               `json:"credits"`
}

type ScrapeResponse struct {
	Text     string            `json:"text"`
	Markdown string            `json:"markdown,omitempty"`
	Metadata map[string]string `json:"metadata"`
	Credits  int               `json:"credits"`
}

func (c *Client) WebSearch(ctx context.Context, req *WebSearchRequest) (*WebSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/search/web", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result WebSearchResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding web search response: %w", err)
	}
	return &result, nil
}

func (c *Client) ImageSearch(ctx context.Context, req *ImageSearchRequest) (*ImageSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/search/images", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result ImageSearchResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding image search response: %w", err)
	}
	return &result, nil
}

func (c *Client) NewsSearch(ctx context.Context, req *NewsSearchRequest) (*NewsSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/search/news", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result NewsSearchResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding news search response: %w", err)
	}
	return &result, nil
}

func (c *Client) Scrape(ctx context.Context, req *ScrapeRequest) (*ScrapeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/scrape", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result ScrapeResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding scrape response: %w", err)
	}
	return &result, nil
}
