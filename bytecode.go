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

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// Bytecode holds the compiled functions and constants.
type Bytecode struct {
	FileSet    *source.SourceFileSet
	Main       *CompiledFunction
	Constants  []Object
	NumModules int
	NumEmbeds  int
}

// Fprint writes constants and instructions to given Writer in a human readable form.
func (bc *Bytecode) Fprint(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Bytecode")
	_, _ = fmt.Fprintf(w, "Modules:%d\n", bc.NumModules)
	_, _ = fmt.Fprintf(w, "Embeds:%d\n", bc.NumEmbeds)
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

type ModuleInfo struct {
	Name string
	File string
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
	sourceFile     *source.File
	module         *ModuleInfo
}

var (
	_ Object       = (*CompiledFunction)(nil)
	_ CallerObject = (*CompiledFunction)(nil)
)

func (*CompiledFunction) Type() ObjectType {
	return TCompiledFunction
}

func (o CompiledFunction) ClearSourceFileInfo() *CompiledFunction {
	o.module = nil
	o.sourceFile = nil
	return &o
}

func (o *CompiledFunction) ToString() string {
	var (
		s      []string
		params []string
	)
	s = append(s, " "+o.Name)
	if !o.Params.Empty() {
		params = append(params, o.Params.String())
	}
	if o.NamedParams.len > 0 {
		params = append(params, o.NamedParams.String())
	}
	s = append(s, "("+strings.Join(params, ", ")+")")
	return ReprQuote("compiledFunction" + strings.Join(s, ""))
}

func (o *CompiledFunction) Repr(*VM) (_ string, err error) {
	var (
		s      []string
		params []string
	)
	s = append(s, " "+o.Name)
	if !o.Params.Empty() {
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

func (o *CompiledFunction) WithParams(names ...string) *CompiledFunction {
	o.Params = make(Params, len(names))
	for i := range o.Params {
		o.Params[i] = new(Param)
	}

	for i, name := range names {
		if name[0] == '*' {
			o.Params[i].Var = true
			name = name[1:]
		}
		if pos := strings.IndexByte(name, ' '); pos > 0 {
			t := name[pos+1:]
			o.Params[i].Name = name[:pos]
			if t[0] == '[' {
				t = strings.ReplaceAll(t[1:len(t)-1], " ", "")
			}
			tnames := strings.Split(t, ",")
			symbols := make(ParamType, len(tnames))
			for i2, tname := range tnames {
				tname = strings.TrimSpace(tname)
				symbols[i2] = &SymbolInfo{Name: tname}
			}
			o.Params[i].Type = symbols
		} else {
			o.Params[i].Name = name
		}
	}
	return o
}

// Fprint writes constants and instructions to given Writer in a human readable form.
func (o *CompiledFunction) Fprint(w io.Writer) {
	o.FprintLP("", w)
}

// FprintLP writes constants and instructions to given Writer in a human readable form with line prefix.
func (o *CompiledFunction) FprintLP(linePrefix string, w io.Writer) {
	_, _ = fmt.Fprintf(w, "%sLocals: %d\n", linePrefix, o.NumLocals)
	_, _ = fmt.Fprintf(w, "%sParams: %s\n", linePrefix, o.Params.String())
	_, _ = fmt.Fprintf(w, "%sNamedParams: %s\n", linePrefix, o.NamedParams.String())
	_, _ = fmt.Fprintf(w, "%sInstructions:\n", linePrefix)

	i := 0
	var operands []int

	for i < len(o.Instructions) {

		op := o.Instructions[i]
		numOperands := OpcodeOperands[op]
		operands, offset := ReadOperands(numOperands, o.Instructions[i+1:], operands)
		_, _ = fmt.Fprintf(w, "%s\t%04d %-12s", linePrefix, i, OpcodeNames[op])

		if len(operands) > 0 {
			for _, r := range operands {
				_, _ = fmt.Fprint(w, "    ", strconv.Itoa(r))
			}
			switch Opcode(op) {
			case OpBinaryOp:
				_, _ = fmt.Fprint(w, " (", token.Token(operands[0]).String(), ")")
			}
		}

		_, _ = fmt.Fprintln(w)
		i += offset + 1
	}

	if o.Free != nil {
		_, _ = fmt.Fprintf(w, "%sFree:%v\n", linePrefix, o.Free)
	}
	_, _ = fmt.Fprintf(w, "%sSourceMap:%v\n", linePrefix, o.SourceMap)
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
	if !o.Params.Empty() {
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
	if o.Params.Var() {
		if required := len(o.Params) - 1; args.Length() < required {
			return ErrWrongNumArguments.NewError(fmt.Sprintf("expected >= %d but got %d", required, args.Length()))
		}
	} else if args.Length() != len(o.Params) {
		return ErrWrongNumArguments.NewError(fmt.Sprintf("expected %d but got %d", len(o.Params), args.Length()))
	}

	if o.Params.Typed() {
		var (
			l       = len(o.Params)
			argType ObjectType
			t       ParamType
			accept  bool
			last    = o.Params[l-1]
		)
		if last.Var {
			l--
		}

		for i := 0; i < l; i++ {
			argType = args.GetOnly(i).Type()
			t = o.Params[i].Type
			if t != nil {
				if accept, err = t.Accept(vm, argType); err != nil {
					return
				} else if !accept {
					return NewArgumentTypeError(strconv.Itoa(i+1)+"st ("+o.Params[i].Name+")", t.String(), argType.Name())
				}
			}
		}

		if last.Var {
			t = last.Type
			args.WalkSkip(l, func(i int, arg Object) any {
				if accept, err = t.Accept(vm, arg.Type()); err == nil && !accept {
					err = NewArgumentTypeError(strconv.Itoa(i+1)+"st ("+o.Params[i].Name+")", t.String(), arg.Type().Name())
				}
				return err
			})
		}
	}
	return
}

func (o *CompiledFunction) CanValidateParamTypes() bool {
	return o.Params.Typed()
}

func (o *CompiledFunction) ParamTypes(vm *VM) (types MultipleObjectTypes, err error) {
	if o.Params.Typed() {
		types = make(MultipleObjectTypes, len(o.Params))
		for i, p := range o.Params {
			ts := make([]ObjectType, len(p.Type))
			for i2, symbol := range p.Type {
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
