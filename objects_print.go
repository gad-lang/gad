package gad

import (
	"context"
	"io"

	"github.com/gad-lang/gad/repr"
)

const PrintLoop = repr.QuotePrefix + "↶" + repr.QuoteSufix

type PrinterStateOption func(s *PrinterState)

func PrinterStateWithContext(ctx context.Context) PrinterStateOption {
	return func(s *PrinterState) {
		if ctx != nil {
			s.context = ctx
		}
	}
}

func PrinterStateWithOptions(options PrinterStateOptions) PrinterStateOption {
	return func(s *PrinterState) {
		s.options = options

		if md, ok := options.MaxDepth(); ok {
			s.maxDepth = md
		}

		if indent, ok := options.Indent(); ok {
			if len(indent) > 0 {
				s.indent = []byte(indent)
			}
		}

		if v, ok := s.options.Repr(); v && ok {
			s.IsRepr = true
		}

		if raw, _ := s.options.Raw(); raw {
			s.builtinType = BuiltinRawStr
		} else {
			s.builtinType = BuiltinStr
		}
	}
}

func PrinterStateWithOptionsFromNamedArgs(na *NamedArgs) PrinterStateOption {
	return func(s *PrinterState) {
		s.ParseOptions(na)
	}
}

type PrinterStateStack struct {
	prev     *PrinterStateStack
	value    Object
	fallback bool
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
	IsRepr        bool
	writer        io.Writer
	context       context.Context
	builtinType   BuiltinType
	options       PrinterStateOptions
	depth         int64
	maxDepth      int64
	indent        []byte
	currentIndent []byte

	bytesWriten int64
	visited     map[any]bool
	stack       PrinterStateStack
	onVisite    func(o Object) (done func())
}

func NewPrinterState(VM *VM, writer io.Writer, option ...PrinterStateOption) *PrinterState {
	var ctx context.Context
	if VM != nil {
		ctx = VM.Context
	}

	if ctx == nil {
		ctx = context.Background()
	}

	s := &PrinterState{
		VM:          VM,
		writer:      writer,
		context:     ctx,
		options:     make(PrinterStateOptions, 0),
		visited:     make(map[any]bool),
		builtinType: BuiltinStr,
	}

	for _, opt := range option {
		opt(s)
	}

	return s
}

func (s *PrinterState) GoWriter() io.Writer {
	return s
}

func (s *PrinterState) QuoteNextStr(level int64) {
	s.OnVisite(func(old func(o Object) func()) func(o Object) func() {
		depth := s.depth + level
		return func(o Object) func() {
			isq := s.options.IsQuoteStr()
			done := old(o)

			if isq || s.depth > depth {
				return done
			}

			depth++
			restore := s.options.Backup(PrintStateOptionQuoteStr)
			s.options.WithQuoteStr()

			return func() {
				depth--
				restore()
				done()
			}
		}
	})
}

func (s *PrinterState) OnVisite(f func(old func(o Object) (done func())) func(o Object) (done func())) {
	if s.onVisite == nil {
		s.onVisite = f(func(o Object) (done func()) {
			return func() {
			}
		})
	} else {
		s.onVisite = f(s.onVisite)
	}
}

func (s *PrinterState) Update() *PrinterState {
	PrinterStateWithOptions(s.options)(s)
	return s
}

func (s *PrinterState) ParseOptions(na *NamedArgs) *PrinterState {
	var (
		maxDepth, _ = na.GetValue(PrintStateOptionMaxDepth).(Int)
		indent      = na.GetValue(PrintStateOptionIndent)
	)

	options := PrinterStateOptions(na.unreadDict())
	options.SetRaw(!na.GetValue(PrintStateOptionRaw).IsFalsy())
	options.SetMaxDepth(int64(maxDepth))

	if !indent.IsFalsy() {
		options.SetIndent(indent)
	}

	for K, v := range options {
		s.options[K] = v
	}

	return s.Update()
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
	if addr := AddressOf(obj); addr != nil {
		return s.visited[addr]
	}
	return false
}

func (s *PrinterState) DoVisit(obj Object, f func() error) (err error) {
	addr := AddressOf(obj)
	if addr != nil {
		if s.visited[addr] {
			if s.stack.value == obj {
				_, err = s.Write([]byte(obj.ToString()))
			} else {
				_, err = s.Write([]byte(PrintLoop))
			}
			return
		}
		s.visited[addr] = true
		defer delete(s.visited, addr)
	}
	if s.onVisite != nil {
		defer s.onVisite(obj)()
	}
	prev := s.stack
	s.stack = PrinterStateStack{
		prev:  &prev,
		value: obj,
	}

	defer func() {
		s.stack = *s.stack.prev
	}()
	return f()
}

func (s *PrinterState) IsFalsy() bool {
	return false
}

func (s *PrinterState) Type() ObjectType {
	return TPrinterState
}

func (s *PrinterState) Options() PrinterStateOptions {
	return s.options
}

func (s *PrinterState) Do(f func(s *PrinterState)) *PrinterState {
	f(s)
	return s
}

func (s *PrinterState) ToString() string {
	return Dict{
		"builtinType":   Str(s.builtinType.String()),
		"depth":         Int(s.depth),
		"maxDepth":      Int(s.maxDepth),
		"options":       s.options.Dict(),
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

func (s *PrinterState) SkipNexDepth() bool {
	return s.maxDepth > 0 && (s.depth+1) == s.maxDepth
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

func (s *PrinterState) WriteByte(c byte) (err error) {
	_, err = s.Write([]byte{c})
	return
}

func (s *PrinterState) WriteString(b string) (err error) {
	_, err = s.Write([]byte(b))
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
		return s.options.Dict(), nil
	case "bytesWriten":
		return Int(s.bytesWriten), nil
	case "indent":
		return Bytes(s.indent), nil
	case "isIndent":
		return Bool(s.Indented()), nil
	case "currentIndent":
		return Bytes(s.currentIndent), nil
	case "isRepr":
		return Bool(s.IsRepr), nil
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
