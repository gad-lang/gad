package gad

import (
	"sort"
	"strings"
)

// TContract is the type of a contract (a `contract NAME { … }` declaration).
var TContract = NewType("contract", TBase)

// Contract is a `contract NAME { … }`: what a type is — its target — and what
// its metadata may say — the keys of `meta { … }`, an interface's body:
//
//	contract TSelectField {
//	    *TField                      // parent contracts: target and meta merged
//	    target type<str|int|enum>    // the types the target may be (none: any)
//	    meta {
//	        options []str            // a key without `?` is required
//	        layout? str = "table"    // a default: a key not given takes it
//	    }
//	}
//
// It is checked against a SUBJECT: a type (a class, an enum, a typed array, a
// declared union `type city <str>`) — the target is the type, the meta its
// `@meta` —, a field of a class or an interface — the target its type, the
// meta the type's `@meta` and then the field's —, or an instance of a
// contract. `subject :: C` checks it (the subject unchanged); `subject :> C`
// and `C(subject; k=v…)` build the instance — the target, the meta checked
// with its defaults —; `C.check(subject)` says every problem.
//
// Inside the contract `target` is a type and a value — the target given:
// `options []target = target.@members` (MetaFactory is `(target) => interface
// { … }`, the meta of one target).
type Contract struct {
	CName  string
	Module *ModuleSpec
	// Meta is the `[k=v, …]` written before the declaration (`C.@meta`).
	Meta    KeyValueArray
	Parents []*Contract
	// TargetIface is `interface { target T }`, what the target must satisfy;
	// nil: any.
	TargetIface *Interface
	// MetaFactory is `(target) => interface { … }`, the keys of the meta of a
	// target; nil: no keys of its own.
	MetaFactory CallerObject
	// Required is how many keys of its own meta are required (no `?`, no
	// default): what tells overloads apart first (a select's `options`).
	Required int
}

// required is how many keys of the meta of c and its parents are required.
func (c *Contract) required() int {
	n := 0
	for _, x := range c.chain() {
		n += x.Required
	}
	return n
}

var (
	_ ObjectType    = (*Contract)(nil)
	_ vmCanAssigner = (*Contract)(nil)
	_ IndexGetter   = (*Contract)(nil)
)

func (c *Contract) GadObjectType()   {}
func (c *Contract) Type() ObjectType { return TContract }
func (c *Contract) Name() string     { return c.CName }
func (c *Contract) FullName() string {
	if c.Module != nil && c.Module.Name != "" {
		return c.Module.Name + "." + c.CName
	}
	return c.CName
}
func (c *Contract) String() string          { return c.ToString() }
func (c *Contract) ToString() string        { return ReprQuoteTyped("contract", c.FullName()) }
func (c *Contract) IsFalsy() bool           { return false }
func (c *Contract) Equal(right Object) bool { return right == Object(c) }

// chain is c's contracts, its parents first (each once, depth first) and c
// last: the order their targets are checked and their metas merged.
func (c *Contract) chain() []*Contract {
	var out []*Contract
	seen := map[*Contract]bool{}
	var walk func(x *Contract)
	walk = func(x *Contract) {
		if seen[x] {
			return
		}
		seen[x] = true
		for _, p := range x.Parents {
			walk(p)
		}
		out = append(out, x)
	}
	walk(c)
	return out
}

// Is reports whether c is other or derives from it.
func (c *Contract) Is(other *Contract) bool {
	for _, x := range c.chain() {
		if x == other {
			return true
		}
	}
	return false
}

// depth is how deep c is in its parents (a root contract is 0).
func (c *Contract) depth() int {
	d := 0
	for _, p := range c.Parents {
		if pd := p.depth() + 1; pd > d {
			d = pd
		}
	}
	return d
}

