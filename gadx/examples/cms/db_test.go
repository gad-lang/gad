package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAppSetup(t *testing.T) {
	app := newTestApp(t)
	if app.DB == nil {
		t.Fatal("expected DB to be set")
	}
	if app.PublicDir == "" {
		t.Fatal("expected PublicDir to be set")
	}
	mux := http.NewServeMux()
	app.routes(mux)
}

func TestTranspiledFilesCleanup(t *testing.T) {
	root := t.TempDir()
	publicDir := filepath.Join(root, "public")
	os.MkdirAll(publicDir, 0755)
	transpiledDir := filepath.Join(publicDir, ".transpiled")
	os.MkdirAll(transpiledDir, 0755)

	stale := filepath.Join(transpiledDir, "stale.gad")
	os.WriteFile(stale, []byte("stale"), 0644)

	db, _ := gorm.Open(sqlite.Open(filepath.Join(root, "cms.db")), &gorm.Config{})
	app := &App{
		DB:           db,
		Root:         root,
		PublicDir:    publicDir,
		TranspileDir: transpiledDir,
	}
	db.AutoMigrate(&Page{}, &Tag{}, &Post{}, &MenuItem{})
	app.cleanTranspiled()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("expected stale transpiled file to be removed")
	}
}
