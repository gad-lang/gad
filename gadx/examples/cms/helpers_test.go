package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
	giom "github.com/gad-lang/gad/giom"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestApp(t testing.TB) *App {
	t.Helper()
	root := findRoot()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cms_test.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	app := &App{
		DB:           db,
		Root:         root,
		PublicDir:    filepath.Join(root, "public"),
		TranspileDir: filepath.Join(root, "public", ".transpiled"),
	}
	app.renderer = giom.NewRender(app.PublicDir)
	app.renderer.BuiltinsFunc = func() *gad.Builtins {
		return giom.AppendBuiltins(gad.NewBuiltins())
	}
	if err := db.AutoMigrate(&Page{}, &Tag{}, &Post{}, &MenuItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	app.cleanTranspiled()
	if err := app.seed(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return app
}

func newTestServer(t testing.TB) (*App, *http.ServeMux) {
	t.Helper()
	app := newTestApp(t)
	mux := http.NewServeMux()
	app.routes(mux)
	return app, mux
}

func findRoot() string {
	candidates := []string{".", "..", "../.."}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "seed.yaml")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	wd, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(wd, "seed.yaml")); err == nil {
		return wd
	}
	return "."
}

func getJSON(t testing.TB, mux *http.ServeMux, route string, v any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, route, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d: %s", route, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("%s: json decode: %v", route, err)
	}
}

func assertOK(t testing.TB, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
