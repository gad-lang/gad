package gad

// SourceStackEntry is one frame of the VM source-name stack: the module or the
// included file whose code is currently running. `@file` reports the innermost
// entry's name, and `@files` exposes the whole stack (read-only). A `*Module`
// satisfies it (a module frame), and so does SourceName (an included file).
type SourceStackEntry interface {
	Object
	// Name is the source name reported by `@file` for this entry.
	Name() string
}

// SourceName is a source-stack entry naming an included file (the value
// `OpPushSource` pushes for `include`). It behaves as a string in the
// language (its Type() is `str`), so `@file`/`@files` values compare and print
// like ordinary strings.
type SourceName string

var _ SourceStackEntry = SourceName("")
var _ SourceStackEntry = (*Module)(nil)

// Name implements SourceStackEntry.
func (o SourceName) Name() string { return string(o) }

func (o SourceName) Type() ObjectType { return TStr }

func (o SourceName) ToString() string { return string(o) }

func (o SourceName) IsFalsy() bool { return len(o) == 0 }

func (o SourceName) Equal(right Object) bool {
	switch v := right.(type) {
	case SourceName:
		return o == v
	case Str:
		return string(o) == string(v)
	case RawStr:
		return string(o) == string(v)
	}
	return false
}
