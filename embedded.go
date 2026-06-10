package gad

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// EmbeddedExtImporter wraps methods for a embedded which will be imported dynamically like a file.
type EmbeddedExtImporter interface {
	EmbeddedImporter
	// Get returns EmbeddedExtImporter instance which will import an embedded file.
	Get(name string) EmbeddedExtImporter
	// Paths returns the unique relative and absolute paths of embedded.
	Paths() (relPath, absPath string, err error)
}

var (
	_ Object          = (*EmbeddedNodeFS)(nil)
	_ IndexGetter     = (*EmbeddedNodeFS)(nil)
	_ Iterabler       = (*EmbeddedNodeFS)(nil)
	_ Printabler      = (*EmbeddedNodeFS)(nil)
	_ ToDictConverter = (*EmbeddedNodeFS)(nil)
)

type EmbeddedNodeFS struct {
	node *Embedded
}

func (e *EmbeddedNodeFS) Print(state *PrinterState) error {
	if state.IsRepr {
		defer state.WrapRepr(e)()
	}

	var entries PrintStateDictEntries

	for name, value := range e.node.Entries {
		entries = append(entries, &PrintStateDictEntry{name, value})
	}

	return state.PrintDictEntries(entries)
}

func (e *EmbeddedNodeFS) IndexGet(_ *VM, index Object) (value Object, err error) {
	if name, _ := index.(Str); len(name) > 0 {
		var ok bool
		if value, ok = e.node.Entries[string(name)]; !ok {
			return nil, ErrInvalidIndex.NewError(string(name))
		}
	} else {
		return nil, ErrInvalidIndex
	}
	return
}

func (e *EmbeddedNodeFS) ToDict() (d Dict) {
	d = make(Dict, len(e.node.Entries))
	for k, v := range e.node.Entries {
		d[k] = v
	}
	return
}

func (e *EmbeddedNodeFS) IsFalsy() bool {
	return len(e.node.Entries) == 0
}

func (e *EmbeddedNodeFS) Type() ObjectType {
	return TEmbeddedFS
}

func (e *EmbeddedNodeFS) ToString() string {
	return fmt.Sprintf("%s of %q", TEmbeddedFS.name, e.node.Path())
}

func (e *EmbeddedNodeFS) Equal(right Object) bool {
	switch right := right.(type) {
	case *EmbeddedNodeFS:
		return e.node == right.node
	}
	return false
}

func (e *EmbeddedNodeFS) Iterate(_ *VM, na *NamedArgs) Iterator {
	keys := make([]string, 0, len(e.node.Entries))
	for k := range e.node.Entries {
		keys = append(keys, k)
	}
	if !na.GetValue("sorted").IsFalsy() || !na.MustGetValue("reversed").IsFalsy() {
		sort.Strings(keys)
	}
	return SliceEntryIteration(TEmbeddedNodeEntriesIterator, e, keys, func(v string) (_, _ Object, _ error) {
		return Str(v), e.node.Entries[v], nil
	}).ParseNamedArgs(na)
}

type EmbeddedReaderFactory interface {
	Reader(e *Embedded) (io.ReadSeeker, error)
}

type EmbeddedOsFileReaderFactory struct{}

func (s *EmbeddedOsFileReaderFactory) Reader(e *Embedded) (io.ReadSeeker, error) {
	if len(e.AbsPath) == 0 {
		return nil, errors.New("EmbeddedOsFileReaderFactory needs an absolute path")
	}
	return os.OpenFile(e.AbsPath, os.O_RDONLY, 0666)
}

type EmbeddedBytesReaderFactory []byte

func (b EmbeddedBytesReaderFactory) Reader(*Embedded) (io.ReadSeeker, error) {
	return bytes.NewReader(b), nil
}

type EmbeddedLimittedReaderFactory struct {
	AtReader io.ReaderAt
	Offset   int64
	Limit    int64
}

func (b *EmbeddedLimittedReaderFactory) Reader(*Embedded) (io.ReadSeeker, error) {
	return io.NewSectionReader(b.AtReader, b.Offset, b.Limit), nil
}

var (
	_ Object         = (*Embedded)(nil)
	_ IndexGetter    = (*Embedded)(nil)
	_ BytesConverter = (*Embedded)(nil)
	_ Printabler     = (*Embedded)(nil)
)

type Embedded struct {
	ReaderFactory EmbeddedReaderFactory
	Name          string
	Entries       map[string]*Embedded
	Parent        *Embedded
	ModTime       time.Time
	Mode          os.FileMode
	AbsPath       string
}

