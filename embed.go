package gad

import "context"

// EmbedMap represents a set of named modules. Use NewEmbedMap to create a
// new module map.
type EmbedMap struct {
	m  map[string]Importable
	im ExtImporter
}

// NewEmbedMap creates a new module map.
func NewEmbedMap() *EmbedMap {
	return &EmbedMap{m: make(map[string]Importable)}
}

// SetExtImporter sets an ExtImporter to EmbedMap, which will be used to
// embed path dynamically.
func (m *EmbedMap) SetExtImporter(im ExtImporter) *EmbedMap {
	m.im = im
	return m
}

// Add adds an importable module.
func (m *EmbedMap) Add(name string, module Importable) *EmbedMap {
	m.m[name] = module
	return m
}

// AddFile adds a source file data.
func (m *EmbedMap) AddFile(path string, src []byte) *EmbedMap {
	m.m[path] = &EmbededFileData{Path: path, Src: src}
	return m
}

// Remove removes a named module.
func (m *EmbedMap) Remove(name string) {
	delete(m.m, name)
}

// Get returns an import module identified by name.
// It returns nil if the name is not found.
func (m *EmbedMap) Get(name string) Importable {
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
func (m *EmbedMap) Copy() *EmbedMap {
	c := &EmbedMap{m: make(map[string]Importable), im: m.im}

	for name, mod := range m.m {
		c.m[name] = mod
	}
	return c
}

// EmbededFileData is an importable embed that's written in Gad.
type EmbededFileData struct {
	Path string
	Src  []byte
}

// Import returns a embeded data.
func (m *EmbededFileData) Import(_ context.Context, name string) (any, string, error) {
	return m.Src, "source:" + name, nil
}
