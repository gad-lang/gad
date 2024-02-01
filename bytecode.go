// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
)

// Bytecode holds the compiled functions and constants.
type Bytecode struct {
	FileSet    *parser.SourceFileSet
	Main       *CompiledFunction
	Constants  []Object
	NumModules int
}

// Fprint writes constants and instructions to given Writer in a human readable form.
func (bc *Bytecode) Fprint(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Bytecode")
	_, _ = fmt.Fprintf(w, "Modules:%d\n", bc.NumModules)
	bc.putConstants(w)
	bc.Main.Fprint(w)
}

func (bc *Bytecode) String() string {
	var buf bytes.Buffer
	bc.Fprint(&buf)
	return buf.String()
}

func (bc *Bytecode) putConstants(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Constants:\n")
	for i := range bc.Constants {
		var cf *CompiledFunction

		switch t := bc.Constants[i].(type) {
		case *CompiledFunction:
			cf = t
		case *CallerObjectWithMethods:
			if !t.HasCallerMethods() {
				cf, _ = t.CallerObject.(*CompiledFunction)
			}
		}

		if cf != nil {
			_, _ = fmt.Fprintf(w, "%4d: CompiledFunction\n", i)

			var b bytes.Buffer
			cf.Fprint(&b)

			_, _ = fmt.Fprint(w, "\t")

			str := b.String()
			c := strings.Count(str, "\n")
			_, _ = fmt.Fprint(w, strings.Replace(str, "\n", "\n\t", c-1))
			continue
		}
		_, _ = fmt.Fprintf(w, "%4d: %#v|%s\n",
			i, bc.Constants[i], bc.Constants[i].Type().Name())
	}
}

type ParamType []*Symbol

func (t ParamType) String() string {
	l := len(t)
	switch l {
	case 0:
		return ""
	case 1:
		return t[0].Name
	default:
		var s = make([]string, len(t))
		for i, symbol := range t {
			s[i] = symbol.Name
		}
		return "[" + strings.Join(s, ", ") + "]"
	}
}

func (t ParamType) Accept(vm *VM, ot ObjectType) (ok bool, err error) {
	if len(t) == 0 {
		ok = true
		return
	}

	var st Object

	for _, symbol := range t {
		if st, err = vm.GetSymbolValue(symbol); err != nil {
			return
		} else {
			if cwm, _ := st.(*CallerObjectWithMethods); cwm != nil {
				st = cwm.CallerObject
			}
			if ot == st {
				ok = true
				return
			} else if stot, _ := st.(ObjectType); stot != nil {
				if ok = IsTypeAssignableTo(stot, ot); ok {
					return
				}
			}
		}
	}
	return
}

type Params struct {
	Names []string
	Type  []ParamType
	Typed bool
	Len   int
	Var   bool
}

func (p *Params) Min() int {
	if p.Var {
		return p.Len - 1
	}
	return p.Len
}

func (p *Params) String() string {
	var s = make([]string, p.Len)
	if len(p.Type) > 0 {
		for i, t := range p.Type {
			if ts := t.String(); ts == "" {
				s[i] = p.Names[i]
			} else {
				s[i] = p.Names[i] + " " + ts
			}
		}
	} else {
		copy(s, p.Names)
	}
	if p.Var {
		s[p.Len-1] = "*" + s[p.Len-1]
	}
	return strings.Join(s, ", ")
}

type NamedParam struct {
	Name string
	// Value is a script of default value
	Value string
}

type NamedParams struct {
	Params   []*NamedParam
	len      int
	variadic bool
	byName   map[string]int
}

func NewNamedParams(params ...*NamedParam) (np *NamedParams) {
	np = &NamedParams{Params: params}
	np.len = len(params)
	np.Params = params

	if np.len > 0 {
		np.byName = make(map[string]int, np.len)
		for i, p := range params {
			np.byName[p.Name] = i
		}
		np.variadic = params[len(params)-1].Value == ""
	}
	return
}

func (n *NamedParams) Names() (names []string) {
	names = make([]string, n.len)
	for i, param := range n.Params {
		names[i] = param.Name
	}
	return
}

func (n *NamedParams) Len() int {
	return n.len
}

func (n *NamedParams) Variadic() bool {
	return n.variadic
}

func (n *NamedParams) ByName() map[string]int {
	return n.byName
}

