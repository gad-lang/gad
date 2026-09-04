package importers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
)

// FileImporter is an implemention of gad.ExtImporter to import files from file
// system. It uses absolute paths of module as import names.
//
// Import returns a gad.SourceCode pairing the file bytes with the dialect chosen
// from its extension (.gadt -> template, .gadx -> Gadx, otherwise plain Gad), so
// the compiler parses each module with the matching front-end.
type FileImporter struct {
	NameResolver func(cwd, name string) (string, error)
	WorkDir      string
	// Root is the base a plain import name resolves against, and the boundary
	// no import may cross. It stays put as imports nest, while WorkDir follows
	// the module being compiled, which is what makes the two forms differ:
	//
	//	"layouts/default.gadx"    from Root — the same name means the same file,
	//	                          whichever module writes it
	//	"./sibling.gadx"          from WorkDir, the importing module's directory
	//	"../comps.gadx"           likewise, and refused if it climbs above Root
	//
	// Empty Root keeps the older behaviour, where every name resolved against
	// WorkDir and nesting shifted what a plain name meant.
	Root       string
	FileReader func(string) (data []byte, uri string, err error)
	// TranspilePath, when set and non-empty for a ".gadx" module, is the output
	// path its transpiled Gad source is written to on import (see
	// gad.TranspileGadx).
	TranspilePath func(srcPath string) string
	name          string
}

// root reports the boundary for this importer, falling back to WorkDir so that
// the first fork of an importer created without one still gets a root.
func (m *FileImporter) root() string {
	if m.Root != "" {
		return m.Root
	}
	return m.WorkDir
}

// isRelative reports whether name asks to be resolved against the importing
// module's directory rather than the root.
func isRelative(name string) bool {
	return strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") ||
		name == "." || name == ".."
}

// ErrImportOutsideRoot is returned for an import that climbs above the root.
var ErrImportOutsideRoot = errors.New("import path escapes the root directory")

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

	pth := m.name
	if filepath.IsAbs(pth) {
		return pth, nil
	}

	root := m.root()
	base := root
	if root != "" && isRelative(pth) {
		// Relative to the module doing the importing, not to the root.
		base = m.WorkDir
	}

	pth = filepath.Join(base, pth)
	if p, err := filepath.Abs(pth); err == nil {
		pth = p
	}

	if root != "" {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			absRoot = root
		}
		rel, err := filepath.Rel(absRoot, pth)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("%w: %s", ErrImportOutsideRoot, m.name)
		}
	}

	return pth, nil
}

// Import returns the module source paired with its dialect as a gad.SourceCode.
// The dialect is chosen from the file extension (.gadt -> template, .gadx ->
// Gadx, otherwise plain Gad); the compiler parses the bytes with the matching
// front-end. Empty name returns an error.
func (m *FileImporter) Import(ctx context.Context, module *gad.ModuleSpec) (data any, uri string, err error) {
	// Note that; moduleName == Literal()
	if m.name == "" || module.Name == "" {
		err = errors.New("invalid import call")
		return
	}

	var src []byte
	if m.FileReader == nil {
		if src, err = os.ReadFile(module.Name); err != nil {
			return
		}
		// Gad addresses modules with forward slashes; normalise so uris are
		// consistent across OSes (Windows filepath.Join yields backslashes).
		uri = "file:" + filepath.ToSlash(module.Name)
	} else if src, uri, err = m.FileReader(module.Name); err != nil {
		return
	}

	kind := gad.SourceKindForExt(module.Name)
	// A .gadx module may additionally be transpiled to readable Gad on import.
	if kind == gad.SourceKindGadx && m.TranspilePath != nil {
		if outPath := m.TranspilePath(module.Name); outPath != "" {
			if terr := gad.TranspileGadx(module.Name, src, outPath); terr != nil {
				return nil, "", terr
			}
		}
	}
	data = gad.SourceCode{Data: src, Kind: kind}
	return
}

// Fork returns a new instance of FileImporter as gad.ExtImporter by capturing
// the working directory of the module. moduleName should be the same value
// provided by Name call.
func (m *FileImporter) Fork(moduleName string) gad.ExtImporter {
	// Note that; moduleName == Literal()
	return &FileImporter{
		WorkDir: filepath.Dir(moduleName),
		// Carried through so nesting never moves the root: a plain name means
		// the same file however deep the import chain goes.
		Root:          m.root(),
		FileReader:    m.FileReader,
		NameResolver:  m.NameResolver,
		TranspilePath: m.TranspilePath,
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
	// Forward-slash uri for cross-OS consistency (see FileImporter.Import).
	return data, "file:" + filepath.ToSlash(path), err
}

// Shebang2Slashes replaces first two bytes of given p with two slashes if they
// are Shebang chars.
func Shebang2Slashes(p []byte) {
	if len(p) > 1 && string(p[:2]) == "#!" {
		copy(p, "//")
	}
}
