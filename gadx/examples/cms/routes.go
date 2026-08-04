package main

import (
	"net/http"
	"path/filepath"
)

func (a *App) routes(mux *http.ServeMux) {
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(a.Root, "static")))))
	mux.Handle("/seed-data/", http.StripPrefix("/seed-data/", http.FileServer(http.Dir(filepath.Join(a.Root, "seed-data")))))
	mux.HandleFunc("/admin", a.admin)
	mux.HandleFunc("/admin/", a.admin)
	mux.HandleFunc("/api/dashboard", a.apiDashboard)
	mux.HandleFunc("/api/pages", a.collection(&Page{}))
	mux.HandleFunc("/api/pages/", a.member(&Page{}))
	mux.HandleFunc("/api/posts", a.postsCollection)
	mux.HandleFunc("/api/posts/", a.postMember)
	mux.HandleFunc("/api/tags", a.collection(&Tag{}))
	mux.HandleFunc("/api/tags/", a.member(&Tag{}))
	mux.HandleFunc("/api/menus", a.collection(&MenuItem{}))
	mux.HandleFunc("/api/menus/", a.member(&MenuItem{}))
	mux.HandleFunc("/pages/", a.page)
	mux.HandleFunc("/posts/", a.post)
	mux.HandleFunc("/tags/", a.tagPosts)
	mux.HandleFunc("/", a.index)
}

func (a *App) collection(model any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			a.list(w, model)
		case http.MethodPost:
			a.create(w, r, model)
		default:
			methodNotAllowed(w)
		}
	}
}

func (a *App) member(model any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idFromPath(r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodPut:
			a.update(w, r, model, id)
		case http.MethodDelete:
			a.delete(w, model, id)
		default:
			methodNotAllowed(w)
		}
	}
}
