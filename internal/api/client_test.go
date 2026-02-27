package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newMockNoteListServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &NoteListResponse{
				Items: []Note{
					{
						ID:        "note-1",
						UserID:    "user-1",
						Subject:   "First note",
						Content:   "Hello world",
						Labels:    []string{"work"},
						CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        "note-2",
						UserID:    "user-1",
						Subject:   "Second note",
						Content:   "Goodbye world",
						Labels:    []string{"personal"},
						CreatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
				Total:    2,
				Page:     1,
				PageSize: 30,
			}),
		})
	}))
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshaling test data: %v", err)
	}
	return b
}

func TestListNotes(t *testing.T) {
	server := newMockNoteListServer(t, func(r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		q := r.URL.Query()
		if got := q.Get("page"); got != "1" {
			t.Errorf("page = %q, want 1", got)
		}
		if got := q.Get("page_size"); got != "30" {
			t.Errorf("page_size = %q, want 30", got)
		}
		if q.Has("labels") {
			t.Errorf("labels should not be present, got %q", q.Get("labels"))
		}
		if q.Has("status") {
			t.Errorf("status should not be present, got %q", q.Get("status"))
		}
		if q.Has("q") {
			t.Errorf("q should not be present, got %q", q.Get("q"))
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	list, err := client.ListNotes(context.Background(), 1, 30, nil, "", "")
	if err != nil {
		t.Fatalf("ListNotes() error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(list.Items))
	}
	if list.Total != 2 {
		t.Errorf("Total = %d, want 2", list.Total)
	}
	if list.Page != 1 {
		t.Errorf("Page = %d, want 1", list.Page)
	}
	if list.PageSize != 30 {
		t.Errorf("PageSize = %d, want 30", list.PageSize)
	}
	if list.Items[0].Subject != "First note" {
		t.Errorf("Items[0].Subject = %q, want First note", list.Items[0].Subject)
	}
}

func TestListNotesWithLabels(t *testing.T) {
	server := newMockNoteListServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("labels"); got != "work,personal" {
			t.Errorf("labels = %q, want work,personal", got)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.ListNotes(context.Background(), 1, 30, []string{"work", "personal"}, "", "")
	if err != nil {
		t.Fatalf("ListNotes() error: %v", err)
	}
}

func TestListNotesWithStatus(t *testing.T) {
	server := newMockNoteListServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "trashed" {
			t.Errorf("status = %q, want trashed", got)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.ListNotes(context.Background(), 1, 30, nil, "trashed", "")
	if err != nil {
		t.Fatalf("ListNotes() error: %v", err)
	}
}

func TestListNotesWithQuery(t *testing.T) {
	server := newMockNoteListServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "search term" {
			t.Errorf("q = %q, want search term", got)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.ListNotes(context.Background(), 1, 30, nil, "", "search term")
	if err != nil {
		t.Fatalf("ListNotes() error: %v", err)
	}
}

func TestListNotesAllParams(t *testing.T) {
	server := newMockNoteListServer(t, func(r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("labels"); got != "work,personal" {
			t.Errorf("labels = %q, want work,personal", got)
		}
		if got := q.Get("status"); got != "trashed" {
			t.Errorf("status = %q, want trashed", got)
		}
		if got := q.Get("q"); got != "hello world" {
			t.Errorf("q = %q, want hello world", got)
		}
		if got := q.Get("page"); got != "2" {
			t.Errorf("page = %q, want 2", got)
		}
		if got := q.Get("page_size"); got != "10" {
			t.Errorf("page_size = %q, want 10", got)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.ListNotes(context.Background(), 2, 10, []string{"work", "personal"}, "trashed", "hello world")
	if err != nil {
		t.Fatalf("ListNotes() error: %v", err)
	}
}

func TestListNotesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success:   false,
			RequestID: "req_abc123",
			Error: &ErrorBody{
				Code:    CodeAuthUnauthorized,
				Message: "invalid or expired token",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-token")
	_, err := client.ListNotes(context.Background(), 1, 30, nil, "", "")
	if err == nil {
		t.Fatal("expected error for API error response")
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if apiErr.Code != CodeAuthUnauthorized {
		t.Errorf("Code = %q, want %q", apiErr.Code, CodeAuthUnauthorized)
	}
	if apiErr.Message != "invalid or expired token" {
		t.Errorf("Message = %q, want 'invalid or expired token'", apiErr.Message)
	}
	if apiErr.RequestID != "req_abc123" {
		t.Errorf("RequestID = %q, want req_abc123", apiErr.RequestID)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
	}
	if apiErr.ExitCode() != ExitAuth {
		t.Errorf("ExitCode() = %d, want %d", apiErr.ExitCode(), ExitAuth)
	}
}

func TestWebSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/search/web" {
			t.Errorf("path = %s, want /api/v1/search/web", r.URL.Path)
		}

		var req WebSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q != "golang concurrency" {
			t.Errorf("q = %q, want golang concurrency", req.Q)
		}
		if req.Num != 5 {
			t.Errorf("num = %d, want 5", req.Num)
		}
		if req.GL != "us" {
			t.Errorf("gl = %q, want us", req.GL)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &WebSearchResponse{
				SearchParameters: &SearchParameters{Q: "golang concurrency", Type: "search", Engine: "google"},
				Organic: []*OrganicResult{
					{Title: "Go Concurrency", Link: "https://example.com", Snippet: "Learn Go", Position: 1},
				},
				Credits: 1,
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.WebSearch(context.Background(), &WebSearchRequest{
		Q:   "golang concurrency",
		GL:  "us",
		Num: 5,
	})
	if err != nil {
		t.Fatalf("WebSearch() error: %v", err)
	}
	if len(result.Organic) != 1 {
		t.Errorf("len(Organic) = %d, want 1", len(result.Organic))
	}
	if result.Organic[0].Title != "Go Concurrency" {
		t.Errorf("Organic[0].Title = %q, want Go Concurrency", result.Organic[0].Title)
	}
}

func TestImageSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/images" {
			t.Errorf("path = %s, want /api/v1/search/images", r.URL.Path)
		}

		var req ImageSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q != "golang gopher" {
			t.Errorf("q = %q, want golang gopher", req.Q)
		}
		if req.TBS != "isz:l" {
			t.Errorf("tbs = %q, want isz:l", req.TBS)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &ImageSearchResponse{
				SearchParameters: &SearchParameters{Q: "golang gopher", Type: "images", Engine: "google"},
				Images: []*ImageResult{
					{Title: "Gopher", ImageURL: "https://example.com/gopher.png", ImageWidth: 1200, ImageHeight: 800, Domain: "example.com", Position: 1},
				},
				Credits: 1,
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ImageSearch(context.Background(), &ImageSearchRequest{
		Q:   "golang gopher",
		TBS: "isz:l",
	})
	if err != nil {
		t.Fatalf("ImageSearch() error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Errorf("len(Images) = %d, want 1", len(result.Images))
	}
	if result.Images[0].ImageWidth != 1200 {
		t.Errorf("Images[0].ImageWidth = %d, want 1200", result.Images[0].ImageWidth)
	}
}

func TestNewsSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/news" {
			t.Errorf("path = %s, want /api/v1/search/news", r.URL.Path)
		}

		var req NewsSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q != "golang release" {
			t.Errorf("q = %q, want golang release", req.Q)
		}
		if req.TBS != "qdr:w" {
			t.Errorf("tbs = %q, want qdr:w", req.TBS)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &NewsSearchResponse{
				SearchParameters: &SearchParameters{Q: "golang release", Type: "news", Engine: "google"},
				News: []*NewsResult{
					{Title: "Go 1.23 Released", Link: "https://go.dev/blog", Snippet: "New release", Date: "2 hours ago", Source: "Go Blog", Position: 1},
				},
				Credits: 1,
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.NewsSearch(context.Background(), &NewsSearchRequest{
		Q:   "golang release",
		TBS: "qdr:w",
	})
	if err != nil {
		t.Fatalf("NewsSearch() error: %v", err)
	}
	if len(result.News) != 1 {
		t.Errorf("len(News) = %d, want 1", len(result.News))
	}
	if result.News[0].Source != "Go Blog" {
		t.Errorf("News[0].Source = %q, want Go Blog", result.News[0].Source)
	}
}

func TestScrape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/scrape" {
			t.Errorf("path = %s, want /api/v1/scrape", r.URL.Path)
		}

		var req ScrapeRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.URL != "https://example.com" {
			t.Errorf("url = %q, want https://example.com", req.URL)
		}
		if !req.IncludeMarkdown {
			t.Error("includeMarkdown = false, want true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &ScrapeResponse{
				Text:     "Example Domain",
				Markdown: "# Example Domain",
				Metadata: map[string]string{"title": "Example Domain"},
				Credits:  1,
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.Scrape(context.Background(), &ScrapeRequest{
		URL:             "https://example.com",
		IncludeMarkdown: true,
	})
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}
	if result.Text != "Example Domain" {
		t.Errorf("Text = %q, want Example Domain", result.Text)
	}
	if result.Markdown != "# Example Domain" {
		t.Errorf("Markdown = %q, want # Example Domain", result.Markdown)
	}
	if result.Metadata["title"] != "Example Domain" {
		t.Errorf("Metadata[title] = %q, want Example Domain", result.Metadata["title"])
	}
}

// Drive API tests

func newMockDriveListServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &DriveListResponse{
				Items: []DriveObject{
					{
						ID:         "obj-1",
						Path:       "/docs/readme.txt",
						SizeBytes:  1024,
						MimeType:   "text/plain",
						Visibility: "private",
						Properties: map[string]any{},
						CreatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				Total:    1,
				Page:     1,
				PageSize: 30,
			}),
		})
	}))
}

func TestListDriveObjects(t *testing.T) {
	server := newMockDriveListServer(t, func(r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		q := r.URL.Query()
		if got := q.Get("page"); got != "1" {
			t.Errorf("page = %q, want 1", got)
		}
		if got := q.Get("pageSize"); got != "30" {
			t.Errorf("pageSize = %q, want 30", got)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	list, err := client.ListDriveObjects(context.Background(), 1, 30, "")
	if err != nil {
		t.Fatalf("ListDriveObjects() error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(list.Items))
	}
	if list.Items[0].Path != "/docs/readme.txt" {
		t.Errorf("Items[0].Path = %q, want /docs/readme.txt", list.Items[0].Path)
	}
}

func TestListDriveObjectsWithPrefix(t *testing.T) {
	server := newMockDriveListServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("prefix"); got != "/docs" {
			t.Errorf("prefix = %q, want /docs", got)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.ListDriveObjects(context.Background(), 1, 30, "/docs")
	if err != nil {
		t.Fatalf("ListDriveObjects() error: %v", err)
	}
}

func TestGetDriveObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/drive/objects/obj-1" {
			t.Errorf("path = %s, want /api/v1/drive/objects/obj-1", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &DriveObject{
				ID:         "obj-1",
				Path:       "/docs/readme.txt",
				SizeBytes:  1024,
				MimeType:   "text/plain",
				Visibility: "private",
				Properties: map[string]any{},
				CreatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	obj, err := client.GetDriveObject(context.Background(), "obj-1")
	if err != nil {
		t.Fatalf("GetDriveObject() error: %v", err)
	}
	if obj.ID != "obj-1" {
		t.Errorf("ID = %q, want obj-1", obj.ID)
	}
	if obj.Path != "/docs/readme.txt" {
		t.Errorf("Path = %q, want /docs/readme.txt", obj.Path)
	}
}

func TestGetDriveObjectByPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/drive/object" {
			t.Errorf("path = %s, want /api/v1/drive/object", r.URL.Path)
		}
		if got := r.URL.Query().Get("path"); got != "/docs/readme.txt" {
			t.Errorf("path param = %q, want /docs/readme.txt", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &DriveObject{
				ID:         "obj-1",
				Path:       "/docs/readme.txt",
				SizeBytes:  1024,
				Visibility: "private",
				Properties: map[string]any{},
				CreatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	obj, err := client.GetDriveObjectByPath(context.Background(), "/docs/readme.txt")
	if err != nil {
		t.Fatalf("GetDriveObjectByPath() error: %v", err)
	}
	if obj.ID != "obj-1" {
		t.Errorf("ID = %q, want obj-1", obj.ID)
	}
}

func TestDeleteDriveObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/drive/objects/obj-1" {
			t.Errorf("path = %s, want /api/v1/drive/objects/obj-1", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &DriveObject{
				ID:         "obj-1",
				Path:       "/docs/readme.txt",
				SizeBytes:  1024,
				Visibility: "private",
				Properties: map[string]any{},
				CreatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	obj, err := client.DeleteDriveObject(context.Background(), "obj-1")
	if err != nil {
		t.Fatalf("DeleteDriveObject() error: %v", err)
	}
	if obj.ID != "obj-1" {
		t.Errorf("ID = %q, want obj-1", obj.ID)
	}
}

func TestInitUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/drive/upload" {
			t.Errorf("path = %s, want /api/v1/drive/upload", r.URL.Path)
		}

		var req InitUploadRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Path != "/test.txt" {
			t.Errorf("path = %q, want /test.txt", req.Path)
		}
		if req.SizeBytes != 1024 {
			t.Errorf("sizeBytes = %d, want 1024", req.SizeBytes)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &InitUploadResponse{
				ID:        "upload-1",
				UploadURL: "https://r2.example.com/presigned",
				ExpiresIn: 3600,
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.InitUpload(context.Background(), &InitUploadRequest{
		Path:      "/test.txt",
		SizeBytes: 1024,
		MimeType:  "text/plain",
	})
	if err != nil {
		t.Fatalf("InitUpload() error: %v", err)
	}
	if resp.ID != "upload-1" {
		t.Errorf("ID = %q, want upload-1", resp.ID)
	}
	if resp.UploadURL != "https://r2.example.com/presigned" {
		t.Errorf("UploadURL = %q, want https://r2.example.com/presigned", resp.UploadURL)
	}
}

func TestCompleteUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/drive/upload/upload-1/complete" {
			t.Errorf("path = %s, want /api/v1/drive/upload/upload-1/complete", r.URL.Path)
		}

		var req CompleteUploadRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ETag != "etag-abc" {
			t.Errorf("etag = %q, want etag-abc", req.ETag)
		}
		if req.SizeBytes != 1024 {
			t.Errorf("sizeBytes = %d, want 1024", req.SizeBytes)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &DriveObject{
				ID:         "obj-new",
				Path:       "/test.txt",
				SizeBytes:  1024,
				Visibility: "private",
				Properties: map[string]any{},
				CreatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	obj, err := client.CompleteUpload(context.Background(), "upload-1", "etag-abc", 1024)
	if err != nil {
		t.Fatalf("CompleteUpload() error: %v", err)
	}
	if obj.ID != "obj-new" {
		t.Errorf("ID = %q, want obj-new", obj.ID)
	}
}

func TestGetDownloadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/drive/objects/obj-1/download" {
			t.Errorf("path = %s, want /api/v1/drive/objects/obj-1/download", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: mustMarshal(t, &DownloadResponse{
				DownloadURL: "https://r2.example.com/download",
				ExpiresIn:   3600,
			}),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetDownloadURL(context.Background(), "obj-1")
	if err != nil {
		t.Fatalf("GetDownloadURL() error: %v", err)
	}
	if resp.DownloadURL != "https://r2.example.com/download" {
		t.Errorf("DownloadURL = %q, want https://r2.example.com/download", resp.DownloadURL)
	}
}

func TestUploadToPresignedURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "text/plain" {
			t.Errorf("Content-Type = %q, want text/plain", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" {
			t.Errorf("body = %q, want hello", string(body))
		}
		w.Header().Set("ETag", `"etag-123"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("unused", "unused")
	etag, err := client.UploadToPresignedURL(context.Background(), server.URL, "text/plain", strings.NewReader("hello"), 5)
	if err != nil {
		t.Fatalf("UploadToPresignedURL() error: %v", err)
	}
	if etag != "etag-123" {
		t.Errorf("etag = %q, want etag-123", etag)
	}
}

func TestDownloadFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("binary data"))
	}))
	defer server.Close()

	client := NewClient("unused", "unused")
	body, err := client.DownloadFromURL(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("DownloadFromURL() error: %v", err)
	}
	defer body.Close()

	data, _ := io.ReadAll(body)
	if string(data) != "binary data" {
		t.Errorf("body = %q, want binary data", string(data))
	}
}
