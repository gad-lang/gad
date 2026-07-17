package giom

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	giomnode "github.com/gad-lang/gad/giom/node"
	giomparser "github.com/gad-lang/gad/giom/parser"
)

// FileImporter imports Gad and Giom files from the filesystem.
//
// Files ending in .giom are returned as gad.BuiltinCompileModuleFunc so they are
// parsed and compiled with Giom syntax during Gad import compilation.
type FileImporter struct {
	NameResolver  func(cwd, name string) (string, error)
	WorkDir       string
	FileReader    func(string) (data []byte, uri string, err error)
	TranspilePath func(srcPath string) string
	name          string
}

var _ gad.ExtImporter = (*FileImporter)(nil)

// Get returns this importer for a non-empty module name.
func (m *FileImporter) Get(name string) gad.ExtImporter {
	if name == "" {
		return nil
	}
	m.name = name
	return m
}

// Name resolves the current import name into the compiler module cache key.
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
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	return path, nil
}

// Import reads the resolved module. Giom modules are compiled through a builtin
// module compiler function; other files are returned as source bytes.
func (m *FileImporter) Import(ctx context.Context, module *gad.ModuleSpec) (data any, uri string, err error) {
	if m.name == "" || module.Name == "" {
		return nil, "", errors.New("invalid import call")
	}

	src, uri, err := m.readFile(module.Name)
	if err != nil {
		return nil, "", err
	}
	if filepath.Ext(module.Name) != ".giom" {
		return src, uri, nil
	}

	compile := func(ctx *gad.BuiltinCompileModuleContext) (bc *gad.Bytecode, err error) {
		file := ctx.SetFileData(src)
		if rel, err := filepath.Rel(m.WorkDir, module.Name); err == nil {
			file.Name = rel
		}
		p := giomparser.NewParser(file)
		parsed, err := p.ParseFile()
		if err != nil {
			return nil, ctx.Compiler.Errorf(ctx.Node, "parse file %q error: %w", file.Name, err)
		}

		if m.TranspilePath != nil {
			if outPath := m.TranspilePath(module.Name); outPath != "" {
				if err := Transpile(module.Name, src, outPath); err != nil {
					return nil, err
				}
			}
		}

		if err = ctx.Compile(parsed.Stmts); err != nil {
			return nil, err
		}

		bc = ctx.Compiler.Bytecode()

		return
	}
	return gad.BuiltinCompileModuleFunc(compile), uri, nil
}

// Fork returns an importer rooted at the imported module's directory.
func (m *FileImporter) Fork(moduleName string) gad.ExtImporter {
	return &FileImporter{
		WorkDir:       filepath.Dir(moduleName),
		FileReader:    m.FileReader,
		NameResolver:  m.NameResolver,
		TranspilePath: m.TranspilePath,
	}
}

func (m *FileImporter) readFile(path string) ([]byte, string, error) {
	if m.FileReader != nil {
		return m.FileReader(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return data, "file:" + path, nil
}

func writeTranspiled(outPath string, stmts gnode.Stmts) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("create transpile dir: %w", err)
	}
	converted := giomnode.ConvertFile(stmts)
	var buf bytes.Buffer
	gnode.CodeW(&buf, converted, gnode.CodeWithPrefix("\t"), gnode.CodeFormat())
	if !strings.HasSuffix(outPath, ".gad") {
		outPath += ".gad"
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write transpiled %s: %w", outPath, err)
	}
	return nil
}

// Transpile parses Giom source and writes the converted Gad source to outPath.
func Transpile(name string, src []byte, outPath string) error {
	fileSet := source.NewFileSet()
	file := fileSet.AddFileData(name, -1, src)
	p := giomparser.NewParser(file)
	parsed, err := p.ParseFile()
	if err != nil {
		return err
	}
	return writeTranspiled(outPath, parsed.Stmts)
}
