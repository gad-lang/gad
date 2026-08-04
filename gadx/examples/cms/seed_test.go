package main

import "testing"

func TestSeedDataIntegrity(t *testing.T) {
	_, mux := newTestServer(t)
	var result map[string]float64
	getJSON(t, mux, "/api/dashboard", &result)
	if result["pages"] != 3 {
		t.Errorf("expected 3 pages, got %v", result["pages"])
	}
	if result["tags"] != 4 {
		t.Errorf("expected 4 tags, got %v", result["tags"])
	}
	if result["posts"] != 6 {
		t.Errorf("expected 6 posts, got %v", result["posts"])
	}
	if result["menus"] != 7 {
		t.Errorf("expected 7 menus, got %v", result["menus"])
	}
}
