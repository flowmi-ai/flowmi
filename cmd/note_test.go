package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func noteListMockServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"items": []map[string]any{
					{
						"id":        "note-1",
						"userId":    "user-1",
						"subject":   "Test Note",
						"content":   "Test content",
						"labels":    []string{"work"},
						"createdAt": "2025-01-01T00:00:00Z",
						"updatedAt": "2025-01-01T00:00:00Z",
					},
					{
						"id":        "note-2",
						"userId":    "user-1",
						"subject":   "Another Note",
						"content":   "More content",
						"labels":    []string{"personal"},
						"createdAt": "2025-01-02T00:00:00Z",
						"updatedAt": "2025-01-02T00:00:00Z",
					},
				},
				"total":    2,
				"page":     1,
				"pageSize": 30,
			},
		})
	}))
}

func setupNoteTest(t *testing.T, server *httptest.Server) {
	t.Helper()
	viper.Set("api_server_url", server.URL)
	viper.Set("access_token", "test-token")
	// Reset flags that accumulate across Execute() calls in the same process.
	// StringSlice's Set("") appends an empty string, so use Replace instead.
	if sv, ok := noteListCmd.Flags().Lookup("label").Value.(pflag.SliceValue); ok {
		sv.Replace(nil)
	}
	t.Cleanup(func() {
		viper.Set("api_server_url", "")
		viper.Set("access_token", "")
	})
}

func TestNoteList(t *testing.T) {
	server := noteListMockServer(t, nil)
	defer server.Close()
	setupNoteTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"note", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("note list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test Note") {
		t.Errorf("output missing 'Test Note', got:\n%s", output)
	}
	if !strings.Contains(output, "Another Note") {
		t.Errorf("output missing 'Another Note', got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 2 of 2 notes") {
		t.Errorf("output missing summary line, got:\n%s", output)
	}
}

func TestNoteListWithSearch(t *testing.T) {
	server := noteListMockServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "foo" {
			t.Errorf("q = %q, want foo", got)
		}
	})
	defer server.Close()
	setupNoteTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"note", "list", "--search", "foo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("note list --search failed: %v", err)
	}
}

func TestNoteListWithLabel(t *testing.T) {
	server := noteListMockServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("labels"); got != "work" {
			t.Errorf("labels = %q, want work", got)
		}
	})
	defer server.Close()
	setupNoteTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"note", "list", "--label", "work"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("note list --label failed: %v", err)
	}
}

func TestNoteListSearchAndLabel(t *testing.T) {
	server := noteListMockServer(t, func(r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("q"); got != "bar" {
			t.Errorf("q = %q, want bar", got)
		}
		if got := q.Get("labels"); got != "personal" {
			t.Errorf("labels = %q, want personal", got)
		}
	})
	defer server.Close()
	setupNoteTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"note", "list", "--search", "bar", "--label", "personal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("note list --search --label failed: %v", err)
	}
}

func TestNoteListJSON(t *testing.T) {
	server := noteListMockServer(t, nil)
	defer server.Close()
	setupNoteTest(t, server)
	viper.Set("output", "json")
	t.Cleanup(func() { viper.Set("output", "") })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"note", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("note list -o json failed: %v", err)
	}

	var list api.NoteListResponse
	if err := json.Unmarshal(buf.Bytes(), &list); err != nil {
		t.Fatalf("JSON output not parseable as NoteListResponse: %v\nOutput:\n%s", err, buf.String())
	}
	if len(list.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(list.Items))
	}
	if list.Items[0].Subject != "Test Note" {
		t.Errorf("Items[0].Subject = %q, want Test Note", list.Items[0].Subject)
	}
}

func TestNoteListCmdHasFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"search", "S"},
		{"label", "l"},
		{"limit", "L"},
	}
	for _, tt := range flags {
		f := noteListCmd.Flags().Lookup(tt.name)
		if f == nil {
			t.Errorf("--%s flag not found", tt.name)
			continue
		}
		if f.Shorthand != tt.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tt.name, f.Shorthand, tt.shorthand)
		}
	}
}
