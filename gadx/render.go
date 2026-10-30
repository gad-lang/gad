package gadx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/importers"
)

type templateCacheEntry struct {
	bc       *gad.Bytecode
	builtins *gad.Builtins
	// ifaceCache memoizes interface-satisfaction checks across renders of this
	// compiled template. It is created fresh per compile, so it is automatically
	// reset when the template recompiles (its bytecode changes).
	ifaceCache *gad.InterfaceSatCache
	files      map[string]time.Time
	// dirs are the directories the template depends on by their listing (a
	// glob's, a SourceFiles module's), each with its signature when compiled
	dirs       map[string]dirState
	changedAt  time.Time
	compiledAt time.Time
}

type trackingReader struct {
	files map[string]struct{}
	// dirs are the directories depended on by their listing, with how deep
	// (gad.GlobDepth: -1 all the way down)
	dirs map[string]int
}

func newTrackingReader() *trackingReader {
	return &trackingReader{files: make(map[string]struct{}), dirs: make(map[string]int)}
}

// dir records the directory dir as depended on to depth: the deepest asked
// wins.
func (r *trackingReader) dir(dir string, depth int) {
	if old, ok := r.dirs[dir]; ok && (old < 0 || (depth >= 0 && depth <= old)) {
		return
	}
	r.dirs[dir] = depth
}

