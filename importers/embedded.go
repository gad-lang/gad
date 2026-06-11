package importers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gad-lang/gad"
)

func matchFilePath(opts *gad.EmbeddedImportOptions, relPath string) bool {
	base := filepath.Base(relPath)

	for _, pattern := range opts.Excludes {
		if matched, _ := filepath.Match(pattern, base); matched {
			return false
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return false
		}
	}
	for _, pattern := range opts.ExcludesRe {
		if matched, _ := regexp.MatchString(pattern, relPath); matched {
			return false
		}
	}

	if len(opts.Includes) > 0 {
		included := false
		for _, pattern := range opts.Includes {
			if matched, _ := filepath.Match(pattern, base); matched {
				included = true
				break
			}
			if matched, _ := filepath.Match(pattern, relPath); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	if len(opts.IncludesRe) > 0 {
		included := false
		for _, pattern := range opts.IncludesRe {
			if matched, _ := regexp.MatchString(pattern, relPath); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	return true
}

var _ gad.EmbeddedExtImporter = (*EmbeddedFileImporter)(nil)

// EmbeddedFileImporter is an implemention of gad.ExtImporter to import files from file
// system. It uses absolute paths of module as import names.
type EmbeddedFileImporter struct {
	NameResolver func(dirs []string, name string) (relPath, absPath string, err error)
	WorkDirs     []string
	AbsDisabled  bool
	name         string
}

// Get impelements gad.ExtImporter and returns itself if name is not empty.
func (i *EmbeddedFileImporter) Get(name string) gad.EmbeddedExtImporter {
	if name == "" {
		return nil
	}
	i.name = name
	return i
}

// Paths returns the paths of the file.
func (i *EmbeddedFileImporter) Paths() (name, absPath string, err error) {
	if i.name == "" {
		return
	}

	if i.NameResolver != nil {
		return i.NameResolver(i.WorkDirs, i.name)
	}

	return ResolvePath(i.WorkDirs, !i.AbsDisabled, i.name)
}

// Import returns the content of the path determined by Name call. Empty name
// will return an error.
func (i *EmbeddedFileImporter) Import(ctx context.Context, name string, absPath string, opts *gad.EmbeddedImportOptions) (e *gad.Embedded, err error) {
	if name == "" {
		err = errors.New("invalid import")
		return
	}

	name = filepath.Clean(name)

	if strings.HasPrefix(name, ".") {
		err = errors.New("invalid import path")
		return
	}

	if opts == nil {
		opts = new(gad.EmbeddedImportOptions)
	}

	e = &gad.Embedded{Name: name}

	var (
		s           os.FileInfo
		ready       = make(map[string]bool)
		resolvePath = func(absPath string, s os.FileInfo) (_ string, _ os.FileInfo, err error) {
			for !s.IsDir() && !s.Mode().IsRegular() {
				if _, ok := ready[absPath]; ok {
					err = errors.New("link loop: " + absPath)
					return
				}
				ready[absPath] = true

				if absPath, err = os.Readlink(absPath); err != nil {
					return
				}
			}

			if !filepath.IsAbs(absPath) {
				if absPath, err = filepath.Abs(absPath); err != nil {
					return
				}
			}

			for _, dir := range i.WorkDirs {
				if strings.HasPrefix(absPath, dir) {
					if s, err = os.Stat(absPath); err != nil {
						return
					}
					return absPath, s, nil
				}
			}
			err = fmt.Errorf("invalid import path: %s", absPath)
			return
		}
		addNode = func(root string, abs string, info os.FileInfo) (err error) {
			var pth string
			if pth, err = filepath.Rel(root, abs); err != nil {
				return
			}

			parts := strings.Split(pth, string(filepath.Separator))
			dir := e

			for len(parts) > 1 {
				p := parts[0]

				if sub, ok := dir.Entries[p]; ok {
					dir = sub
				} else {
					sub := new(gad.Embedded)
					sub.Parent = dir
					sub.Name = p
					sub.Entries = make(map[string]*gad.Embedded)
					if dir.Entries == nil {
						dir.Entries = make(map[string]*gad.Embedded)
					}
					dir.Entries[p] = sub
					dir = sub
				}

				parts = parts[1:]
			}

			if dir.Entries == nil {
				dir.Entries = make(map[string]*gad.Embedded)
			}
			dir.Entries[parts[0]] = &gad.Embedded{
				Name:          parts[0],
				ModTime:       info.ModTime(),
				Mode:          info.Mode(),
				ReaderFactory: &gad.EmbeddedOsFileReaderFactory{},
				Parent:        dir,
				AbsPath:       abs,
			}

			return
		}
	)

	// if not resolved
	if len(absPath) == 0 {
		sources := opts.Sources
		if len(sources) == 0 {
			sources = []string{"."}
		}

		if opts.Tree && len(sources) > 1 {
			for _, src := range sources {
				for _, dir := range i.WorkDirs {
					pth := filepath.Join(dir, src)
					if s, err = os.Stat(pth); err != nil {
						err = nil
						continue
					}

					if !s.IsDir() {
						if pth, s, err = resolvePath(pth, s); err != nil {
							return
						}
					}

					if !s.IsDir() {
						continue
					}

					err = filepath.Walk(pth, func(epth string, info os.FileInfo, err_ error) (err error) {
						if err_ != nil {
							return err_
						}
						if !info.IsDir() {
							if relPath, rErr := filepath.Rel(pth, epth); rErr == nil {
								if !matchFilePath(opts, relPath) {
									return
								}
							}
							err = addNode(pth, epth, info)
						}
						return
					})
					if err != nil {
						return
					}
				}
			}

			return
		} else {
		sources:
			for _, src := range sources {
				for _, dir := range i.WorkDirs {
					pth := filepath.Join(dir, filepath.FromSlash(src), filepath.Clean(name))
					if s, err = os.Stat(pth); err != nil {
						err = nil
						continue
					}
					e.AbsPath = pth
					break sources
				}
			}
		}
	} else if s, err = os.Stat(absPath); err != nil {
		return
	} else {
		e.AbsPath = absPath
	}

	if e.AbsPath, s, err = resolvePath(e.AbsPath, s); err != nil {
		return
	}

	if s == nil {
		return nil, os.ErrNotExist
	}

	if s.IsDir() {
		err = filepath.Walk(e.AbsPath, func(pth string, info os.FileInfo, err_ error) (err error) {
			if err_ != nil {
				return err_
			}
			if !info.IsDir() {
				if relPath, rErr := filepath.Rel(e.AbsPath, pth); rErr == nil {
					if !matchFilePath(opts, relPath) {
						return
					}
				}
				err = addNode(e.AbsPath, pth, info)
			}
			return
		})
	} else {
		e.Mode = s.Mode()
		e.ModTime = s.ModTime()
		e.ReaderFactory = &gad.EmbeddedOsFileReaderFactory{}
	}
	return
}
