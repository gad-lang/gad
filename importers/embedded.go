package importers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
)

var _ gad.EmbeddedExtImporter = (*EmbeddedFileImporter)(nil)

// EmbeddedFileImporter is an implemention of gad.ExtImporter to import files from file
// system. It uses absolute paths of module as import names.
type EmbeddedFileImporter struct {
	NameResolver func(cwd, name string) (string, error)
	WorkDir      string
	FileReader   func(string) (data []byte, uri string, err error)
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

// Name returns the absoule path of the module. A previous Get call is required
// to get the name of the imported module.
func (i *EmbeddedFileImporter) Name() (string, error) {
	if i.name == "" {
		return "", nil
	}
	if i.NameResolver != nil {
		return i.NameResolver(i.WorkDir, i.name)
	}

	path := i.name
	if !filepath.IsAbs(path) {
		path = filepath.Join(i.WorkDir, path)
		if p, err := filepath.Abs(path); err == nil {
			path = p
		}
	}
	return path, nil
}

// Import returns the content of the path determined by Name call. Empty name
// will return an error.
func (i *EmbeddedFileImporter) Import(_ context.Context, pth string) (e *gad.Embedded, err error) {
	e = &gad.Embedded{Name: i.name}

	// Note that; moduleName == Literal()
	if i.name == "" || pth == "" {
		err = errors.New("invalid import call")
		return
	}
	if i.FileReader == nil {
		var s os.FileInfo
		if s, err = os.Stat(pth); err != nil {
			return
		}
		if s.IsDir() {
			d := gad.Dict{}
			root := pth
			filepath.Walk(pth, func(pth string, info os.FileInfo, err_ error) (err error) {
				if err_ != nil {
					return err_
				}
				if !info.IsDir() {
					var b []byte
					if b, err = os.ReadFile(pth); err != nil {
						return
					}
					if pth, err = filepath.Rel(root, pth); err != nil {
						return
					}

					d := d
					parts := strings.Split(filepath.ToSlash(pth), "/")
					for len(parts) > 1 {
						p := parts[0]

						if sub, ok := d[p]; ok {
							d = sub.(gad.Dict)
						} else {
							sub := gad.Dict{}
							d[p] = sub
							d = sub
						}

						parts = parts[1:]
					}

					d[parts[0]] = gad.Bytes(b)
				}
				return
			})
			e.Path = pth
			e.Data = d
			return
		}
		var data []byte
		if data, err = os.ReadFile(pth); err != nil {
			return
		}
		e.Path = pth
		e.Data = gad.Bytes(data)
		return
	}

	var data []byte
	if data, pth, err = i.FileReader(pth); err != nil {
		return
	}
	e.Path = pth
	e.Data = gad.Bytes(data)
	return
}

// Fork returns a new instance of EmbeddedFileImporter as gad.ExtImporter by capturing
// the working directory of the module. moduleName should be the same value
// provided by Name call.
func (i *EmbeddedFileImporter) Fork(pth string) gad.EmbeddedExtImporter {
	// Note that; moduleName == Literal()
	return &EmbeddedFileImporter{
		WorkDir:      filepath.Dir(pth),
		FileReader:   i.FileReader,
		NameResolver: i.NameResolver,
	}
}
