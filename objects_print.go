package gad

import (
	"context"
	"io"
	"reflect"

	"github.com/gad-lang/gad/repr"
)

const PrintLoop = repr.QuotePrefix + "↶" + repr.QuoteSufix

func IsPrimitive(obj Object) bool {
	val := reflect.ValueOf(obj)
try:
	// skip primitive values
	switch val.Type().Kind() {
	case reflect.Interface:
		val = val.Elem()
		goto try
	case reflect.Map, reflect.Ptr, reflect.Slice, reflect.Array, reflect.Func, reflect.Chan:
		return false
	default:
		return true
	}
}

type PrinterStateOption func(s *PrinterState)

func PrinterStateWithRaw(v bool) PrinterStateOption {
	return func(s *PrinterState) {
		if v {
			s.builtinType = BuiltinRawStr
		} else {
			s.builtinType = BuiltinStr
		}
	}
}

func PrinterStateWithContext(ctx context.Context) PrinterStateOption {
	return func(s *PrinterState) {
		s.context = ctx
	}
}

func PrinterStateWithOptions(options Dict) PrinterStateOption {
	return func(s *PrinterState) {
		s.options = options
	}
}

func PrinterStateWithMaxDepth(depth int) PrinterStateOption {
	return func(s *PrinterState) {
		s.maxDepth = depth
	}
}

func PrinterStateWithIndent(indent Object) PrinterStateOption {
	return func(s *PrinterState) {
		if indent != nil {
			if !indent.IsFalsy() {
				if indent == Yes {
					s.indent = []byte{'\t'}
				} else {
					s.indent = []byte(indent.ToString())
				}
			}
		}
	}
}

type printerStateEntry struct {
	object Object
}

type PrinterStateStack struct {
	prev  *PrinterStateStack
	value Object
}

func (e *PrinterStateStack) Prev() *PrinterStateStack {
	return e.prev
}

func (e *PrinterStateStack) Value() Object {
	return e.value
}

func (e *PrinterStateStack) PrevValue() Object {
	if e.prev == nil {
		return nil
	}
	return e.prev.value
}

// PrinterState represents the printer state passed to custom formatters.
// It provides access to the [io.Writer] interface plus information about
// the flags and options for the operand's format specifier.
type PrinterState struct {
	VM            *VM
	writer        io.Writer
	context       context.Context
	builtinType   BuiltinType
	options       Dict
	depth         int
	maxDepth      int
	indent        []byte
	currentIndent []byte
	bytesWriten   int64
	visited       map[any]bool
	stack         PrinterStateStack
}

func NewPrinterState(VM *VM, writer io.Writer, option ...PrinterStateOption) *PrinterState {
	s := &PrinterState{
		VM:          VM,
		writer:      writer,
		context:     VM.Context,
		options:     Dict{},
		visited:     make(map[any]bool),
		builtinType: BuiltinStr,
	}

	for _, opt := range option {
		opt(s)
	}
	return s
}

func (s *PrinterState) Stack() *PrinterStateStack {
	return &s.stack
}

func (s *PrinterState) Indented() bool {
	return len(s.indent) > 0
}

func (s *PrinterState) Indent() []byte {
	return s.indent
}

func (s *PrinterState) PrintIndent() {
	_, _ = s.Write(s.currentIndent)
}

func (s *PrinterState) PrintLine() {
	_, _ = s.Write([]byte{'\n'})
}

func (s *PrinterState) PrintLineIndent() {
	_, _ = s.Write(append([]byte{'\n'}, s.currentIndent...))
}

func (s *PrinterState) Visited(obj Object) bool {
	if !IsPrimitive(obj) {
		entry := printerStateEntry{obj}
		key := reflect.ValueOf(entry.object).UnsafePointer()
		return s.visited[key]
	}
	return false
}

func (s *PrinterState) DoVisit(obj Object, f func() error) (err error) {
	prev := s.stack
	s.stack = PrinterStateStack{
		prev:  &prev,
		value: obj,
	}

	defer func() {
		s.stack = *s.stack.prev
	}()

	if !IsPrimitive(obj) {
		entry := printerStateEntry{obj}
		key := reflect.ValueOf(entry.object).UnsafePointer()
		if s.visited[key] {
			_, err = s.Write([]byte(PrintLoop))
			return
		}
		s.visited[key] = true
		defer delete(s.visited, key)
	}
	return f()
}

func (s *PrinterState) IsFalsy() bool {
	return false
}

func (s *PrinterState) Type() ObjectType {
	return TPrinterState
}

func (s *PrinterState) Options() Dict {
	return s.options
}

func (s *PrinterState) ToString() string {
	return Dict{
		"builtinType":   Str(s.builtinType.String()),
		"depth":         Int(s.depth),
		"maxDepth":      Int(s.maxDepth),
		"options":       s.options,
		"bytesWriten":   Int(s.bytesWriten),
		"indent":        Bytes(s.indent),
		"currentIndent": Bytes(s.currentIndent),
	}.ToString()
}

func (s *PrinterState) Equal(right Object) bool {
	switch right := right.(type) {
	case *PrinterState:
		return s == right
	default:
		return false
	}
}

func (s *PrinterState) BytesWritten() int64 {
	return s.bytesWriten
}

func (s *PrinterState) Context() context.Context {
	return s.context
}

// WithContext override context
func (s *PrinterState) WithContext(ctx context.Context) *PrinterState {
	s.context = ctx
	return s
}

// WithValue add context value by key
func (s *PrinterState) WithValue(key, value any) *PrinterState {
	s.context = context.WithValue(s.context, key, value)
	return s
}

