package gad

// EnvType is the object type of *Env (gad.Env): the VM-scoped environment
// variable table reachable from Gad code via the `env` keyword.
var EnvType = registerGadNamespaceType(BuiltinEnvType, "Env", (*Env)(nil))

// Env is a VM-scoped environment variable table: a string-to-string map with an
// optional parent. It backs the `env` keyword. Env supports index get/set/delete
// and iteration over its own entries, exposes the `.parent` and `.fork`
// pseudo-keys, and implements the `with` resource protocol (Enter/Exit): entering
// forks the VM's current env so mutations are confined to the block and restored
// on exit.
type Env struct {
	Data   map[string]string
	Parent *Env
}

var (
	_ Object       = (*Env)(nil)
	_ IndexGetter  = (*Env)(nil)
	_ IndexSetter  = (*Env)(nil)
	_ IndexDeleter = (*Env)(nil)
	_ LengthGetter = (*Env)(nil)
	_ Copier       = (*Env)(nil)
	_ ObjectEnter  = (*Env)(nil)
	_ ObjectExit   = (*Env)(nil)
)

// NewEnv returns an empty Env with the given parent (nil for a root env).
func NewEnv(parent *Env) *Env {
	return &Env{Data: map[string]string{}, Parent: parent}
}

// NewEnvFromMap returns a root Env seeded with the given entries (copied).
func NewEnvFromMap(data map[string]string) *Env {
	e := &Env{Data: make(map[string]string, len(data))}
	for k, v := range data {
		e.Data[k] = v
	}
	return e
}

// Fork returns a new child env with a copy of this env's entries and this env as
// its parent. Mutations to the fork do not affect the parent.
func (e *Env) Fork() *Env {
	data := make(map[string]string, len(e.Data))
	for k, v := range e.Data {
		data[k] = v
	}
	return &Env{Data: data, Parent: e}
}

func (e *Env) Type() ObjectType { return EnvType }

func (e *Env) ToString() string { return "env" }

func (e *Env) IsFalsy() bool { return len(e.Data) == 0 }

// Equal reports pointer identity (each Env instance is distinct).
func (e *Env) Equal(right Object) bool {
	r, ok := right.(*Env)
	return ok && r == e
}

// Length is the number of entries in this env (excluding the parent chain).
func (e *Env) Length() int { return len(e.Data) }

// Copy implements Copier: a shallow copy with the same parent (unlike Fork,
// which sets this env as the copy's parent).
func (e *Env) Copy() Object {
	data := make(map[string]string, len(e.Data))
	for k, v := range e.Data {
		data[k] = v
	}
	return &Env{Data: data, Parent: e.Parent}
}

// Get returns the value of key and whether it is present in this env.
func (e *Env) Get(key string) (string, bool) {
	v, ok := e.Data[key]
	return v, ok
}

// IndexGet resolves the `.parent` and `.fork` pseudo-keys, otherwise the value of
// an environment variable as a str (Nil when absent).
func (e *Env) IndexGet(_ *VM, index Object) (Object, error) {
	key := index.ToString()
	switch key {
	case "parent":
		if e.Parent == nil {
			return Nil, nil
		}
		return e.Parent, nil
	case "fork":
		return e.Fork(), nil
	}
	if v, ok := e.Data[key]; ok {
		return Str(v), nil
	}
	return Nil, nil
}

// IndexSet sets an environment variable; the value is stored as its str form.
func (e *Env) IndexSet(_ *VM, index, value Object) error {
	e.Data[index.ToString()] = value.ToString()
	return nil
}

// IndexDelete removes an environment variable.
func (e *Env) IndexDelete(_ *VM, key Object) error {
	delete(e.Data, key.ToString())
	return nil
}

// Iterate yields the env entries as key(str) -> value(str), reusing the Dict
// iterator over a snapshot Dict.
func (e *Env) Iterate(vm *VM, na *NamedArgs) Iterator {
	d := make(Dict, len(e.Data))
	for k, v := range e.Data {
		d[k] = Str(v)
	}
	return d.Iterate(vm, na)
}

// Enter forks the VM's current env and makes the fork current, so mutations
// inside a `with env { … }` block are confined to the fork.
func (e *Env) Enter(vm *VM) error {
	vm.SetEnv(e.Fork())
	return nil
}

// Exit restores the VM's env to this one (the state captured before Enter forked
// it), discarding the fork's mutations.
func (e *Env) Exit(vm *VM, _ error) (Object, error) {
	vm.SetEnv(e)
	return e, nil
}
