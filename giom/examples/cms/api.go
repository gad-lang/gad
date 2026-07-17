package main

import (
	"encoding/json"
	"net/http"

	"github.com/gad-lang/gad"
)

func (a *App) apiDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var pages, posts, tags, menus int64
	a.DB.Model(&Page{}).Count(&pages)
	a.DB.Model(&Post{}).Count(&posts)
	a.DB.Model(&Tag{}).Count(&tags)
	a.DB.Model(&MenuItem{}).Count(&menus)
	writeJSON(w, gad.Dict{"pages": gad.Int(pages), "posts": gad.Int(posts), "tags": gad.Int(tags), "menus": gad.Int(menus)})
}

func (a *App) list(w http.ResponseWriter, model any) {
	switch m := model.(type) {
	case *Page:
		var rows []Page
		a.DB.Order("id DESC").Find(&rows)
		writeJSON(w, rows)
	case *Tag:
		var rows []Tag
		a.DB.Order("name ASC").Find(&rows)
		writeJSON(w, rows)
	case *MenuItem:
		var rows []MenuItem
		a.DB.Preload("Page").Preload("Tag").Order("position ASC, id ASC").Find(&rows)
		writeJSON(w, rows)
	default:
		_ = m
	}
}

func (a *App) create(w http.ResponseWriter, r *http.Request, model any) {
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		badRequest(w, err)
		return
	}
	if err := a.DB.Create(model).Error; err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, model)
}

func (a *App) update(w http.ResponseWriter, r *http.Request, model any, id uint) {
	if err := a.DB.First(model, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		badRequest(w, err)
		return
	}
	if err := a.DB.Save(model).Error; err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, model)
}

func (a *App) delete(w http.ResponseWriter, model any, id uint) {
	if err := a.DB.Delete(model, id).Error; err != nil {
		a.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) postsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var posts []Post
		a.DB.Preload("Tags").Order("id DESC").Find(&posts)
		writeJSON(w, posts)
	case http.MethodPost:
		var input Post
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			badRequest(w, err)
			return
		}
		if err := a.savePost(&input); err != nil {
			a.serverError(w, err)
			return
		}
		writeJSON(w, input)
	default:
		methodNotAllowed(w)
	}
}

func (a *App) postMember(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var input Post
		if err := a.DB.Preload("Tags").First(&input, id).Error; err != nil {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			badRequest(w, err)
			return
		}
		input.ID = id
		if err := a.savePost(&input); err != nil {
			a.serverError(w, err)
			return
		}
		writeJSON(w, input)
	case http.MethodDelete:
		a.delete(w, &Post{}, id)
	default:
		methodNotAllowed(w)
	}
}

func (a *App) savePost(post *Post) error {
	tags := post.Tags
	post.Tags = nil
	if post.ID == 0 {
		if err := a.DB.Create(post).Error; err != nil {
			return err
		}
	} else if err := a.DB.Save(post).Error; err != nil {
		return err
	}
	return a.DB.Model(post).Association("Tags").Replace(tags)
}