func (n *Embedded) Print(state *PrinterState) (err error) {
	var s []string
	if n.ReaderFactory == nil {
		s = append(s, "dir", ReprQuote(n.Path()))
	} else {
		var (
			size int64
			err  error
		)

		s = append(s, "file", ReprQuote(n.Path()))

		if size, err = n.Size(); err == nil {
			s = append(s, humanize.Bytes(uint64(size)))
		}
	}

	if len(n.AbsPath) > 0 {
		absPath := n.AbsPath
		trim, _ := state.options.TrimEmbedPath()
		if len(trim) > 0 {
			for _, v := range trim {
				absPath = strings.TrimPrefix(absPath, v.ToString())
				if len(absPath) != len(n.AbsPath) {
					break
				}
			}
		}
		s = append(s, strconv.Quote(absPath))
	}

	if state.SkipNexDepth() && len(n.Entries) > 0 {
		var fCount, dCount int
		n.Walk(func(path []string, n *Embedded) error {
			if n.IsDir() {
				dCount++
			} else {
				fCount++
			}
			return nil
		})
		if dCount+fCount > 0 {
			s = append(s, "with")
			if dCount > 0 {
				s = append(s, fmt.Sprintf("%d dirs", dCount))
			}
			if fCount > 0 {
				s = append(s, fmt.Sprintf("%d files", fCount))
			}
		}
	}

	defer state.WrapRepr(n)()
	state.WriteString(strings.Join(s, " "))

	if len(n.Entries) > 0 && !state.SkipNexDepth() {
		var (
			names   = n.SortedNames()
			entries = make([]*Embedded, len(names))
		)

		for i, name := range names {
			entries[i] = n.Entries[name]
		}

		return state.PrintArray(len(entries), func(i int) (Object, error) {
			return entries[i], nil
		})
	}

	return
}

func (n *Embedded) SortedNames() (ret []string) {
	ret = make([]string, 0, len(n.Entries))
	for name := range n.Entries {
		ret = append(ret, name)
	}
	sort.Strings(ret)
	return
}

func (n *Embedded) Get(pth string) (e *Embedded, err error) {
	if n.ReaderFactory != nil {
		return nil, NewEmbeddedPathIsNtDir(n.Path())
	}
	parts := strings.Split(pth, "/")
	last, parts := parts[len(parts)-1], parts[:len(parts)-1]

	e = n

	for i, part := range parts {
		if e = e.Entries[part]; e == nil {
			return nil, ErrEmbedded.NewErrorf("%q does not exists", path.Join(parts[:i]...))
		} else if !e.IsDir() {
			return nil, NewEmbeddedPathIsNtDir(path.Join(parts[:i]...))
		}
	}

	if e = e.Entries[last]; e == nil {
		return nil, ErrEmbedded.NewErrorf("%q does not exists", pth)
	}
	return
}

func (n *Embedded) ToBytes() (Bytes, error) {
	if n.ReaderFactory == nil {
		return nil, NewEmbeddedPathIsDir(n.Path())
	}
	return n.Read()
}

func (n *Embedded) FS() (*EmbeddedNodeFS, error) {
	if n.ReaderFactory != nil {
		return nil, NewEmbeddedPathIsNtDir(n.Path())
	}
	return &EmbeddedNodeFS{n}, nil
}

