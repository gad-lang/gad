package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/gad-lang/gad"
)

func (a *App) transpilePath(srcPath string) string {
	rel, err := filepath.Rel(a.PublicDir, srcPath)
	if err != nil {
		rel = filepath.Base(srcPath)
	}
	return filepath.Join(a.TranspileDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".gad")
}

func (a *App) model(title string, crumbs []crumb, values gad.Dict) gad.Dict {
	model := gad.Dict{
		"SiteTitle":   gad.Str("GION CMS"),
		"Title":       gad.Str(title),
		"Menu":        gad.MustNewReflectValue(a.menuItems()),
		"Breadcrumbs": crumbsValue(crumbs),
		"Year":        gad.Int(time.Now().Year()),
	}
	for k, v := range values {
		model[k] = v
	}
	return model
}