func (r *trackingReader) Read(path string) ([]byte, string, error) {
	r.files[path] = struct{}{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return data, "file:" + path, nil
}

// SourceFile is a module an application adds to the templates' module map
// (Render.ModuleMapFunc) that is read from a file: SourceFile is its path. A
// template that imports it recompiles when the file changes — or is gone.
type SourceFile interface {
	gad.Importable
	SourceFile() string
}

// SourceFiles is a module an application adds to the templates' module map
// (Render.ModuleMapFunc) that is made of several files — a directory of
// modules it loads itself, say —: SourceFiles are their paths. A path that is
// a directory is depended on by its whole tree — a file added, taken, renamed
// or written in it —; any other, as a SourceFile's. A template that imports
// the module recompiles when one of them changes. The paths are asked each
// time a template importing it compiles.
type SourceFiles interface {
	gad.Importable
	SourceFiles() []string
}

// trackedImport is a SourceFile (or SourceFiles) whose import records its
// files as ones the template depends on.
type trackedImport struct {
	gad.Importable
	files func() []string
	tr    *trackingReader
}

func (t *trackedImport) Import(ctx context.Context, module *gad.ModuleSpec) (any, string, error) {
	for _, p := range t.files() {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		// a directory — or a path not there yet: depended on as one, so
		// that it appearing (a directory, or a file) recompiles
		if fi, err := os.Stat(p); err != nil || fi.IsDir() {
			t.tr.dir(p, -1)
		} else {
			t.tr.files[p] = struct{}{}
		}
	}
	return t.Importable.Import(ctx, module)
}

// Render handles Gadx template rendering with bytecode caching and
// automatic recompilation on file changes. It is safe for concurrent use.
type Render struct {
	// TemplateDelay is the debounce duration before recompiling after
	// a file change is detected. Defaults to 15s.
	TemplateDelay time.Duration

	// workDir is the directory used as the base for resolving module
	// imports via FileImporter. Defaults to the directory of the
	// rendered file if empty.
	workDir string

	// Fallback is the template tree a template is read from when it is not
	// in workDir: the rendered file, a module it imports (a name, a glob),
	// at the same path below workDir (importers.FileImporter.Fallback). A
	// file in workDir always wins, file by file — an application overrides a
	// library's templates by writing its own, and one written later
	// recompiles the templates that read the library's. It needs workDir.
	Fallback fs.FS

	// TranspilePath returns the output path for transpiled .gad files.
	// If nil, transpilation is skipped.
	TranspilePath func(srcPath string) string

	// Inspect, when set, is given every source file of the template each time
	// it compiles — on the first render and after a change —, parsed as the
	// compiler parses it (gad.ParseSource): the rendered file, and every module
	// it imports and file it includes, recursively. An application looks at
	// what the templates say through it (the texts they ask for, where); its
	// error fails the compilation. Every file of a compilation has been given
	// when the OnRender callbacks run.
	Inspect importers.InspectFunc

	// BuiltinsFunc returns the Gad builtins to use for compilation.
	// If nil, defaults to AppendBuiltins(gad.NewBuiltins()).
	BuiltinsFunc func() *gad.Builtins

	ModuleMapFunc func(mm *gad.ModuleMap) *gad.ModuleMap

	// RunOptsFunc, when set, is called with the *gad.RunOpts assembled for each
	// render just before execution, letting the caller customise it (flags,
	// env, stdin/stderr, …). The opts are pre-initialised with StdOut, Globals
	// and gad.RunFlagSkipReceiverTypeCheck; the callback may add to or override
	// any field. StdOut and Globals are set by Render and normally left as-is.
	RunOptsFunc func(opts *gad.RunOpts)

	// AttrHandler, when set, is given each attribute of every tag the
	// render writes, and says the value written (see AttrHandler): to make
	// the URIs of the static files versioned, say, with no template changed.
	AttrHandler AttrHandler

	mu             sync.Mutex
	compileMu      sync.Mutex
	templateCache  map[string]*templateCacheEntry
	onRenderFuncs  []func(first bool, mainFile string, files []string, lastTime time.Time, err error)
	cachedBuiltins *gad.Builtins
	// cachedStatic is the built (immutable) form of cachedBuiltins, produced once
	// under builtinsOnce. Building mutates shared builtin source objects (lazy
	// caches), so it must happen single-threaded; renders then reuse this
	// read-only result instead of calling Build() concurrently.
	cachedStatic *gad.StaticBuiltins
	builtinsOnce sync.Once
}

// NewRender creates a Render with the given workDir. Non-empty paths are
// resolved to an absolute path. Other fields (TemplateDelay, TranspilePath,
// BuiltinsFunc) may be set on the returned value before use.
func NewRender(workDir string) *Render {
	if workDir != "" {
		abs, err := filepath.Abs(workDir)
		if err == nil {
			workDir = abs
		}
	}
	return &Render{
		workDir:       workDir,
		TemplateDelay: time.Second * 15,
	}
}

// WorkDir returns the base directory for resolving module imports.
func (r *Render) WorkDir() string { return r.workDir }

// OnRender appends callback functions invoked after compilation.
// On first render, first=true with empty files and zero lastTime.
// On recompilation due to file changes, first=false with the changed file paths.
// If compilation fails, err is non-nil and the cached entry is not updated.
// Returns the Render for chaining.
func (r *Render) OnRender(f ...func(first bool, mainFile string, files []string, lastTime time.Time, err error)) *Render {
	r.onRenderFuncs = append(r.onRenderFuncs, f...)
	return r
}

// HandlerFunc returns an http.HandlerFunc that renders filePath for every
// request, ready to hand to http.HandleFunc / http.Handle. model builds the
// template globals from the request (return a nil Dict for none); pass nil when
// the template needs no request data.
//
// The template is rendered into a buffer first, so a model or render error
// yields a clean 500 with no partial body. On success the HTML is written with a
// "text/html; charset=utf-8" Content-Type. For full control over status codes,
// streaming or headers, call Render directly from your own handler.
func (r *Render) HandlerFunc(filePath string, model func(*http.Request) (gad.Dict, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var globals gad.Dict
		if model != nil {
			g, err := model(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			globals = g
		}
		var buf bytes.Buffer
		if err := r.Render(&buf, filePath, globals); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = buf.WriteTo(w)
	}
}

// Render reads the Gadx template at filePath, compiles or retrieves cached
// bytecode, and executes it with the keys of globals available as global
// variables, writing the output to out.
func (r *Render) Render(out io.Writer, filePath string, globals gad.Dict) error {
	src, _, err := importers.ReadFallback(r.Fallback, r.workDir, filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}

	delay := r.TemplateDelay
	if delay <= 0 {
		delay = 15 * time.Second
	}

	globalNames := make([]string, 0, len(globals))
	for name := range globals {
		globalNames = append(globalNames, name)
	}

	var (
		first        bool
		changed      []string
		lastTime     time.Time
		needsCompile bool
		base         string
	)

	r.mu.Lock()
	if r.templateCache == nil {
		r.templateCache = make(map[string]*templateCacheEntry)
	}
	entry := r.templateCache[filePath]
	first = entry == nil
	base = r.workDir
	if base == "" {
		base = filepath.Dir(filePath)
	}
	if entry != nil {
		lastTime = entry.compiledAt
		changedFiles, newest := changedPaths(entry.files, base)
		if changedDirs, dnewest := changedDirPaths(entry.dirs, base); len(changedDirs) > 0 {
			changedFiles = append(changedFiles, changedDirs...)
			if dnewest.After(newest) {
				newest = dnewest
			}
		}
		if len(changedFiles) > 0 {
			changed = changedFiles

			// The delay exists to avoid compiling a file still being written,
			// so it is measured from the file's own mtime. Anchoring it on the
			// moment a request first noticed instead made the wait start at the
			// page load: an edit saved long ago still served the old build, and
			// no amount of waiting helped — only a second request did.
			switch {
			case !newest.IsZero() && time.Since(newest) >= delay:
				needsCompile = true
			case entry.changedAt.IsZero():
				entry.changedAt = time.Now()
			}
		}
		if !needsCompile && !entry.changedAt.IsZero() && time.Since(entry.changedAt) >= delay {
			needsCompile = true
		}
	}
	r.mu.Unlock()

	if first || needsCompile {
		newEntry, cerr := r.compile(filePath, src, globalNames)
		if cerr == nil {
			newEntry.compiledAt = time.Now()
			r.mu.Lock()
			r.templateCache[filePath] = newEntry
			entry = newEntry
			r.mu.Unlock()
		}
		mainRel := filePath
		if base != "" {
			if rel, err := filepath.Rel(base, filePath); err == nil {
				// Report portable forward-slash paths to the callback regardless
				// of the host OS (filepath.Rel yields `\` on Windows).
				mainRel = filepath.ToSlash(rel)
			}
		}
		for _, fn := range r.onRenderFuncs {
			fn(first, mainRel, changed, lastTime, cerr)
		}
		if cerr != nil {
			return cerr
		}
	}

	if r.TranspilePath != nil {
		_ = gad.TranspileGadx(filePath, src, r.TranspilePath(filePath))
	}

	st := gad.NewSymbolTable(entry.builtins.NameSet)
	if _, err := st.DefineGlobals(globalNames); err != nil {
		return err
	}
	// The template builds and returns a render tree; the receiver of every
	// gadx.Tag/component method is trusted, so skip the `this` type check by
	// default. RunOptsFunc may override this (or any other option).
	runOpts := &gad.RunOpts{
		StdOut:  out,
		Globals: gad.Dict(globals),
		Flags:   gad.RunFlagSkipReceiverTypeCheck,
	}
	if r.RunOptsFunc != nil {
		r.RunOptsFunc(runOpts)
	}
	e := gad.NewEval(r.cachedStatic, st, gad.CompileOptions{}, runOpts)
	defer WithAttrHandler(e.VM, r.AttrHandler)()
	// Reuse the compiled template's interface-satisfaction cache across renders;
	// it is reset only when the template recompiles (a fresh entry, above), so
	// `obj :: Interface` checks in the template are validated once per type.
	if entry.ifaceCache != nil {
		e.VM.SetInterfaceSatCache(entry.ifaceCache)
	}
	ret, err := e.Run(context.Background(), entry.bc)
	if err != nil {
		return fmt.Errorf("render %s: %w", filePath, err)
	}
	// The compiled template builds a render tree and returns its root element;
	// walk it to write the HTML output.
	if el, ok := ret.(Element); ok {
		if _, err = el.WriteTo(e.VM, out); err != nil {
			return fmt.Errorf("render %s: %w", filePath, err)
		}
	}
	return nil
}

func (r *Render) compile(filePath string, src []byte, globalNames []string) (*templateCacheEntry, error) {
	r.compileMu.Lock()
	defer r.compileMu.Unlock()

	r.builtinsOnce.Do(func() {
		builtinsFn := r.BuiltinsFunc
		if builtinsFn == nil {
			builtinsFn = func() *gad.Builtins { return gad.NewBuiltins() }
		}
		r.cachedBuiltins = AppendBuiltins(builtinsFn())
		r.cachedStatic = r.cachedBuiltins.Build()
	})

	tr := newTrackingReader()

	// The root is the template tree: a plain import name means the same file
	// from every template, and no template reaches outside the tree with "..".
	root := r.workDir
	if root == "" {
		root = filepath.Dir(filePath)
	}

	// The working directory is the entry file's own, which is what "./" and
	// "../" are relative to. For an entry sitting in a subdirectory the two
	// differ, and using the root for both made "./sibling.gadx" look for the
	// sibling beside the root instead of beside the file.
	workDir := filepath.Dir(filePath)
	if workDir == "" {
		workDir = root
	}

	mm := gad.NewModuleMap().SetExtImporter(&importers.FileImporter{
		WorkDir: workDir,
		Root:    root,
		// A name without an extension resolves in the entry's dialect order
		// (.gadx first from a .gadx entry).
		SourceKind:    gad.SourceKindForExt(filePath),
		FileReader:    tr.Read,
		TranspilePath: r.TranspilePath,
		Inspect:       r.Inspect,
		Fallback:      r.Fallback,
		// a glob depends on its directory's listing
		OnGlob: tr.dir,
	})

	if r.ModuleMapFunc != nil {
		mm = r.ModuleMapFunc(mm)
		// A module the application adds from a file of its own (a SourceFile)
		// or from several (SourceFiles) is tracked as the templates' files
		// are, once a template imports it: a change to it recompiles.
		// (in a copy: the map given may outlive this compilation)
		var tracked *gad.ModuleMap
		for name, im := range mm.Importers() {
			var files func() []string
			switch sf := im.(type) {
			case SourceFiles:
				files = sf.SourceFiles
			case SourceFile:
				files = func() []string { return []string{sf.SourceFile()} }
			default:
				continue
			}
			if tracked == nil {
				tracked = mm.Copy()
			}
			tracked.Add(name, &trackedImport{Importable: im, files: files, tr: tr})
		}
		if tracked != nil {
			mm = tracked
		}
	}

	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		ModuleFile:  filePath,
		ModuleMap:   mm,
		EmbededdMap: gad.NewEmbedMap().SetExtImporter(&importers.EmbeddedFileImporter{WorkDirs: []string{root}}),
	}}
	// A .gadx ModuleFile compiles through gad's native Gadx front-end; a plain
	// .gad entry compiles as ordinary Gad. The dialect is chosen by extension.

	// The rendered file is inspected here; what it imports and includes, by
	// the importer as it reads them.
	if r.Inspect != nil {
		file, err := gad.ParseSource(filePath, src, gad.SourceKindForExt(filePath))
		if err != nil {
			return nil, fmt.Errorf("compile %s: %w", filePath, err)
		}
		if err = r.Inspect(filePath, file); err != nil {
			return nil, fmt.Errorf("inspect %s: %w", filePath, err)
		}
	}

	st := gad.NewSymbolTable(r.cachedBuiltins.NameSet)
	if _, err := st.DefineGlobals(globalNames); err != nil {
		return nil, err
	}

	var (
		bc  *gad.Bytecode
		err error
	)

	res, err := gad.Compile(st, src, opts)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %+v", filePath, err)
	}
	bc = res.Bytecode

	files := make(map[string]time.Time)

	// Track imported files — and the main template file. One read from the
	// Fallback is tracked at no time: never changed, until a file of
	// workDir takes its place.
	tr.files[filePath] = struct{}{}
	for p := range tr.files {
		if fi, err := os.Stat(p); err == nil {
			files[p] = fi.ModTime()
		} else if _, ok := importers.FallbackPath(r.Fallback, root, p); ok {
			files[p] = time.Time{}
		}
	}

	dirs := make(map[string]dirState, len(tr.dirs))
	for d, depth := range tr.dirs {
		sig, _ := dirSignature(d, depth)
		dirs[d] = dirState{depth: depth, sig: sig}
	}

	return &templateCacheEntry{
		bc:         bc,
		builtins:   r.cachedBuiltins,
		ifaceCache: gad.NewInterfaceSatCache(),
		files:      files,
		dirs:       dirs,
	}, nil
}

