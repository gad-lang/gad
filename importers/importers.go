package importers

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
)

// FileImporter is an implemention of gad.ExtImporter to import files from file
// system. It uses absolute paths of module as import names.
type FileImporter struct {
	NameResolver func(cwd, name string) (string, error)
	WorkDir      string
	FileReader   func(string) (data []byte, uri string, err error)
	name         string
}

var _ gad.ExtImporter = (*FileImporter)(nil)

// Get impelements gad.ExtImporter and returns itself if name is not empty.
func (m *FileImporter) Get(name string) gad.ExtImporter {
	if name == "" {
		return nil
	}
	m.name = name
	return m
}

// Name returns the absoule path of the module. A previous Get call is required
// to get the name of the imported module.
func (m *FileImporter) Name() (string, error) {
	if m.name == "" {
		return "", nil
	}
	if m.NameResolver != nil {
		return m.NameResolver(m.WorkDir, m.name)
	}

	path := m.name
	if !filepath.IsAbs(path) {
		path = filepath.Join(m.WorkDir, path)
		if p, err := filepath.Abs(path); err == nil {
			path = p
		}
	}
	return path, nil
}

// Import returns the content of the path determined by Name call. Empty name
// will return an error.
func (m *FileImporter) Import(ctx context.Context, module *gad.Module) (data any, uri string, err error) {
	// Note that; moduleName == Literal()
	if m.name == "" || module.Name() == "" {
		err = errors.New("invalid import call")
		return
	}
	if m.FileReader == nil {
		if data, err = os.ReadFile(module.Name()); err != nil {
			return
		}
		uri = "file:" + module.Name()
		return
	}
	return m.FileReader(module.Name())
}

// Fork returns a new instance of FileImporter as gad.ExtImporter by capturing
// the working directory of the module. moduleName should be the same value
// provided by Name call.
func (m *FileImporter) Fork(moduleName string) gad.ExtImporter {
	// Note that; moduleName == Literal()
	return &FileImporter{
		WorkDir:      filepath.Dir(moduleName),
		FileReader:   m.FileReader,
		NameResolver: m.NameResolver,
	}
}

// EmbedFileImporter is an implemention of gad.ExtImporter to import files from file
// system. It uses absolute paths of module as import names.
type EmbedFileImporter struct {
	NameResolver func(cwd, name string) (string, error)
	WorkDir      string
	FileReader   func(string) (data []byte, uri string, err error)
	name         string
}

var _ gad.ExtImporter = (*EmbedFileImporter)(nil)

// Get impelements gad.ExtImporter and returns itself if name is not empty.
func (i *EmbedFileImporter) Get(name string) gad.ExtImporter {
	if name == "" {
		return nil
	}
	i.name = name
	return i
}

// Name returns the absoule path of the module. A previous Get call is required
// to get the name of the imported module.
func (i *EmbedFileImporter) Name() (string, error) {
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
func (i *EmbedFileImporter) Import(ctx context.Context, mod *gad.Module) (data any, uri string, err error) {
	pth := mod.Name()
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
			return d, "dir:" + pth, err
		}
		if data, err = os.ReadFile(pth); err != nil {
			return
		}
		uri = "file:" + pth
		return
	}
	return i.FileReader(pth)
}

// Fork returns a new instance of EmbedFileImporter as gad.ExtImporter by capturing
// the working directory of the module. moduleName should be the same value
// provided by Name call.
func (i *EmbedFileImporter) Fork(moduleName string) gad.ExtImporter {
	// Note that; moduleName == Literal()
	return &EmbedFileImporter{
		WorkDir:      filepath.Dir(moduleName),
		FileReader:   i.FileReader,
		NameResolver: i.NameResolver,
	}
}

// OsDirsNameResolver reads given path and returns the content of the file. If file
// starts with Shebang #! , it is replaced with //.
// This function can be used as ReadFile callback in FileImporter.
func OsDirsNameResolver(dirs PathList) func(cwd, path string) (string, error) {
	return OsDirsNameResolverPtr(&dirs)
}

type PathList []string

func (d *PathList) Prepend(v string) {
	*d = append([]string{v}, *d...)
}

func (d *PathList) Append(v string) {
	*d = append(*d, v)
}

func (d *PathList) Remove(count int) {
	if count > 0 {
		*d = (*d)[count:]
	} else {
		*d = (*d)[:len(*d)+count]
	}
}

// OsDirsNameResolverPtr is similar to `OsDirsNameResolver`, but receives ptr of `dirs`.
func OsDirsNameResolverPtr(dirs *PathList) func(cwd, path string) (string, error) {
	if len(*dirs) == 0 {
		return func(_, path string) (string, error) {
			return path, nil
		}
	}
	return func(cwd string, p string) (name string, err error) {
		p = path.Clean(p)
		name = filepath.Join(cwd, p)
		if _, err = os.Stat(name); err == nil || !os.IsNotExist(err) {
			return
		}
		for _, dir := range *dirs {
			name = filepath.Join(dir, p)
			if _, err = os.Stat(name); err == nil || !os.IsNotExist(err) {
				return
			}
		}
		return "", os.ErrNotExist
	}
}

// ShebangReadFile reads given path and returns the content of the file. If file
// starts with Shebang #! , it is replaced with //.
// This function can be used as ReadFile callback in FileImporter.
func ShebangReadFile(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		Shebang2Slashes(data)
	}
	return data, "file:" + path, err
}

// Shebang2Slashes replaces first two bytes of given p with two slashes if they
// are Shebang chars.
func Shebang2Slashes(p []byte) {
	if len(p) > 1 && string(p[:2]) == "#!" {
		copy(p, "//")
	}
}
