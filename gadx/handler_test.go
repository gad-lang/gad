package gadx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
)

func TestHandlerFunc(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n    p Hello {= Name }"
	file := filepath.Join(dir, "hi.gadx")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	r := newTestRender(t, dir)
	h := r.HandlerFunc(file, func(req *http.Request) (gad.Dict, error) {
		return gad.Dict{"Name": gad.Str(req.URL.Query().Get("n"))}, nil
	})

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/?n=Ada", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if body := rec.Body.String(); body != "<p>Hello Ada</p>" {
		t.Fatalf("body = %q, want <p>Hello Ada</p>", body)
	}
}

func TestHandlerFuncNilModel(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "p.gadx")
	if err := os.WriteFile(file, []byte("@main\n    p Hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	newTestRender(t, dir).HandlerFunc(file, nil)(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "<p>Hi</p>" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHandlerFuncModelError(t *testing.T) {
	r := newTestRender(t, t.TempDir())
	h := r.HandlerFunc("whatever.gadx", func(*http.Request) (gad.Dict, error) {
		return nil, errors.New("boom")
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandlerFuncRenderError(t *testing.T) {
	// A missing template file makes Render fail, so the handler must 500 without
	// having written a partial body.
	r := newTestRender(t, t.TempDir())
	h := r.HandlerFunc(filepath.Join(t.TempDir(), "missing.gadx"), nil)
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
