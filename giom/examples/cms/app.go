package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	giom "github.com/gad-lang/gad/giom"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type App struct {
	DB           *gorm.DB
	Root         string
	PublicDir    string
	TranspileDir string

	renderer *giom.Render
}

func NewApp(root string) (*App, error) {
	dbPath := filepath.Join(root, "cms.db")
	_, err := os.Stat(dbPath)
	firstRun := os.IsNotExist(err)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	app := &App{
		DB:           db,
		Root:         root,
		PublicDir:    filepath.Join(root, "public"),
		TranspileDir: filepath.Join(root, "public", ".transpiled"),
		renderer:     giom.NewRender(filepath.Join(root, "public")),
	}
	app.renderer.TemplateDelay = 1 * time.Second
	app.renderer.TranspilePath = app.transpilePath
	stderrIsTTY := isTerminal(os.Stderr)
	app.renderer.OnRender(func(first bool, mainFile string, files []string, lastTime time.Time, err error) {
		if err != nil {
			if stderrIsTTY {
				fmt.Fprintf(os.Stderr, "\033[31m[giom] error compiling %s: %v\033[0m\n", mainFile, err)
			} else {
				log.Printf("[giom] error compiling %s: %v", mainFile, err)
			}
			return
		}
		if first {
			log.Printf("[giom] first render: %s", mainFile)
		} else {
			log.Printf("[giom] recompile: %s (changed: %v)", mainFile, files)
		}
	})
	app.cleanTranspiled()
	if err := app.DB.AutoMigrate(&Page{}, &Tag{}, &Post{}, &MenuItem{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if firstRun {
		if err := app.seed(); err != nil {
			return nil, err
		}
	}
	return app, nil
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func (a *App) cleanTranspiled() {
	os.RemoveAll(a.TranspileDir)
}
