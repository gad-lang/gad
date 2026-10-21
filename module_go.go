package gad

import (
	"slices"
	"sort"
	"strings"
)

// A module written in gad and shipped in Go (issue #17, `gad gomodule`): its
// gad part compiled once (a `.gadb`), linked into the program that imports it,
// and a Go init of its own, which runs after the gad part, in the VM that
// imports the module: it may import modules — declared ones — and get the gad
// part's exports, and what it returns is exported beside them.

// ModuleGoInit is the Go init of a module: the objects it returns are
// exported by the module, beside its gad part's.
type ModuleGoInit func(ctx *ModuleInitContext) (Dict, error)

// LinkedModule is what an Importable gives for a module compiled on its own
// (its Bytecode, linked) that has a Go init: the modules it imports by name
// (the gad part's are known already), and the init (nil: none).
type LinkedModule struct {
	Bytecode *Bytecode
	Imports  []string
	Init     ModuleGoInit
}

// ModuleInitContext is what the Go init of a module is given.
type ModuleInitContext struct {
	vm      *VM
	module  *Module
	replace Dict
}

// VM is the VM that imports the module.
func (c *ModuleInitContext) VM() *VM { return c.vm }

// Spec is the spec of the module.
func (c *ModuleInitContext) Spec() *ModuleSpec { return c.module.Spec }

// Self is the module: the exports of its gad part.
func (c *ModuleInitContext) Self() *Module { return c.module }

// Get is the export name of the module's gad part.
func (c *ModuleInitContext) Get(name string) (Object, error) {
	return c.module.Get(name)
}

// Import is the module name, as `import(name)` gives it: one the module
// imports — its gad part, or declared (LinkedModule.Imports).
func (c *ModuleInitContext) Import(name string) (*Module, error) {
	idx, ok := c.module.Spec.GoImports[name]
	if !ok {
		return nil, ErrNotImportable.NewErrorf("module %s: %q is not an import of it: declare it (`gad gomodule --import %s`)",
			c.module.Spec.Name, name, name)
	}
	return c.vm.loadModule(idx)
}

// Replace exports obj as name, in place of the gad part's export of the name.
func (c *ModuleInitContext) Replace(name string, obj Object) {
	if c.replace == nil {
		c.replace = Dict{}
	}
	c.replace[name] = obj
}

// Get is the member name of the module, an error when it has none.
func (m *Module) Get(name string) (Object, error) {
	if !slices.ContainsFunc(m.Keys(), func(k Object) bool { return k.ToString() == name }) {
		return nil, ErrInvalidIndex.NewErrorf("module %s has no %q", m.Spec.Name, name)
	}
	return m.Data.IndexGet(nil, Str(name))
}

// Call calls the member name of the module with args.
func (m *Module) Call(vm *VM, name string, args ...Object) (Object, error) {
	f, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	return vm.Call(f, Args{args}, nil)
}

// runGoInit runs the Go init of module: its objects exported beside the gad
// part's — a name of both an error, but for ctx.Replace.
func (vm *VM) runGoInit(module *Module) error {
	ctx := &ModuleInitContext{vm: vm, module: module}
	d, err := module.Spec.GoInit(ctx)
	if err != nil {
		return err
	}
	var dup []string
	for k := range d {
		if _, err := module.Get(k); err == nil {
			dup = append(dup, k)
		}
	}
	if len(dup) > 0 {
		sort.Strings(dup)
		return ErrType.NewErrorf("module %s: %s exported by its gad part too (ctx.Replace replaces an export)",
			module.Spec.Name, strings.Join(dup, ", "))
	}
	for k, v := range d {
		module.Set(k, v)
	}
	for k, v := range ctx.replace {
		module.Set(k, v)
	}
	return nil
}

// loadModule is the module of index idx, initialized when it is not yet —
// as OpLoadModule and OpInitModule do.
func (vm *VM) loadModule(idx int) (*Module, error) {
	if m := vm.modulesCache[idx]; m != nil {
		return m, nil
	}
	spec := vm.modules[idx]
	module := NewModule(spec)
	vm.modulesCache[idx] = module
	if spec.InitGoFunc != nil {
		if _, err := vm.Call(spec.InitGoFunc(module), Args{}, nil); err != nil {
			return nil, err
		}
	} else if spec.InitCompiledFunc != nil {
		if _, err := vm.callCompiledInline(spec.InitCompiledFunc, Args{}, NewNamedArgs(), false); err != nil {
			return nil, err
		}
	}
	if spec.GoInit != nil {
		if err := vm.runGoInit(module); err != nil {
			return nil, err
		}
	}
	return module, nil
}
