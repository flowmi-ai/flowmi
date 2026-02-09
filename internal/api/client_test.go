package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			Success: false,
			Error: &ErrorBody{
				Code:    "UNAUTHORIZED",
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
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestWebSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/tools/web-search" {
			t.Errorf("path = %s, want /api/v1/tools/web-search", r.URL.Path)
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
		if r.URL.Path != "/api/v1/tools/image-search" {
			t.Errorf("path = %s, want /api/v1/tools/image-search", r.URL.Path)
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
		if r.URL.Path != "/api/v1/tools/news-search" {
			t.Errorf("path = %s, want /api/v1/tools/news-search", r.URL.Path)
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
		if r.URL.Path != "/api/v1/tools/web-scrape" {
			t.Errorf("path = %s, want /api/v1/tools/web-scrape", r.URL.Path)
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
