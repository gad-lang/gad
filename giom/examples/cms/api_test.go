package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIDashboard(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
	var result map[string]float64
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	for _, key := range []string{"pages", "posts", "tags", "menus"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected %q in dashboard response", key)
		}
	}
}

func TestAPICollections(t *testing.T) {
	_, mux := newTestServer(t)

	t.Run("GET /api/pages", func(t *testing.T) {
		var pages []Page
		getJSON(t, mux, "/api/pages", &pages)
		if len(pages) != 3 {
			t.Errorf("expected 3 pages, got %d", len(pages))
		}
	})

	t.Run("GET /api/tags", func(t *testing.T) {
		var tags []Tag
		getJSON(t, mux, "/api/tags", &tags)
		if len(tags) != 4 {
			t.Errorf("expected 4 tags, got %d", len(tags))
		}
	})

	t.Run("GET /api/menus", func(t *testing.T) {
		var menus []MenuItem
		getJSON(t, mux, "/api/menus", &menus)
		if len(menus) != 7 {
			t.Errorf("expected 7 menu items, got %d", len(menus))
		}
	})

	t.Run("GET /api/posts", func(t *testing.T) {
		var posts []Post
		getJSON(t, mux, "/api/posts", &posts)
		if len(posts) != 6 {
			t.Errorf("expected 6 posts, got %d", len(posts))
		}
	})
}

func TestCRUDCreatePage(t *testing.T) {
	_, mux := newTestServer(t)
	payload := `{"title":"Test Page","slug":"test-page","summary":"A test page","body":"<p>hello</p>","published":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/pages", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
	var page Page
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if page.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", page.Title)
	}
}

func TestCRUDUpdateAndDeletePage(t *testing.T) {
	_, mux := newTestServer(t)

	var pages []Page
	getJSON(t, mux, "/api/pages", &pages)
	if len(pages) == 0 {
		t.Fatal("expected some pages")
	}
	id := pages[0].ID

	payload := fmt.Sprintf(`{"title":"Updated %s","slug":"%s"}`, pages[0].Title, pages[0].Slug)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/pages/%d", id), strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)

	var updated []Page
	getJSON(t, mux, "/api/pages", &updated)

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/pages/%d", id), nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	var afterDelete []Page
	getJSON(t, mux, "/api/pages", &afterDelete)
	if len(afterDelete) != len(updated)-1 {
		t.Errorf("expected %d pages after delete, got %d", len(updated)-1, len(afterDelete))
	}
}

func TestPostsByTagAssociation(t *testing.T) {
	_, mux := newTestServer(t)
	var posts []Post
	getJSON(t, mux, "/api/posts", &posts)

	tagPostCounts := map[string]int{
		"design":      3,
		"engineering": 3,
		"news":        2,
		"tutorials":   2,
	}
	for _, post := range posts {
		for _, tag := range post.Tags {
			tagPostCounts[tag.Slug]--
		}
	}
	for slug, remaining := range tagPostCounts {
		if remaining != 0 {
			t.Errorf("tag %q has %d remaining (expected 0)", slug, remaining)
		}
	}
}

func TestHTTPMethods(t *testing.T) {
	_, mux := newTestServer(t)
	tests := []struct {
		method string
		route  string
		code   int
	}{
		{http.MethodHead, "/", http.StatusOK},
		{http.MethodPost, "/api/dashboard", http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/pages", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/pages", http.StatusMethodNotAllowed},
	}
	for _, tc := range tests {
		t.Run(tc.method+"_"+tc.route, func(t *testing.T) {
			var reqBody io.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				reqBody = strings.NewReader(`{}`)
			}
			req := httptest.NewRequest(tc.method, tc.route, reqBody)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.code {
				t.Errorf("expected %d, got %d", tc.code, rec.Code)
			}
		})
	}
}