// MetaIface is the interface of the meta of target: the keys of the contracts
// of c, a child's key over its parent's of the same name.
func (c *Contract) MetaIface(vm *VM, target Object) (*Interface, error) {
	out := &Interface{IName: c.CName + ".meta", Module: c.Module}
	at := map[string]int{}
	for _, x := range c.chain() {
		if x.MetaFactory == nil {
			continue
		}
		v, err := vm.Call(x.MetaFactory, Args{{target}}, nil)
		if err != nil {
			return nil, err
		}
		iface, ok := v.(*Interface)
		if !ok {
			return nil, ErrType.NewErrorf("contract %s: its meta is a %s, not an interface", x.CName, v.Type().Name())
		}
		if iface, err = iface.Flatten(vm); err != nil {
			return nil, err
		}
		for _, f := range iface.Fields {
			if i, ok := at[f.Name]; ok {
				out.Fields[i] = f // the child's, over its parent's
				continue
			}
			at[f.Name] = len(out.Fields)
			out.Fields = append(out.Fields, f)
		}
	}
	return out, nil
}

// subject is what a contract is checked against: its target and the layers of
// its meta, the weakest first.
func contractSubject(vm *VM, obj Object) (target Object, layers KeyValueArray) {
	metaOf := func(o Object) KeyValueArray {
		if ig, ok := o.(IndexGetter); ok {
			if v, err := ig.IndexGet(vm, Str("@meta")); err == nil {
				kva, _ := v.(KeyValueArray)
				return kva
			}
		}
		return nil
	}
	typeOf := func(types []TypeAssigner) Object {
		switch len(types) {
		case 0:
			return TAny
		case 1:
			return types[0]
		}
		objs := make([]Object, len(types))
		for i, t := range types {
			objs[i] = t
		}
		return NewTypeUnion(objs...)
	}
	// a declared union of one type (`[options=[…]] type city <str>`) is that
	// type, its meta the union's
	unwrap := func(o Object) Object {
		if u, ok := o.(*TypeUnion); ok && len(u.Types) == 1 {
			return u.Types[0]
		}
		return o
	}
	switch t := obj.(type) {
	case *ContractInstance:
		return t.Target, t.RawMeta
	case *ClassField:
		target = typeOf(t.Types)
		return unwrap(target), append(append(KeyValueArray{}, metaOf(target)...), t.Meta...)
	case *InterfaceField:
		target = typeOf(t.resolveAssigners(vm))
		return unwrap(target), append(append(KeyValueArray{}, metaOf(target)...), t.Meta...)
	}
	return unwrap(obj), metaOf(obj)
}

// ContractProblems are what a check of a contract found: errors (the subject
// does not fit) and warnings (a key the contract does not declare — another
// plugin's, maybe).
type ContractProblems struct {
	Errors   []string
	Warnings []string
}

// build checks subject against c — every problem, not only the first — and
// builds its instance (nil when there are errors). extra is the strongest
// layer of the meta (a call's named arguments).
func (c *Contract) build(vm *VM, subject Object, extra KeyValueArray) (*ContractInstance, *ContractProblems, error) {
	p := &ContractProblems{}
	target, layers := contractSubject(vm, subject)

	// the target, against the target of every contract of c
	for _, x := range c.chain() {
		if x.TargetIface == nil || len(x.TargetIface.Fields) == 0 {
			continue
		}
		f := x.TargetIface.Fields[0]
		if _, reason := coerceValue(vm, f.resolveAssigners(vm), target); reason != nil {
			p.Errors = append(p.Errors, "target: "+ReprQuote(contractObjName(target))+" is not assignable to "+
				ReprQuote(strings.Join(f.typeNames(), "|"))+" ("+x.CName+")")
		}
	}
	if len(p.Errors) > 0 {
		// the meta is the meta of a target that fits
		return nil, p, nil
	}

	mi, err := c.MetaIface(vm, target)
	if err != nil {
		return nil, p, err
	}
	raw := append(append(KeyValueArray{}, layers...), extra...).Normalized()
	given := raw.ToDict() // the resets out: not given
	meta := make(Dict, len(mi.Fields))
	declared := map[string]bool{}
	for _, f := range mi.Fields {
		declared[f.Name] = true
		v, has := given[f.Name]
		if !has && f.Default != nil {
			if v, err = f.defaultValue(vm); err != nil {
				return nil, p, err
			}
			has = true
		}
		if !has {
			if !f.Nullable {
				p.Errors = append(p.Errors, "meta "+Str(f.Name).Quoted()+" is required")
			}
			meta[f.Name] = No // not given
			continue
		}
		if v == Nil {
			if !f.Nullable {
				p.Errors = append(p.Errors, "meta "+Str(f.Name).Quoted()+" is required (nil given)")
			}
			meta[f.Name] = Nil
			continue
		}
		cv, reason := coerceValue(vm, f.resolveAssigners(vm), v)
		if reason != nil {
			p.Errors = append(p.Errors, "meta "+Str(f.Name).Quoted()+": "+reason.Error())
			continue
		}
		meta[f.Name] = cv
	}
	var unknown []string
	for k := range given {
		if !declared[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		p.Warnings = append(p.Warnings, "meta "+Str(k).Quoted()+": not declared by "+ReprQuote(c.CName))
	}
	if len(p.Errors) > 0 {
		return nil, p, nil
	}
	return &ContractInstance{Contract: c, Target: target, Meta: meta, RawMeta: raw, given: given}, p, nil
}

// Instance is `subject :> c`: the instance of c of subject, or why not —
// every error.
func (c *Contract) Instance(vm *VM, subject Object, extra KeyValueArray) (*ContractInstance, error) {
	if inst, ok := subject.(*ContractInstance); ok && inst.Contract == c && len(extra) == 0 {
		return inst, nil
	}
	inst, p, err := c.build(vm, subject, extra)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: %s",
			ReprQuote(contractObjName(subject)), ReprQuote(c.CName), strings.Join(p.Errors, "; "))
	}
	return inst, nil
}

