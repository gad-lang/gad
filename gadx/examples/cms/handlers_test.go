package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	giom "github.com/gad-lang/gad/giom"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestIndexRoute(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected DOCTYPE html")
	}
	if !strings.Contains(body, "GION CMS") {
		t.Error("expected site title in response")
	}
}

func TestPageRoutes(t *testing.T) {
	for _, title := range []string{"About", "Guides", "Contact"} {
		t.Run(title, func(t *testing.T) {
			_, mux := newTestServer(t)
			slug := strings.ToLower(title)
			req := httptest.NewRequest(http.MethodGet, "/pages/"+slug, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertOK(t, rec)
			if !strings.Contains(rec.Body.String(), title) {
				t.Errorf("expected body to contain %q", title)
			}
		})
	}
}

func TestPostRoutes(t *testing.T) {
	posts := []struct{ title, slug string }{
		{"Designing fast editorial pages", "designing-fast-editorial-pages"},
		{"SQLite is enough for a compact CMS", "sqlite-compact-cms"},
		{"Modern admin interfaces without ceremony", "modern-admin-interfaces"},
		{"Reusable GION components", "reusable-gion-components"},
		{"Shipping a friendly first page", "shipping-friendly-first-page"},
		{"Building a gallery component", "building-gallery-component"},
	}
	for _, p := range posts {
		t.Run(p.slug, func(t *testing.T) {
			_, mux := newTestServer(t)
			req := httptest.NewRequest(http.MethodGet, "/posts/"+p.slug, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertOK(t, rec)
			if !strings.Contains(rec.Body.String(), p.title) {
				t.Errorf("expected body to contain %q", p.title)
			}
		})
	}
}

func TestTagRoutes(t *testing.T) {
	_, mux := newTestServer(t)
	for _, name := range []string{"Design", "Engineering", "News", "Tutorials"} {
		t.Run(name, func(t *testing.T) {
			slug := strings.ToLower(name)
			req := httptest.NewRequest(http.MethodGet, "/tags/"+slug, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertOK(t, rec)
			if !strings.Contains(rec.Body.String(), "Posts tagged "+name) {
				t.Errorf("expected body to contain %q", "Posts tagged "+name)
			}
		})
	}
}

func TestTagPagination(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/tags/design?page=1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
}

func TestAdminRoute(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
}

func TestConsecutiveNavigationDoesNotError(t *testing.T) {
	_, mux := newTestServer(t)
	routes := []string{
		"/",
		"/pages/about",
		"/pages/guides",
		"/pages/contact",
		"/posts/designing-fast-editorial-pages",
		"/posts/sqlite-compact-cms",
		"/posts/modern-admin-interfaces",
		"/posts/reusable-gion-components",
		"/posts/shipping-friendly-first-page",
		"/posts/building-gallery-component",
		"/tags/design",
		"/tags/engineering",
		"/tags/news",
		"/tags/tutorials",
		"/tags/design?page=1",
		"/admin",
		"/api/dashboard",
	}
	for i, route := range routes {
		t.Run(fmt.Sprintf("route_%d", i), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: expected 200, got %d: %s", route, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMenuNavigation(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
	body := rec.Body.String()

	links := []struct{ label, href string }{
		{"Home", "/"},
		{"About", "/pages/about"},
		{"Guides", "/pages/guides"},
		{"Design", "/tags/design"},
		{"Engineering", "/tags/engineering"},
		{"Tutorials", "/tags/tutorials"},
	}
	for _, link := range links {
		if !strings.Contains(body, link.href) {
			t.Errorf("expected menu link %q (href=%q) not found in home page", link.label, link.href)
		}
	}
}

func TestNonExistentRoutes(t *testing.T) {
	_, mux := newTestServer(t)
	for _, route := range []string{"/nonexistent", "/pages/nonexistent", "/posts/nonexistent", "/tags/nonexistent"} {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Errorf("expected 404, got %d", rec.Code)
			}
		})
	}
}

func TestStaticFileServing(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/seed-data/images/about-hero.jpg", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertOK(t, rec)
}

func TestResponseContainsNoPanic(t *testing.T) {
	_, mux := newTestServer(t)
	for _, route := range []string{"/", "/pages/about", "/posts/designing-fast-editorial-pages", "/tags/design"} {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return
			}
			body := rec.Body.String()
			for _, sig := range []string{"panic", "runtime error", "nil pointer"} {
				if strings.Contains(body, sig) {
					t.Errorf("response contains error signal %q", sig)
				}
			}
		})
	}
}

func TestEmptyDBDoesNotPanic(t *testing.T) {
	root := findRoot()
	db, _ := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "empty.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	emptyApp := &App{
		DB:           db,
		Root:         root,
		PublicDir:    filepath.Join(root, "public"),
		TranspileDir: filepath.Join(root, "public", ".transpiled"),
	}
	emptyApp.renderer = giom.NewRender(emptyApp.PublicDir)
	emptyApp.renderer.BuiltinsFunc = func() *gad.Builtins {
		return giom.AppendBuiltins(gad.NewBuiltins())
	}
	db.AutoMigrate(&Page{}, &Tag{}, &Post{}, &MenuItem{})
	emptyApp.cleanTranspiled()

	mux := http.NewServeMux()
	emptyApp.routes(mux)

	for _, route := range []string{"/", "/pages/about", "/posts/hello"} {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code == http.StatusInternalServerError {
				t.Fatalf("%s: internal server error: %s", route, rec.Body.String())
			}
		})
	}
}
