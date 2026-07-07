// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"context"
	"errors"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// Importable interface represents importable module instance.
type Importable interface {
	// Import should return either an Object or module source code ([]byte).
	Import(ctx context.Context, module *ModuleSpec) (data any, uri string, err error)
}

// ExtImporter wraps methods for a module which will be impored dynamically like
// a file.
type ExtImporter interface {
	Importable
	// Get returns Extimporter instance which will import a module.
	Get(moduleName string) ExtImporter
	// Name returns the full name of the module e.g. absoule path of a file.
	// Import names are generally relative, this overwrites module name and used
	// as unique key for compiler module cache.
	Name() (string, error)
	// Fork returns an ExtImporter instance which will be used to import the
	// modules. Fork will get the result of Name() if it is not empty, otherwise
	// module name will be same with the Get call.
	Fork(moduleName string) ExtImporter
}

type CompilableImporter interface {
	Importable
	CompileModule(compiler *Compiler, nd ast.Node, module *ModuleSpec, moduleMap *ModuleMap, src []byte) (bc *Bytecode, err error)
}

// ModuleMap represents a set of named modules. Use NewModuleMap to create a
// new module map.
type ModuleMap struct {
	m  map[string]Importable
	im ExtImporter
}

// NewModuleMap creates a new module map.
func NewModuleMap() *ModuleMap {
	return &ModuleMap{m: make(map[string]Importable)}
}

// SetExtImporter sets an ExtImporter to ModuleMap, which will be used to
// import modules dynamically.
func (m *ModuleMap) SetExtImporter(im ExtImporter) *ModuleMap {
	m.im = im
	return m
}

// Fork creates a new ModuleMap instance if ModuleMap has an ExtImporter to
// make ExtImporter preseve state.
func (m *ModuleMap) Fork(moduleName string) *ModuleMap {
	if m == nil {
		return nil
	}
	if m.im != nil {
		fork := m.im.Fork(moduleName)
		return &ModuleMap{m: m.m, im: fork}
	}
	return m
}

// Add adds an importable module.
func (m *ModuleMap) Add(name string, module Importable) *ModuleMap {
	m.m[name] = module
	return m
}

// AddBuiltinModule adds a builtin module.
func (m *ModuleMap) AddBuiltinModule(
	name string,
	attrs map[string]Object,
) *ModuleMap {
	m.m[name] = &BuiltinModule{Attrs: attrs}
	return m
}

// AddBuiltinModuleInit adds a builtin module.
func (m *ModuleMap) AddBuiltinModuleInit(
	name string,
	init ModuleInitFunc,
) *ModuleMap {
	m.m[name] = &BuiltinInitModule{Init: init}
	return m
}

// AddBuiltinCompilableModule adds a builtin compilable module.
func (m *ModuleMap) AddBuiltinCompilableModule(
	name string,
	compile BuiltinCompileModuleFunc,
) *ModuleMap {
	m.m[name] = &BuiltinCompilableModule{Compile: compile}
	return m
}

// AddSourceModule adds a source module.
func (m *ModuleMap) AddSourceModule(name string, src []byte) *ModuleMap {
	m.m[name] = &SourceModule{Src: src}
	return m
}

// Remove removes a named module.
func (m *ModuleMap) Remove(name string) {
	delete(m.m, name)
}

// Get returns an import module identified by name.
// It returns nil if the name is not found.
func (m *ModuleMap) Get(name string) Importable {
	if m == nil {
		return nil
	}

	v, ok := m.m[name]
	if ok || m.im == nil {
		return v
	}
	return m.im.Get(name)
}

func (m *ModuleMap) Importers() map[string]Importable {
	return m.m
}

// Copy creates a copy of the module map.
func (m *ModuleMap) Copy() *ModuleMap {
	c := &ModuleMap{m: make(map[string]Importable), im: m.im}

	for name, mod := range m.m {
		c.m[name] = mod
	}
	return c
}

// SourceModule is an importable module that's written in Gad.
type SourceModule struct {
	Src []byte
}

// Import returns a module source code.
func (m *SourceModule) Import(_ context.Context, module *ModuleSpec) (any, string, error) {
	return m.Src, module.Name, nil
}

// BuiltinModule is an importable module that's written in ToInterface.
type BuiltinModule struct {
	Attrs    Dict
	initFunc ModuleInitFunc
}

func (m *BuiltinModule) InitFunc() ModuleInitFunc {
	if m.initFunc != nil {
		return m.initFunc
	}
	return func(module *Module, c Call) (err error) {
		cp := make(Dict, len(m.Attrs))

		for k, v := range m.Attrs {
			switch t := v.(type) {
			case *Function:
				t = Copy(t)
				t.SetModule(module.Spec)
				v = t
			case *Type:
				t = Copy(t)
				t.Module = module.Spec
				v = t
			case ModuleSetter:
				t.SetModule(module.Spec)
			}
			cp[k] = v
		}

		m.initFunc = func(module *Module, c Call) (err error) {
			module.MergeData(cp)
			return
		}

		return m.initFunc(module, c)
	}
}

// Import returns an immutable map for the module.
func (m *BuiltinModule) Import(_ context.Context, module *ModuleSpec) (any, string, error) {
	if m.Attrs == nil {
		return nil, "", errors.New("module attributes not set")
	}

	return m.InitFunc(), "builtin:" + module.Name, nil
}

type ModuleInitFunc func(module *Module, c Call) (err error)

func (f ModuleInitFunc) MustGetData(module *Module) (data ModuleData) {
	var err error
	if err = f(module, Call{}); err != nil {
		panic(err)
	}
	return module.Data
}

func (f ModuleInitFunc) Caller(spec *ModuleSpec) func(module *Module) CallerObject {
	return func(module *Module) CallerObject {
		return &Function{
			FuncName: "#moduleInitFunc@" + spec.Name,
			Value: func(c Call) (_ Object, err error) {
				return Nil, f(module, c)
			},
		}
	}
}

func ModuleInitWithDataDict(d Dict) ModuleInitFunc {
	return func(module *Module, c Call) (err error) {
		module.MergeData(d)
		return nil
	}
}

// BuiltinInitModule is an importable module that's written in ToInterface.
type BuiltinInitModule struct {
	Init ModuleInitFunc
}

// Import returns an immutable map for the module.
func (m *BuiltinInitModule) Import(_ context.Context, module *ModuleSpec) (any, string, error) {
	if m.Init == nil {
		return nil, "", errors.New("module init func not set")
	}

	return m.Init, "builtinModuleInit:" + module.Name, nil
}

type BuiltinCompileModuleContext struct {
	Node     ast.Node
	Compiler *Compiler
	FileSet  *source.FileSet
	Spec     *ModuleSpec
}

func (c *BuiltinCompileModuleContext) SetFileData(data []byte) *source.File {
	file := c.FileSet.AppendFileData(c.Spec.URL, data)
	c.Compiler.file = file
	return file
}

func (c *BuiltinCompileModuleContext) Compile(smts node.Stmts) error {
	return c.Compiler.compileFileStmts(smts)
}

type BuiltinCompileModuleFunc func(ctx *BuiltinCompileModuleContext) (*Bytecode, error)

// BuiltinCompilableModule is an importable module that's written in ToInterface.
type BuiltinCompilableModule struct {
	Compile BuiltinCompileModuleFunc
}

// Import returns an immutable map for the module.
func (m *BuiltinCompilableModule) Import(_ context.Context, module *ModuleSpec) (any, string, error) {
	if m.Compile == nil {
		return nil, "", errors.New("module init func not set")
	}

	return m.Compile, "moduleCompiler:" + module.Name, nil
}
