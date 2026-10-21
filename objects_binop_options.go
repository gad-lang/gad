package gad

// BinOpOptionNested is the option of the item by item operators that descends
// into nested arrays (`a.+(b; nested)`).
const BinOpOptionNested = "nested"

// BinOpOptions are the options of an operator called with them — the named
// arguments of `a.op(b; …)` —, read and set as PrinterStateOptions are: the
// known ones by their getters (their defaults when absent), any other by its
// key. A handler (ObjectWith{Op}BinOperatorOptions) reads what it takes.
type BinOpOptions map[string]Object

// BinOpOptionsOf are the options of the named arguments kva (the last of a
// repeated key wins); nil when there are none.
func BinOpOptionsOf(kva KeyValueArray) BinOpOptions {
	if len(kva) == 0 {
		return nil
	}
	o := make(BinOpOptions, len(kva))
	for _, kv := range kva {
		o[kv.K.ToString()] = kv.V
	}
	return o
}

func (o BinOpOptions) Dict() Dict {
	return Dict(o)
}

// Nested reports whether the operator descends into nested arrays: the
// option's truth (`nested`, `nested=yes`); false when absent.
func (o BinOpOptions) Nested() bool {
	v, ok := o[BinOpOptionNested]
	return ok && !v.IsFalsy()
}

func (o BinOpOptions) SetNested(v bool) {
	o[BinOpOptionNested] = Flag(v)
}

// WithNested is o with nested set (a new map when o is nil).
func (o BinOpOptions) WithNested(v bool) BinOpOptions {
	if o == nil {
		o = BinOpOptions{}
	}
	o.SetNested(v)
	return o
}

// Unknown is the first key of o that is not one of known (sorted keys: the
// error names the same one every time); "" when there is none.
func (o BinOpOptions) Unknown(known ...string) string {
	var first string
	for k := range o {
		found := false
		for _, n := range known {
			if k == n {
				found = true
				break
			}
		}
		if !found && (first == "" || k < first) {
			first = k
		}
	}
	return first
}
