package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkRoutes(b *testing.B) {
	_, mux := newTestServer(b)
	templates := []string{
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
	}
	for _, route := range templates {
		b.Run(route, func(b *testing.B) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)
			}
		})
	}
}

func BenchmarkSequentialNavigation(b *testing.B) {
	_, mux := newTestServer(b)
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
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, route := range routes {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
		}
	}
}

func BenchmarkColdRender(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := newTestApp(b)
		mux := http.NewServeMux()
		app.routes(mux)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkMixedWorkload(b *testing.B) {
	_, mux := newTestServer(b)
	endpoints := []string{
		"/",
		"/pages/about",
		"/posts/designing-fast-editorial-pages",
		"/tags/design",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, route := range endpoints {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Fatalf("%s: expected 200, got %d", route, rec.Code)
			}
		}
	}
}

func BenchmarkPostWithRelatedContent(b *testing.B) {
	_, mux := newTestServer(b)
	slugs := []string{
		"/posts/designing-fast-editorial-pages",
		"/posts/sqlite-compact-cms",
		"/posts/modern-admin-interfaces",
		"/posts/reusable-gion-components",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, slug := range slugs {
			req := httptest.NewRequest(http.MethodGet, slug, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Fatalf("%s: expected 200, got %d", slug, rec.Code)
			}
		}
	}
}

func BenchmarkTagWithPagination(b *testing.B) {
	_, mux := newTestServer(b)
	pages := []string{
		"/tags/design?page=1",
		"/tags/engineering?page=1",
		"/tags/news?page=1",
		"/tags/tutorials?page=1",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, route := range pages {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Fatalf("%s: expected 200, got %d", route, rec.Code)
			}
		}
	}
}

func BenchmarkColdVsWarmChart(b *testing.B) {
	routes := []struct {
		Label string
		Route string
	}{
		{"index", "/"},
		{"page", "/pages/about"},
		{"post", "/posts/designing-fast-editorial-pages"},
		{"tag", "/tags/design"},
	}

	samples := 5
	var results []benchResult
	for _, rt := range routes {
		var coldSum, warmSum float64
		for s := 0; s < samples; s++ {
			coldSum += measureColdNs(b, rt.Route)
			warmSum += measureWarmNs(b, rt.Route)
		}
		avgCold := coldSum / float64(samples)
		avgWarm := warmSum / float64(samples)
		r := benchResult{
			Label:   rt.Label,
			ColdUS:  avgCold / 1000.0,
			WarmUS:  avgWarm / 1000.0,
			Speedup: avgCold / avgWarm,
		}
		results = append(results, r)
		b.Logf("%-6s  cold=%8.0fµs  warm=%8.0fµs  speedup=%5.1f×",
			r.Label, r.ColdUS, r.WarmUS, r.Speedup)
	}
	root := filepath.Dir(filepath.Dir(findRoot()))
	chartDir := filepath.Join(root, "docs")
	if err := generateChart(results, chartDir); err != nil {
		b.Logf("generate chart: %v", err)
	} else {
		b.Logf("chart saved to %s/bench-cold-vs-warm.svg", chartDir)
	}
}

func measureColdNs(b *testing.B, route string) float64 {
	app := newTestApp(b)
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodGet, route, nil)

	start := time.Now()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		b.Fatalf("cold %s: expected 200, got %d", route, rec.Code)
	}
	return float64(elapsed.Nanoseconds())
}

func measureWarmNs(b *testing.B, route string) float64 {
	app := newTestApp(b)
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodGet, route, nil)

	// First request warms the cache
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		b.Fatalf("warm-prep %s: expected 200, got %d", route, rec1.Code)
	}

	// Measure the cached request
	start := time.Now()
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)
	elapsed := time.Since(start)

	if rec2.Code != http.StatusOK {
		b.Fatalf("warm %s: expected 200, got %d", route, rec2.Code)
	}
	return float64(elapsed.Nanoseconds())
}