// CanAssign: an instance of c (or of a contract that derives from it). The
// VM's check (CanAssignVM) also takes a subject that fits.
func (c *Contract) CanAssign(obj Object) (bool, error) {
	inst, ok := obj.(*ContractInstance)
	return ok && inst.Contract.Is(c), nil
}

// CanAssignVM reports whether obj fits c: an instance of it, or a subject
// whose target and meta c takes (`subject :: c` keeps the subject).
func (c *Contract) CanAssignVM(vm *VM, obj Object) (bool, error) {
	if ok, _ := c.CanAssign(obj); ok {
		return true, nil
	}
	inst, _, err := c.build(vm, obj, nil)
	return inst != nil, err
}

func (c *Contract) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if to != TypeAssigner(c) {
		return nil, ErrIncompatibleAssign
	}
	if ok, err := c.CanAssignVM(vm, obj); err != nil || !ok {
		if err == nil {
			_, err = c.Instance(vm, obj, nil)
		}
		return nil, err
	}
	return obj, nil
}

// Call is `C(subject; k=v…)`: the instance, the named arguments the strongest
// layer of its meta.
func (c *Contract) Call(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	return c.Instance(call.VM, call.Args.GetOnly(0), call.NamedArgs.Join())
}

// IndexGet reflects the contract:
//
//	C.@name, C.@meta, C.@module
//	C.@parents  — the parent contracts
//	C.@target   — the types the target may be (an array; empty: any)
//	C.@depth    — how deep in its parents (a root is 0)
func (c *Contract) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "@name":
		return Str(c.CName), nil
	case "@meta":
		return metaObject(c.Meta), nil
	case "@module":
		if c.Module == nil || vm == nil {
			return Nil, nil
		}
		return vm.ModuleFromIndex(c.Module.Index), nil
	case "@parents":
		out := make(Array, len(c.Parents))
		for i, p := range c.Parents {
			out[i] = p
		}
		return out, nil
	case "@target":
		if c.TargetIface == nil || len(c.TargetIface.Fields) == 0 {
			return Array{}, nil
		}
		return c.TargetIface.Fields[0].IndexGet(vm, Str("types"))
	case "@depth":
		return Int(c.depth()), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// CallName: `C.check(subject)` — {errors: […], warnings: […]}, every problem —
// and `C.metaIface(target)` — the interface of the meta of a target.
func (c *Contract) CallName(name string, call Call) (Object, error) {
	switch name {
	case "check":
		if err := call.Args.CheckLen(1); err != nil {
			return nil, err
		}
		_, p, err := c.build(call.VM, call.Args.GetOnly(0), call.NamedArgs.Join())
		if err != nil {
			return nil, err
		}
		strs := func(s []string) Array {
			out := make(Array, len(s))
			for i, v := range s {
				out[i] = Str(v)
			}
			return out
		}
		return Dict{"errors": strs(p.Errors), "warnings": strs(p.Warnings)}, nil
	case "metaIface":
		if err := call.Args.CheckLen(1); err != nil {
			return nil, err
		}
		return c.MetaIface(call.VM, call.Args.GetOnly(0))
	}
	return nil, ErrInvalidIndex.NewError(name)
}