func (n *NamedParams) String() string {
	var s = make([]string, n.len)
	for i, param := range n.Params {
		if param.Value != "" {
			s[i] = param.Name + "=" + param.Value
		} else {
			s[i] = "**" + param.Name
		}
	}
	return strings.Join(s, ", ")
}

// CompiledFunction holds the constants and instructions to pass VM.
type CompiledFunction struct {
	Name string

	AllowMethods bool
	// number of local variabls including parameters NumLocals>=NumParams
	NumLocals    int
	Instructions []byte
	Free         []*ObjectPtr
	// SourceMap holds the index of instruction and token's position.
	SourceMap map[int]int

	Params Params

	NamedParams NamedParams

	// NamedParamsMap is a map of NamedParams with index
	// this value allow to perform named args validation.
	NamedParamsMap map[string]int
}

var (
	_ Object       = (*CompiledFunction)(nil)
	_ CallerObject = (*CompiledFunction)(nil)
)

func (*CompiledFunction) Type() ObjectType {
	return TCompiledFunction
}

func (o *CompiledFunction) ToString() string {
	var (
		s      []string
		params []string
	)
	s = append(s, " "+o.Name)
	if o.Params.Len > 0 {
		params = append(params, o.Params.String())
	}
	if o.NamedParams.len > 0 {
		params = append(params, o.NamedParams.String())
	}
	s = append(s, "("+strings.Join(params, ", ")+")")
	return ReprQuote("compiledFunction" + strings.Join(s, ""))
}

func (o *CompiledFunction) Repr(vm *VM) (_ string, err error) {
	var (
		s      []string
		params []string
	)
	s = append(s, " "+o.Name)
	if o.Params.Len > 0 {
		params = append(params, o.Params.String())
	}
	if o.NamedParams.len > 0 {
		params = append(params, o.NamedParams.String())
	}
	s = append(s, "("+strings.Join(params, ", ")+")")
	return ReprQuote("compiledFunction" + strings.Join(s, "")), nil
}

func (o *CompiledFunction) Format(f fmt.State, verb rune) {
	if verb == 'v' {
		f.Write([]byte(o.ToString()))
	}
}

// Copy implements the Copier interface.
func (o *CompiledFunction) Copy() Object {
	var insts []byte
	if o.Instructions != nil {
		insts = make([]byte, len(o.Instructions))
		copy(insts, o.Instructions)
	}

	var free []*ObjectPtr
	if o.Free != nil {
		// DO NOT Copy() elements; these are variable pointers
		free = make([]*ObjectPtr, len(o.Free))
		copy(free, o.Free)
	}

	var sourceMap map[int]int
	if o.SourceMap != nil {
		sourceMap = make(map[int]int, len(o.SourceMap))
		for k, v := range o.SourceMap {
			sourceMap[k] = v
		}
	}

	return &CompiledFunction{
		NumLocals:    o.NumLocals,
		Instructions: insts,
		Free:         free,
		SourceMap:    sourceMap,
		Params:       o.Params,
		NamedParams:  o.NamedParams,
	}
}

// IsFalsy implements Object interface.
func (*CompiledFunction) IsFalsy() bool { return false }

// Equal implements Object interface.
func (o *CompiledFunction) Equal(right Object) bool {
	v, ok := right.(*CompiledFunction)
	return ok && o == v
}

// SourcePos returns the source position of the instruction at ip.
func (o *CompiledFunction) SourcePos(ip int) source.Pos {
begin:
	if ip >= 0 {
		if p, ok := o.SourceMap[ip]; ok {
			return source.Pos(p)
		}
		ip--
		goto begin
	}
	return source.NoPos
}

// Fprint writes constants and instructions to given Writer in a human readable form.
func (o *CompiledFunction) Fprint(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Locals: %d\n", o.NumLocals)
	_, _ = fmt.Fprintf(w, "Params: %s\n", o.Params.String())
	_, _ = fmt.Fprintf(w, "NamedParams: %s\n", o.NamedParams.String())
	_, _ = fmt.Fprintf(w, "Instructions:\n")

	i := 0
	var operands []int

	for i < len(o.Instructions) {

		op := o.Instructions[i]
		numOperands := OpcodeOperands[op]
		operands, offset := ReadOperands(numOperands, o.Instructions[i+1:], operands)
		_, _ = fmt.Fprintf(w, "%04d %-12s", i, OpcodeNames[op])

		if len(operands) > 0 {
			for _, r := range operands {
				_, _ = fmt.Fprint(w, "    ", strconv.Itoa(r))
			}
		}

		_, _ = fmt.Fprintln(w)
		i += offset + 1
	}

	if o.Free != nil {
		_, _ = fmt.Fprintf(w, "Free:%v\n", o.Free)
	}
	_, _ = fmt.Fprintf(w, "SourceMap:%v\n", o.SourceMap)
}

