package importers

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"

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
func (m *FileImporter) Import(_ context.Context, moduleName string) (data any, url string, err error) {
	// Note that; moduleName == Literal()
	if m.name == "" || moduleName == "" {
		err = errors.New("invalid import call")
		return
	}
	if m.FileReader == nil {
		if data, err = os.ReadFile(moduleName); err != nil {
			return
		}
		url = "file:" + moduleName
		return
	}
	return m.FileReader(moduleName)
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

// OsDirsNameResolver reads given path and returns the content of the file. If file
// starts with Shebang #! , it is replaced with //.
// This function can be used as ReadFile callback in FileImporter.
func OsDirsNameResolver(dirs []string) func(cwd, path string) (string, error) {
	if len(dirs) == 0 {
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
		for _, dir := range dirs {
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