// ContractInstance is a subject as a contract takes it: its target, its meta
// checked — every key of the contract, the defaults applied, one not given
// `no` — and the meta as given (RawMeta: every key, other plugins' too).
type ContractInstance struct {
	Contract *Contract
	Target   Object
	Meta     Dict
	RawMeta  KeyValueArray
	given    Dict
}

var (
	_ Object      = (*ContractInstance)(nil)
	_ IndexGetter = (*ContractInstance)(nil)
)

// Type is the contract: `typeof(inst) == C`.
func (o *ContractInstance) Type() ObjectType { return o.Contract }
func (o *ContractInstance) IsFalsy() bool    { return false }
func (o *ContractInstance) ToString() string {
	return ReprQuoteTyped(o.Contract.CName, contractObjName(o.Target)+" "+o.Meta.ToString())
}
func (o *ContractInstance) Equal(right Object) bool { return right == Object(o) }

// IndexGet: `inst.target`, `inst.meta` (a dict), `inst.rawMeta` (the
// key-value array), `inst.contract`.
func (o *ContractInstance) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "target":
		return o.Target, nil
	case "meta":
		return o.Meta, nil
	case "rawMeta":
		return o.RawMeta, nil
	case "contract":
		return o.Contract, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// CallName: `inst.given(key)` — whether the key was given (with or without a
// value): a default does not count, a reset (`no`) is not given.
func (o *ContractInstance) CallName(name string, call Call) (Object, error) {
	if name != "given" {
		return nil, ErrInvalidIndex.NewError(name)
	}
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	_, ok := o.given[call.Args.GetOnly(0).ToString()]
	return Bool(ok), nil
}

// NewContractFunc is the `Contract(name, parents, targetIface, metaFactory;
// meta=…)` builtin a `contract NAME { … }` declaration lowers to.
func NewContractFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(4); err != nil {
		return nil, err
	}
	name, ok := c.Args.GetOnly(0).(Str)
	if !ok {
		return nil, NewArgumentTypeError("1st (name)", "str", c.Args.GetOnly(0).Type().Name())
	}
	ct := &Contract{CName: string(name), Module: c.VM.CurrentModuleSpec()}
	parents, _ := c.Args.GetOnly(1).(Array)
	for i, p := range parents {
		pc, ok := p.(*Contract)
		if !ok {
			return nil, ErrType.NewErrorf("contract %s: parent #%d is a %s, not a contract", name, i, p.Type().Name())
		}
		ct.Parents = append(ct.Parents, pc)
	}
	if iface, ok := c.Args.GetOnly(2).(*Interface); ok {
		ct.TargetIface = iface
	}
	if f, ok := c.Args.GetOnly(3).(CallerObject); ok && c.Args.GetOnly(3) != Nil {
		ct.MetaFactory = f
	}
	if m := c.NamedArgs.GetValueOrNil("meta"); m != nil {
		ct.Meta, _ = m.(KeyValueArray)
	}
	if n, ok := c.NamedArgs.GetValueOrNil("required").(Int); ok {
		ct.Required = int(n)
	}
	return ct, nil
}

// contractObjName is the name of a subject or a target in a message: a type's
// own name (TypeAssignerName), else the name of the value's type.
func contractObjName(o Object) string {
	switch t := o.(type) {
	case *ClassField:
		return "field " + t.Name
	case *InterfaceField:
		return "field " + t.Name
	case TypeAssigner:
		return TypeAssignerName(t)
	}
	return o.Type().Name()
}

// assignMismatch says why obj does not fit c (mismatchSayer): every error.
func (c *Contract) assignMismatch(vm *VM, obj Object) string {
	if vm == nil {
		return ""
	}
	_, p, err := c.build(vm, obj, nil)
	if err != nil {
		return err.Error()
	}
	return strings.Join(p.Errors, "; ")
}
