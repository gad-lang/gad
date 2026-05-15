package gad

import (
	"context"
	"fmt"
)

// EmbeddedExtImporter wraps methods for a module which will be impored dynamically like
// a file.
type EmbeddedExtImporter interface {
	EmbeddedImporter
	// Get returns Extimporter instance which will import a module.
	Get(moduleName string) EmbeddedExtImporter
	// Name returns the full name of the module e.g. absoule path of a file.
	// Import names are generally relative, this overwrites module name and used
	// as unique key for compiler module cache.
	Name() (string, error)
	// Fork returns an EmbeddedExtImporter instance which will be used to import the
	// modules. Fork will get the result of Name() if it is not empty, otherwise
	// module name will be same with the Get call.
	Fork(moduleName string) EmbeddedExtImporter
}

// Embedded represents an Embedded Object
type Embedded struct {
	Name string
	Path string
	Data Object
}

func (e *Embedded) IsFalsy() bool {
	return e.Data.IsFalsy()
}

func (e *Embedded) Type() ObjectType {
	return TEmbedded
}

func (e *Embedded) ToString() string {
	r, _ := ToRepr(nil, e.Data)
	return ReprQuote(ReprQuote(e.Path) + " path=" + ReprQuote(e.Path) + " data=" + string(r))
}

func (e *Embedded) Print(state *PrinterState) error {
	defer state.WrapRepr(e)()
	fmt.Fprintf(state, "%s path=%s ", ReprQuote(e.Name), ReprQuote(e.Path))

	return state.WithRepr(func(s *PrinterState) error {
		return s.Print(e.Data)
	})
}

func (e *Embedded) Equal(right Object) bool {
	if r, _ := right.(*Embedded); r == e {
		return true
	}
	return false
}

// EmbeddedImporter interface represents importable embedded instance.
type EmbeddedImporter interface {
	// Import should return either an Object or module source code ([]byte).
	Import(ctx context.Context, pth string) (data *Embedded, err error)
}

// EmbeddedMap represents a set of named modules. Use NewEmbedMap to create a
// new module map.
type EmbeddedMap struct {
	m  map[string]EmbeddedImporter
	im EmbeddedExtImporter
}

// NewEmbedMap creates a new module map.
func NewEmbedMap() *EmbeddedMap {
	return &EmbeddedMap{m: make(map[string]EmbeddedImporter)}
}

// SetExtImporter sets an ExtImporter to EmbeddedMap, which will be used to
// embed path dynamically.
func (m *EmbeddedMap) SetExtImporter(im EmbeddedExtImporter) *EmbeddedMap {
	m.im = im
	return m
}

// Add adds an importable module.
func (m *EmbeddedMap) Add(name string, module EmbeddedImporter) *EmbeddedMap {
	m.m[name] = module
	return m
}

// AddFile adds a source file data.
func (m *EmbeddedMap) AddFile(path string, src []byte) *EmbeddedMap {
	m.m[path] = &EmbeddedFileData{Path: path, Src: src}
	return m
}

// Remove removes a named module.
func (m *EmbeddedMap) Remove(name string) {
	delete(m.m, name)
}

// Get returns an import module identified by name.
// It returns nil if the name is not found.
func (m *EmbeddedMap) Get(name string) EmbeddedImporter {
	if m == nil {
		return nil
	}

	v, ok := m.m[name]
	if ok || m.im == nil {
		return v
	}
	return m.im.Get(name)
}

// Copy creates a copy of the module map.
func (m *EmbeddedMap) Copy() *EmbeddedMap {
	c := &EmbeddedMap{m: make(map[string]EmbeddedImporter), im: m.im}

	for name, mod := range m.m {
		c.m[name] = mod
	}
	return c
}

// EmbeddedFileImporter is an implemention of gad.ExtImporter to import files from file
// system. It uses absolute paths of module as import names.
type EmbeddedFileImporter struct {
	NameResolver func(cwd, name string) (string, error)
	WorkDir      string
	FileReader   func(string) (data []byte, uri string, err error)
}

// EmbeddedFileData is an importable embed that's written in Gad.
type EmbeddedFileData struct {
	Path string
	Src  Bytes
}

// Import returns a embeded data.
func (m *EmbeddedFileData) Import(context.Context, string) (*Embedded, error) {
	return &Embedded{Name: m.Path, Path: m.Path, Data: m.Src}, nil
}