func (n *Embedded) IndexGet(vm *VM, index Object) (value Object, err error) {
	if key, _ := index.(Str); len(key) > 0 {
		switch key {
		case "parent":
			if n.Parent != nil {
				return n.Parent, nil
			}
			return Nil, nil
		case "name":
			return Str(n.Name), nil
		case "path":
			return Str(n.Path()), nil
		case "data":
			return n.ToBytes()
		case "fs":
			return n.FS()
		case "modTime":
			return vm.ToObject(n.ModTime)
		case "size":
			var size int64
			size, err = n.Size()
			return Int(size), err
		case "reader":
			if n.ReaderFactory != nil {
				var r io.ReadSeeker
				if r, err = n.ReaderFactory.Reader(nil); err != nil {
					return nil, err
				}
				return NewReader(r), nil
			} else {
				return nil, ErrEmbedded.NewError("is dir")
			}
		default:
			return n.Get(string(key))
		}
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

func (n *Embedded) IsFalsy() bool {
	return false
}

func (n *Embedded) Type() ObjectType {
	return TEmbedded
}

func (n *Embedded) Size() (_ int64, err error) {
	var r io.ReadSeeker
	if r, err = n.Reader(); err != nil {
		return
	}

	switch t := r.(type) {
	case interface{ Size() int }:
		return int64(t.Size()), nil
	case interface{ Size() int64 }:
		return t.Size(), nil
	}

	return r.Seek(0, io.SeekEnd)
}

func (n *Embedded) Read() (b []byte, err error) {
	var r io.Reader
	if r, err = n.Reader(); err != nil {
		return
	}
	return io.ReadAll(r)
}

func (n *Embedded) Reader() (r io.ReadSeeker, err error) {
	if n.ReaderFactory == nil {
		return nil, ErrEmbedded.NewError("is dir")
	}
	return n.ReaderFactory.Reader(n)
}

func (n *Embedded) ToString() string {
	return string(MustToStr(nil, n))
}

func (n *Embedded) Files(recursive bool) (ret []*Embedded) {
	n.WalkR(recursive, func(path []string, n *Embedded) error {
		if !n.IsDir() {
			ret = append(ret, n)
		}
		return nil
	})
	return
}

func (n *Embedded) Dirs(recursive bool) (ret []*Embedded) {
	n.WalkR(recursive, func(path []string, n *Embedded) error {
		if !n.IsDir() {
			ret = append(ret, n)
		}
		return nil
	})
	return
}

func (n *Embedded) JoinToArray() (ret []*Embedded) {
	n.Walk(func(path []string, n *Embedded) error {
		ret = append(ret, n)
		return nil
	})
	return
}

func (n *Embedded) Equal(right Object) bool {
	if r, ok := right.(*Embedded); ok {
		return r == n
	}
	return false
}

func (n *Embedded) GetNode(name string) *Embedded {
	return n.Entries[name]
}

func (n *Embedded) IsDir() bool {
	return n.ReaderFactory == nil
}

func (n *Embedded) Path() string {
	var pth []string
	for n != nil {
		pth = append(pth, n.Name)
		n = n.Parent
	}

	for i, j := 0, len(pth)-1; i < j; i, j = i+1, j-1 {
		pth[i], pth[j] = pth[j], pth[i]
	}

	return strings.Join(pth, "/")
}

func (n *Embedded) FullPath() string {
	var pth []string
	for n != nil {
		if len(n.AbsPath) > 0 {
			pth = append(pth, n.AbsPath)
			break
		}
		pth = append(pth, n.Name)
		n = n.Parent
	}

	for i, j := 0, len(pth)-1; i < j; i, j = i+1, j-1 {
		pth[i], pth[j] = pth[j], pth[i]
	}

	return strings.Join(pth, "/")
}

func (n *Embedded) Walk(cb func(path []string, n *Embedded) error) (err error) {
	return n.walk(nil, cb)
}

func (n *Embedded) WalkR(recursive bool, cb func(path []string, n *Embedded) error) (err error) {
	if !recursive {
		for _, node := range n.Entries {
			if err = cb([]string{node.Name}, node); err != nil {
				return
			}
		}
		return
	}
	return n.walk(nil, cb)
}

func (n *Embedded) walk(path []string, cb func(path []string, n *Embedded) error) (err error) {
	for _, node := range n.Entries {
		if node.ReaderFactory == nil {
			if err = node.walk(append(path, n.Name), cb); err != nil {
				return
			}
		} else if err = cb(append(path, n.Name), node); err != nil {
			return
		}
	}
	return
}

type EmbeddedImportOptions struct {
	Sources    []string
	Includes   []string
	Excludes   []string
	IncludesRe []string
	ExcludesRe []string
	ConfigFile string
	Tree       bool
}

// EmbeddedImporter interface represents importable embedded instance.
type EmbeddedImporter interface {
	// Import should return either an Object or module source code ([]byte).
	Import(ctx context.Context, relPath string, absPath string, opts *EmbeddedImportOptions) (data *Embedded, err error)
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
func (m *EmbeddedMap) AddFile(path string, data []byte) *EmbeddedMap {
	m.m[path] = EmbeddedFileData(data)
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

// EmbeddedFileData is an importable embed for data that's written in Gad.
type EmbeddedFileData []byte

func (m EmbeddedFileData) Import(_ context.Context, name, _ string, _ *EmbeddedImportOptions) (*Embedded, error) {
	return &Embedded{
		Name:          name,
		ReaderFactory: EmbeddedBytesReaderFactory(m),
	}, nil
}

// EmbeddedFile is an importable embed that's written in Gad.
type EmbeddedFile Embedded

func (m EmbeddedFile) Import(_ context.Context, _, _ string, _ *EmbeddedImportOptions) (*Embedded, error) {
	e := Embedded(m)
	return &e, nil
}
