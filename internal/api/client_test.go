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