// WithValueBackup add context value by key and return restore func
func (s *PrinterState) WithValueBackup(key, value any) (restore func()) {
	old := s.context
	s.context = context.WithValue(s.context, key, value)
	return func() {
		s.context = old
	}
}

func (s *PrinterState) SkipDepth() bool {
	return s.maxDepth > 0 && s.depth == s.maxDepth
}

func (s *PrinterState) DoEnter(f func() error) error {
	if s.SkipDepth() {
		return nil
	}

	defer s.Enter()()

	return f()
}

func (s *PrinterState) Enter() (leave func()) {
	s.depth++

	if len(s.indent) > 0 {
		s.currentIndent = append(s.currentIndent, s.indent...)
	}

	return func() {
		s.depth--
		if s.Indented() {
			s.currentIndent = s.currentIndent[:len(s.currentIndent)-len(s.indent)]
		}
	}
}

// Value get context value by key or nil
func (s *PrinterState) Value(key any) (value any) {
	return s.context.Value(key)
}

// Write is the function to call to emit formatted output to be printed.
func (s *PrinterState) Write(b []byte) (n int, err error) {
	n, err = s.writer.Write(b)
	s.bytesWriten += int64(n)
	return
}

func (s *PrinterState) Option(key string) Object {
	return s.options[key]
}

func (s *PrinterState) OptionOk(key string) (value Object, ok bool) {
	value = s.options[key]
	ok = value != nil
	return
}

func (s *PrinterState) OptionDefault(key string, defaul Object) (value Object) {
	if value = s.options[key]; value == nil {
		value = defaul
	}
	return
}

func (s *PrinterState) Print(o Object) error {
	return s.DoVisit(o, func() (err error) {
		switch t := o.(type) {
		case Printer:
			err = t.Print(s)
		default:
			var str Object
			if str, err = Val(s.VM.Builtins.Call(s.builtinType, Call{
				VM:      s.VM,
				Context: s.context,
				Args:    Args{Array{o}},
			})); err != nil {
				return
			}
			_, err = s.Write([]byte(str.ToString()))
		}
		return
	})
}

func (s *PrinterState) PrintMany(sep []byte, o ...Object) (err error) {
	size := len(o)
	for i := 0; i < size; i++ {
		if i > 0 && len(sep) > 0 {
			_, _ = s.Write(sep)
		}
		if err = s.Print(o[i]); err != nil {
			return
		}
	}
	return
}

func (s *PrinterState) PrintFromArgs(sep []byte, args Args) (err error) {
	size := args.Length()
	for i := 0; i < size; i++ {
		if i > 0 && len(sep) > 0 {
			_, _ = s.Write(sep)
		}
		if err = s.Print(args.Get(i)); err != nil {
			return
		}
	}
	return
}

func (s *PrinterState) IndexGet(_ *VM, index Object) (value Object, err error) {
	key := index.ToString()
	switch key {
	case "builtinType":
		return Str(s.builtinType.String()), nil
	case "depth":
		return Int(s.depth), nil
	case "maxDepth":
		return Int(s.maxDepth), nil
	case "options":
		return s.options, nil
	case "bytesWriten":
		return Int(s.bytesWriten), nil
	case "indent":
		return Bytes(s.indent), nil
	case "currentIndent":
		return Bytes(s.currentIndent), nil
	default:
		return Nil, ErrInvalidIndex.NewError(key)
	}
}

func PrinterStateFromCall(c *Call) (state *PrinterState) {
	arg := c.Args.Get(0)

	switch t := arg.(type) {
	case *PrinterState:
		c.Args.Shift()
		return t
	case Writer:
		c.Args.Shift()
		return MustVal(NewPrinterStateFunc(Call{
			VM:        c.VM,
			Args:      Args{{t}},
			NamedArgs: c.NamedArgs,
			Context:   c.Context,
		})).(*PrinterState)
	}

	return MustVal(NewPrinterStateFunc(Call{
		VM:        c.VM,
		Args:      Args{{c.VM.StdOut}},
		NamedArgs: c.NamedArgs,
	})).(*PrinterState)
}

func (s PrinterState) Copy() *PrinterState {
	return &s
}

type Printer interface {
	Print(state *PrinterState) error
}

const (
	PrintStateOptionIndent    = "indent"
	PrintStateOptionMaxDepth  = "maxDepth"
	PrintStateOptionRaw       = "raw"
	PrintStateOptionZeros     = "zeros"
	PrintStateOptionAnonymous = "anonymous"
	PrintStateOptionSortKeys  = "sortKeys"
	PrintStateOptionIndexes   = "indexes"
)

type PrintStateOptionSortType uint8

const (
	PrintStateOptionSortTypeAscending PrintStateOptionSortType = iota + 1
	PrintStateOptionSortTypeDescending
)

func PrintStateOptionsGetZeros(s *PrinterState) bool {
	return !s.options.Get(PrintStateOptionZeros).IsFalsy()
}

func PrintStateOptionsGetAnonymous(s *PrinterState) bool {
	return !s.options.Get(PrintStateOptionAnonymous).IsFalsy()
}

func PrintStateOptionsGetSortKeysType(s *PrinterState) PrintStateOptionSortType {
	i, _ := s.options.Get(PrintStateOptionSortKeys).(Int)
	return PrintStateOptionSortType(i)
}

func PrintStateOptionsGetIndexes(s *PrinterState) bool {
	return !s.options.Get(PrintStateOptionIndexes).IsFalsy()
}
