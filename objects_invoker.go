package gad

// BoundInvoker is a repeatable, pre-resolved call produced by the gad.invoker
// builtin. It binds a callee — the overload of a function already resolved for a
// fixed parameter-type signature — to a shared, caller-owned argument array.
//
// Calling it invokes the callee with the array's CURRENT values, reusing the same
// backing array (no per-call allocation) and skipping parameter-type validation
// (the overload was resolved once, up front). Mutate the array between calls to
// feed new values — it is the fast path for a hot loop that calls the same
// function shape many times, paying the type dispatch and the argument array only
// once:
//
//	var (
//	    args = [int]                 // the initial elements are the parameter types
//	    inv  = gad.invoker(str, args)
//	)
//	for i in 0 ..< 1000 { args[0] = i; buf.write(inv()) }
//
// Because it skips validation, the caller is trusted to keep the array's values
// compatible with the resolved signature; feeding a different type is undefined.
type BoundInvoker struct {
	// callee is the resolved overload, or the function itself when it has none.
	callee CallerObject
	// args is the shared, caller-owned positional-args array, reused every call.
	args Array
	// argsWrap caches Args{args} so a plain invocation allocates nothing; the
	// array's contents (which the caller mutates in place) are read afresh each
	// call.
	argsWrap Args
	// namedArgs are the **nargs captured at construction, merged ahead of any
	// named args passed to a given call.
	namedArgs KeyValueArray
}

// NewBoundInvoker binds callee to a shared args array and the captured named args.
func NewBoundInvoker(callee CallerObject, args Array, namedArgs KeyValueArray) *BoundInvoker {
	return &BoundInvoker{callee: callee, args: args, argsWrap: Args{args}, namedArgs: namedArgs}
}

func (*BoundInvoker) Type() ObjectType { return TBoundInvoker }

func (b *BoundInvoker) Name() string { return "invoker" }

func (b *BoundInvoker) ToString() string {
	return b.Type().ToString() + "{" + b.callee.ToString() + "}"
}

func (*BoundInvoker) IsFalsy() bool { return false }

func (*BoundInvoker) Equal(Object) bool { return false }

// Call invokes the bound callee with the shared array's current values, merging
// the construction-time named args ahead of any passed to this call. The args are
// not validated (SkipValidation): the callee is the overload already resolved for
// the declared signature.
func (b *BoundInvoker) Call(c Call) (Object, error) {
	args := b.argsWrap
	if len(c.Args) > 0 {
		args = append(Args{b.args}, c.Args...)
	}
	namedArgs := c.NamedArgs
	if len(b.namedArgs) > 0 {
		namedArgs = NamedArgs{sources: KeyValueArrays{b.namedArgs}}
		if len(c.NamedArgs.sources) > 0 {
			namedArgs.Add(c.NamedArgs.UnreadPairs())
		}
	}
	return DoCall(b.callee, Call{VM: c.VM, Args: args, NamedArgs: namedArgs, SkipValidation: true})
}
