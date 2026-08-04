package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm/clause"
)

func (a *App) seed() error {
	data, err := os.ReadFile(filepath.Join(a.Root, "seed.yaml"))
	if err != nil {
		return fmt.Errorf("read seed.yaml: %w", err)
	}
	var sd SeedData
	if err := yaml.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("parse seed.yaml: %w", err)
	}
	return a.importSeedData(&sd)
}

func (a *App) importSeedData(sd *SeedData) error {
	var tags []Tag
	tagByName := map[string]*Tag{}
	tagBySlug := map[string]*Tag{}
	for _, st := range sd.Tags {
		t := Tag{Name: st.Name, Slug: st.Slug}
		tags = append(tags, t)
	}
	if err := a.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "slug"}}, DoNothing: true}).Create(&tags).Error; err != nil {
		return fmt.Errorf("seed tags: %w", err)
	}
	var allTags []Tag
	a.DB.Find(&allTags)
	for i := range allTags {
		tagByName[allTags[i].Name] = &allTags[i]
		tagBySlug[allTags[i].Slug] = &allTags[i]
	}

	var pages []Page
	pageBySlug := map[string]*Page{}
	for _, sp := range sd.Pages {
		p := Page{
			Title:      sp.Title,
			Slug:       sp.Slug,
			Summary:    sp.Summary,
			Body:       sp.Body,
			CoverImage: sp.CoverImage,
			Images:     sp.Images,
			Published:  sp.Published,
		}
		pages = append(pages, p)
	}
	if err := a.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "slug"}}, DoNothing: true}).Create(&pages).Error; err != nil {
		return fmt.Errorf("seed pages: %w", err)
	}
	var allPages []Page
	a.DB.Find(&allPages)
	for i := range allPages {
		pageBySlug[allPages[i].Slug] = &allPages[i]
	}

	for _, sp := range sd.Posts {
		exists := int64(0)
		a.DB.Model(&Post{}).Where("slug = ?", sp.Slug).Count(&exists)
		if exists > 0 {
			continue
		}
		p := Post{
			Title:      sp.Title,
			Slug:       sp.Slug,
			Summary:    sp.Summary,
			Body:       sp.Body,
			RightBody:  sp.RightBody,
			CoverImage: sp.CoverImage,
			Images:     sp.Images,
			Published:  sp.Published,
		}
		if err := a.DB.Create(&p).Error; err != nil {
			return fmt.Errorf("seed post %q: %w", sp.Slug, err)
		}
		var postTags []*Tag
		for _, tname := range sp.Tags {
			if t, ok := tagByName[tname]; ok {
				postTags = append(postTags, t)
			}
		}
		if err := a.DB.Model(&p).Association("Tags").Replace(postTags); err != nil {
			return fmt.Errorf("seed post %q tags: %w", sp.Slug, err)
		}
	}

	var menuItems []MenuItem
	for _, sm := range sd.Menu {
		mi := MenuItem{
			Label:    sm.Label,
			Kind:     sm.Kind,
			Position: sm.Position,
			Visible:  sm.Visible,
		}
		switch sm.Kind {
		case "url":
			mi.URL = sm.URL
		case "page":
			if p, ok := pageBySlug[sm.Page]; ok {
				mi.PageID = &p.ID
				mi.Page = p
			}
		case "tag":
			if t, ok := tagBySlug[sm.Tag]; ok {
				mi.TagID = &t.ID
				mi.Tag = t
			}
		}
		menuItems = append(menuItems, mi)
	}
	if len(menuItems) > 0 {
		a.DB.Where("1 = 1").Delete(&MenuItem{})
		return a.DB.Create(&menuItems).Error
	}
	return nil
}
