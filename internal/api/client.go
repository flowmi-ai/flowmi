package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL     string
	AccessToken string
	HTTPClient  *http.Client
}

type Response struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     *ErrorBody      `json:"error"`
	RequestID string          `json:"requestId"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UserProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type Note struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Subject   string     `json:"subject"`
	Content   string     `json:"content"`
	Labels    []string   `json:"labels"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type NoteListResponse struct {
	Items    []Note `json:"items"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type CreateNoteRequest struct {
	Subject string   `json:"subject"`
	Content string   `json:"content"`
	Labels  []string `json:"labels,omitempty"`
}

type NotePatch struct {
	Subject *string   `json:"subject,omitempty"`
	Content *string   `json:"content,omitempty"`
	Labels  *[]string `json:"labels,omitempty"`
}

func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var envelope Response
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		snippet := strings.TrimSpace(string(bodyBytes))
		if snippet == "" {
			snippet = http.StatusText(resp.StatusCode)
		}
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("unexpected response (status %d): %s", resp.StatusCode, snippet)
	}

	if !envelope.Success {
		msg := "unknown error"
		if envelope.Error != nil && envelope.Error.Message != "" {
			msg = envelope.Error.Message
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("api error: %s", msg)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d for successful response", resp.StatusCode)
	}

	return &envelope, nil
}

func (c *Client) GetMe(ctx context.Context) (*UserProfile, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/me", nil)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal(resp.Data, &profile); err != nil {
		return nil, fmt.Errorf("decoding user profile: %w", err)
	}
	return &profile, nil
}

func (c *Client) CreateNote(ctx context.Context, subject, content string, labels []string) (*Note, error) {
	body, err := json.Marshal(CreateNoteRequest{
		Subject: subject,
		Content: content,
		Labels:  labels,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/notes", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) ListNotes(ctx context.Context, page, pageSize int, labels []string, status, query string) (*NoteListResponse, error) {
	path := fmt.Sprintf("/api/v1/tools/notes?page=%d&page_size=%d", page, pageSize)
	if len(labels) > 0 {
		path += "&labels=" + url.QueryEscape(strings.Join(labels, ","))
	}
	if status != "" {
		path += "&status=" + url.QueryEscape(status)
	}
	if query != "" {
		path += "&q=" + url.QueryEscape(query)
	}
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list NoteListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding note list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetNote(ctx context.Context, id string) (*Note, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/tools/notes/"+id, nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) PatchNote(ctx context.Context, id string, patch *NotePatch) (*Note, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/tools/notes/"+id, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) DeleteNote(ctx context.Context, id string) (*Note, error) {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/tools/notes/"+id, nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) RestoreNote(ctx context.Context, id string) (*Note, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/notes/"+id+"/restore", nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

// Search & Scrape types

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

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/web-search", bytes.NewReader(body))
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

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/image-search", bytes.NewReader(body))
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

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/news-search", bytes.NewReader(body))
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

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/web-scrape", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result ScrapeResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding scrape response: %w", err)
	}
	return &result, nil
}
