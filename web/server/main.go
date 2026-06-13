// Command server is the backend example for the Gad CodeMirror integration. It
// exposes /api/fmt, /api/run and /api/diagnose, which the React app calls to
// format, execute and lint Gad source — mirroring `gad` reading from stdin and
// writing to stdout, but returning structured per-line/column diagnostics.
//
// It also serves the built React app (and the WebAssembly assets) when a static
// directory is provided, so a single binary can host the whole demo.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gad-lang/gad/web/gadbridge"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	static := flag.String("static", "app/dist", "directory with the built React app (optional)")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/fmt", jsonHandler(func(src string) any { return gadbridge.Format(src) }))
	mux.HandleFunc("/api/run", jsonHandler(func(src string) any { return gadbridge.Run(src) }))
	mux.HandleFunc("/api/diagnose", jsonHandler(func(src string) any {
		return map[string]any{"diagnostics": gadbridge.Diagnose(src)}
	}))

	if info, err := os.Stat(*static); err == nil && info.IsDir() {
		log.Printf("serving static files from %s", *static)
		mux.Handle("/", spaFileServer(*static))
	} else {
		log.Printf("no static dir at %s; serving API only", *static)
	}

	log.Printf("gad web server listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, withCORS(mux)))
}

// sourceRequest is the JSON body shared by all API endpoints.
type sourceRequest struct {
	Source string `json:"source"`
}

// jsonHandler decodes {"source": "..."} and writes the handler's value as JSON.
func jsonHandler(fn func(src string) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req sourceRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fn(req.Source))
	}
}

// withCORS allows the Vite dev server (a different origin) to call the API.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// spaFileServer serves files from dir, falling back to index.html for client
// routes (single-page app behavior).
func spaFileServer(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			if r.URL.Path != "/" {
				if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
					http.ServeFile(w, r, filepath.Join(dir, "index.html"))
					return
				}
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