// changedPaths lists the tracked files that no longer match the recorded
// modification times, and reports the newest of their current mtimes. That time
// is what the recompile delay is measured against: it says when the editing
// stopped, which a request arriving later should not have to wait out again.
func changedPaths(files map[string]time.Time, base string) (out []string, newest time.Time) {
	for p, mod := range files {
		fi, err := os.Stat(p)
		if mod.IsZero() && err != nil {
			// the Fallback's, still not overridden
			continue
		}
		if err != nil || !fi.ModTime().Equal(mod) {
			rel, err := filepath.Rel(base, p)
			if err != nil {
				rel = p
			}
			out = append(out, filepath.ToSlash(rel))
			if err == nil && fi != nil && fi.ModTime().After(newest) {
				newest = fi.ModTime()
			}
		}
	}
	return out, newest
}

// dirState is a directory a template depends on by its listing: how deep
// (gad.GlobDepth) and its signature when the template compiled.
type dirState struct {
	depth int
	sig   string
}

// dirSignature is the listing of dir down to depth (-1: all of it) — each
// entry's path, size and modification time —, and the newest of those times
// (dir's own among them). A missing dir is the empty signature: one created
// later differs; a file there is its own size and time.
func dirSignature(dir string, depth int) (string, time.Time) {
	fi, err := os.Stat(dir)
	if err != nil {
		return "", time.Time{}
	}
	if !fi.IsDir() {
		return fmt.Sprintf("\x00file\x00%d\x00%d", fi.Size(), fi.ModTime().UnixNano()), fi.ModTime()
	}
	var (
		sb     strings.Builder
		newest = fi.ModTime()
	)
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil || p == dir {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() && depth >= 0 && strings.Count(filepath.ToSlash(rel), "/")+1 >= depth {
			return filepath.SkipDir
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		fmt.Fprintf(&sb, "%s\x00%d\x00%d\n", filepath.ToSlash(rel), info.Size(), info.ModTime().UnixNano())
		return nil
	})
	return sb.String(), newest
}

// changedDirPaths lists the directories whose listing is no longer the one
// recorded, and the newest time in them (changedPaths' delay anchor).
func changedDirPaths(dirs map[string]dirState, base string) (out []string, newest time.Time) {
	for d, st := range dirs {
		sig, n := dirSignature(d, st.depth)
		if sig == st.sig {
			continue
		}
		rel, err := filepath.Rel(base, d)
		if err != nil {
			rel = d
		}
		out = append(out, filepath.ToSlash(rel)+"/")
		if n.After(newest) {
			newest = n
		}
	}
	return out, newest
}