func (o *CompiledFunction) identical(other *CompiledFunction) bool {
	if o.Name != other.Name ||
		o.NumLocals != other.NumLocals ||
		o.Params.String() != other.Params.String() ||
		o.NamedParams.String() != other.NamedParams.String() ||
		len(o.Instructions) != len(other.Instructions) ||
		len(o.Free) != len(other.Free) ||
		string(o.Instructions) != string(other.Instructions) {
		return false
	}
	for i := range o.Free {
		if o.Free[i].Equal(other.Free[i]) {
			return false
		}
	}
	return true
}

func (o *CompiledFunction) equalSourceMap(other *CompiledFunction) bool {
	if len(o.SourceMap) != len(other.SourceMap) {
		return false
	}
	for k, v := range o.SourceMap {
		vv, ok := other.SourceMap[k]
		if !ok || vv != v {
			return false
		}
	}
	return true
}

func (o *CompiledFunction) hash32() uint32 {
	hash := hashData32(2166136261, []byte{byte(o.NumLocals)})
	if o.Params.Len > 0 {
		hash = hashData32(hash, []byte(o.Params.String()))
	}
	if o.NamedParams.len > 0 {
		hash = hashData32(hash, []byte(o.NamedParams.String()))
	}
	hash = hashData32(hash, o.Instructions)
	return hash
}

func (o *CompiledFunction) Call(c Call) (Object, error) {
	return NewInvoker(c.VM, o).ValidArgs(c.SafeArgs).Invoke(c.Args, &c.NamedArgs)
}

func (o *CompiledFunction) SetNamedParams(params ...*NamedParam) {
	o.NamedParams = *NewNamedParams(params...)
}

func (o *CompiledFunction) ValidateParamTypes(vm *VM, args Args) (err error) {
	if o.Params.Var {
		if min := o.Params.Min(); args.Length() < min {
			return ErrWrongNumArguments.NewError(fmt.Sprintf("expected >= %d but got %d", min, args.Length()))
		}
	} else if args.Length() != o.Params.Len {
		return ErrWrongNumArguments.NewError(fmt.Sprintf("expected %d but got %d", o.Params.Len, args.Length()))
	}

	if o.Params.Typed {
		var (
			l       = o.Params.Len
			argType ObjectType
			t       ParamType
			accept  bool
		)
		if o.Params.Var {
			l--
		}

		for i := 0; i < l; i++ {
			argType = args.GetOnly(i).Type()
			t = o.Params.Type[i]
			if t != nil {
				if accept, err = t.Accept(vm, argType); err != nil {
					return
				} else if !accept {
					return NewArgumentTypeError(strconv.Itoa(i+1)+"st ("+o.Params.Names[i]+")", t.String(), argType.Name())
				}
			}
		}

		if o.Params.Var {
			t = o.Params.Type[o.Params.Len-1]
			args.WalkSkip(o.Params.Len-1, func(i int, arg Object) any {
				if accept, err = t.Accept(vm, arg.Type()); err == nil && !accept {
					err = NewArgumentTypeError(strconv.Itoa(i+1)+"st ("+o.Params.Names[i]+")", t.String(), arg.Type().Name())
				}
				return err
			})
		}
	}
	return
}

func (o *CompiledFunction) CanValidateParamTypes() bool {
	return o.Params.Typed
}

func (o *CompiledFunction) ParamTypes(vm *VM) (types MultipleObjectTypes, err error) {
	if o.Params.Typed {
		types = make(MultipleObjectTypes, len(o.Params.Type))
		for i, t := range o.Params.Type {
			ts := make([]ObjectType, len(t))
			for i2, symbol := range t {
				if typ, err := vm.GetSymbolValue(symbol); err != nil {
					return nil, err
				} else {
					if cwm, _ := typ.(*CallerObjectWithMethods); cwm != nil {
						typ = cwm.CallerObject
					}
					ts[i2] = typ.(ObjectType)
				}
			}
			types[i] = ts
		}
	}
	return
}

func hashData32(hash uint32, data []byte) uint32 {
	for _, c := range data {
		hash *= 16777619 // prime32
		hash ^= uint32(c)
	}
	return hash
}
