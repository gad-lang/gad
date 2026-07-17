package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
)

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	posts, err := a.latestPosts(5)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, "index.giom", a.model("Home", []crumb{{"Home", "/"}}, gad.Dict{
		"Posts": postsValue(posts),
	}))
}

func (a *App) page(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/pages/")
	var p Page
	if err := a.DB.Where("slug = ? AND published = ?", slug, true).First(&p).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, "page.giom", a.model(p.Title, []crumb{{"Home", "/"}, {p.Title, "/pages/" + p.Slug}}, gad.Dict{
		"Page": pageValue(p),
	}))
}

func (a *App) post(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/posts/")
	var p Post
	if err := a.DB.Preload("Tags").Where("slug = ? AND published = ?", slug, true).First(&p).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, "post.giom", a.model(p.Title, []crumb{{"Home", "/"}, {"Posts", "/"}, {p.Title, "/posts/" + p.Slug}}, gad.Dict{
		"Post": postValue(p),
	}))
}

func (a *App) tagPosts(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/tags/")
	pageNo := queryInt(r, "page", 1)
	if pageNo < 1 {
		pageNo = 1
	}
	var tag Tag
	if err := a.DB.Where("slug = ?", slug).First(&tag).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	posts, total, err := a.postsByTag(tag.ID, pageNo, 5)
	if err != nil {
		a.serverError(w, err)
		return
	}
	totalPages := int((total + 4) / 5)
	a.render(w, "tag.giom", a.model(tag.Name, []crumb{{"Home", "/"}, {tag.Name, "/tags/" + tag.Slug}}, gad.Dict{
		"Tag":   tagValue(tag),
		"Posts": postsValue(posts),
		"Pager": gad.Dict{
			"Page":       gad.Int(pageNo),
			"TotalPages": gad.Int(totalPages),
			"HasPrev":    gad.Bool(pageNo > 1),
			"HasNext":    gad.Bool(pageNo < totalPages),
			"PrevURL":    gad.Str(fmt.Sprintf("/tags/%s?page=%d", tag.Slug, pageNo-1)),
			"NextURL":    gad.Str(fmt.Sprintf("/tags/%s?page=%d", tag.Slug, pageNo+1)),
		},
	}))
}

func (a *App) admin(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/admin/src/") {
		name := strings.TrimPrefix(r.URL.Path, "/admin/src/")
		http.ServeFile(w, r, filepath.Join(a.Root, "admin", "src", filepath.Clean(name)))
		return
	}
	http.ServeFile(w, r, filepath.Join(a.Root, "admin", "index.html"))
}
