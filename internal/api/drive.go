package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DriveObject struct {
	ID         string         `json:"id"`
	Path       string         `json:"path"`
	SizeBytes  int64          `json:"sizeBytes"`
	MimeType   string         `json:"mimeType,omitempty"`
	ETag       string         `json:"etag,omitempty"`
	Visibility string         `json:"visibility"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  *time.Time     `json:"deletedAt,omitempty"`
}

type DriveListResponse struct {
	Items    []DriveObject `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type InitUploadRequest struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	MimeType  string `json:"mimeType,omitempty"`
}

type InitUploadResponse struct {
	ID        string `json:"id"`
	UploadURL string `json:"uploadUrl"`
	ExpiresIn int    `json:"expiresIn"`
}

type CompleteUploadRequest struct {
	ETag      string `json:"etag"`
	SizeBytes int64  `json:"sizeBytes"`
}

type DownloadResponse struct {
	DownloadURL string `json:"downloadUrl"`
	ExpiresIn   int    `json:"expiresIn"`
}

func (c *Client) ListDriveObjects(ctx context.Context, page, pageSize int, prefix string) (*DriveListResponse, error) {
	path := fmt.Sprintf("/api/v1/drive/objects?page=%d&pageSize=%d", page, pageSize)
	if prefix != "" {
		path += "&prefix=" + url.QueryEscape(prefix)
	}
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list DriveListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding drive object list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetDriveObject(ctx context.Context, id string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/objects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) GetDriveObjectByPath(ctx context.Context, path string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/object?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) DeleteDriveObject(ctx context.Context, id string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/drive/objects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) DeleteDriveObjectByPath(ctx context.Context, path string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/drive/object?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) ListTrashedDriveObjects(ctx context.Context, page, pageSize int) (*DriveListResponse, error) {
	path := fmt.Sprintf("/api/v1/drive/trash?page=%d&pageSize=%d", page, pageSize)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list DriveListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding trashed drive object list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetTrashedDriveObject(ctx context.Context, id string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/trash/"+id, nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) GetTrashedDownloadURL(ctx context.Context, id string) (*DownloadResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/trash/"+id+"/download", nil)
	if err != nil {
		return nil, err
	}

	var result DownloadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding download response: %w", err)
	}
	return &result, nil
}

func (c *Client) RestoreDriveObject(ctx context.Context, id string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/drive/trash/"+id+"/restore", nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) PermanentlyDeleteDriveObject(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/drive/trash/"+id, nil)
	return err
}

func (c *Client) InitUpload(ctx context.Context, req *InitUploadRequest) (*InitUploadResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/drive/upload", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result InitUploadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding init upload response: %w", err)
	}
	return &result, nil
}

func (c *Client) CompleteUpload(ctx context.Context, id, etag string, sizeBytes int64) (*DriveObject, error) {
	body, err := json.Marshal(&CompleteUploadRequest{
		ETag:      etag,
		SizeBytes: sizeBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/drive/upload/"+id+"/complete", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) GetDownloadURL(ctx context.Context, id string) (*DownloadResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/objects/"+id+"/download", nil)
	if err != nil {
		return nil, err
	}

	var result DownloadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding download response: %w", err)
	}
	return &result, nil
}

func (c *Client) GetDownloadURLByPath(ctx context.Context, path string) (*DownloadResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/object/download?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}

	var result DownloadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding download response: %w", err)
	}
	return &result, nil
}

// UploadToPresignedURL uploads binary data to a presigned R2 URL.
// Returns the ETag from the response header.
func (c *Client) UploadToPresignedURL(ctx context.Context, uploadURL, contentType string, body io.Reader, size int64) (string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Minute}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, body)
	if err != nil {
		return "", fmt.Errorf("creating upload request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.ContentLength = size

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading to presigned URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return "", fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	etag := resp.Header.Get("ETag")
	// Strip surrounding quotes from ETag if present
	if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
		etag, _ = strconv.Unquote(etag)
	}
	return etag, nil
}

// DownloadFromURL downloads binary data from a presigned R2 URL.
// Caller must close the returned ReadCloser.
func (c *Client) DownloadFromURL(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	httpClient := &http.Client{Timeout: 10 * time.Minute}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading from presigned URL: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed (status %d)", resp.StatusCode)
	}

	return resp.Body, nil
}
