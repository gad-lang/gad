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
)

// Bytecode holds the compiled functions and constants.
type Bytecode struct {
	FileSet    *source.FileSet
	Main       *CompiledFunction
	Constants  Array
	NumModules int
	NumEmbeds  int
}

// Fprint writes constants and instructions to given Writer in a human readable form.
func (bc *Bytecode) Fprint(builtins *Builtins, w io.Writer) {
	_, _ = fmt.Fprintln(w, "Bytecode")
	_, _ = fmt.Fprintf(w, "Modules:%d\n", bc.NumModules)
	_, _ = fmt.Fprintf(w, "Embeds:%d\n", bc.NumEmbeds)
	bc.putConstants(builtins, w)
	bc.Main.Fprint(builtins, w, bc)
}

func (bc *Bytecode) String() string {
	var buf bytes.Buffer
	bc.Fprint(nil, &buf)
	return buf.String()
}

func (bc *Bytecode) putConstants(builtins *Builtins, w io.Writer) {
	repr := func(v Object) string {
		if o, err := ToRepr(nil, v, PrinterStateOptions{}.WithIndent()); err != nil {
			panic(err)
		} else {
			return o.ToString()
		}
	}
	_, _ = fmt.Fprintf(w, "Constants (%d):\n", len(bc.Constants))
	for i := range bc.Constants {
		c := bc.Constants[i]

		switch t := c.(type) {
		case *CompiledFunction:
			_, _ = fmt.Fprintf(w, "%04d: ", i)

			var b bytes.Buffer
			t.Fprint(builtins, &b, bc)
			str := b.String()
			c := strings.Count(str, "\n")
			_, _ = fmt.Fprint(w, strings.Replace(str, "\n", "\n\t", c-1))
			continue
		}

		_, _ = fmt.Fprintf(w, "%04d: %s\n",
			i, repr(bc.Constants[i]))
	}
}

type ModuleInfo struct {
	Name string
	File string
}

var (
	_ Object       = (*CompiledFunction)(nil)
	_ CallerObject = (*CompiledFunction)(nil)
	_ Printer      = (*CompiledFunction)(nil)
)

// CompiledFunction holds the constants and instructions to pass VM.
type CompiledFunction struct {
	FuncName string

	AllowMethods bool
	// number of local variabls including parameters NumLocals>=NumParams
	NumLocals    int
	Instructions []byte
	Free         []*ObjectPtr
	Return       *ObjectPtr
	// SourceMap holds the index of instruction and token's position.
	SourceMap map[int]int

	Params Params

	NamedParams NamedParams

	// NamedParamsMap is a map of NamedParams with index
	// this value allow to perform named args validation.
	NamedParamsMap map[string]int
	module         *Module
}

func (o *CompiledFunction) SetModule(module *Module) {
	o.module = module
}

func (o *CompiledFunction) GetModule() *Module {
	return o.module
}

func (o *CompiledFunction) Name() string {
	return o.FuncName
}

func (*CompiledFunction) Type() ObjectType {
	return TCompiledFunction
}

func (o *CompiledFunction) FullName() string {
	if o.FuncName == "" {
		return ""
	}
	return o.module.Info.Name + "." + o.FuncName
}

func (o *CompiledFunction) HeaderString() string {
	var buf strings.Builder
	buf.WriteString(o.FullName())
	buf.WriteString("(")
	if !o.Params.Empty() {
		buf.WriteString(o.Params.String())
	}

	if o.NamedParams.len > 0 {
		buf.WriteString("; ")
		buf.WriteString(o.NamedParams.String())
	}
	buf.WriteByte(')')
	return buf.String()
}

func (o *CompiledFunction) ToString() string {
	return ReprQuoteTyped("compiledFunction", o.HeaderString())
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
func (o *CompiledFunction) WithNamedParams(names ...string) *CompiledFunction {
	params := make([]*NamedParam, len(names))

	for i, name := range names {
		p := &NamedParam{
			TypesSymbols: make(ParamType, 0),
		}
		
		if strings.HasPrefix(name, "**") {
			params[i].Var = true
			name = name[2:]
		}

		if pos := strings.IndexByte(name, '='); pos > 0 {
			p.Value = strings.TrimSpace(name[pos+1:])
			name = strings.TrimSpace(name[:pos])
		}

		if pos := strings.IndexByte(name, ' '); pos > 0 {
			t := name[pos+1:]
			name = name[:pos]
			if t[0] == '[' {
				t = strings.ReplaceAll(t[1:len(t)-1], " ", "")
			}
			tnames := strings.Split(t, ",")
			symbols := make(ParamType, len(tnames))
			for i2, tname := range tnames {
				tname = strings.TrimSpace(tname)
				symbols[i2] = &SymbolInfo{Name: tname}
			}
			p.TypesSymbols = symbols
		}

		name = strings.TrimSpace(name)
		p.Name = name
		p.Symbol = &SymbolInfo{Name: name}
		params[i] = p
	}

	o.NamedParams = *NewNamedParams(params...)
	return o
}

func (o *CompiledFunction) WithParams(names ...string) *CompiledFunction {
	params := make([]*Param, len(names))
	var (
		si    = -1
		newSi = func() int {
			si++
			return si
		}
	)

	for i, name := range names {
		p := &Param{
			Index:        i,
			Symbol:       &SymbolInfo{Index: newSi()},
			TypesSymbols: make(ParamType, 0),
		}

		if name[0] == '*' {
			p.Var = true
			name = name[1:]
		}

		if pos := strings.IndexByte(name, ' '); pos > 0 {
			t := name[pos+1:]
			p.Name = name[:pos]
			if t[0] == '[' {
				t = strings.ReplaceAll(t[1:len(t)-1], " ", "")
			}
			tnames := strings.Split(t, ",")
			symbols := make(ParamType, len(tnames))
			for i2, tname := range tnames {
				tname = strings.TrimSpace(tname)
				symbols[i2] = &SymbolInfo{Name: tname, Index: newSi()}
			}
			p.TypesSymbols = symbols
		} else {
			p.Name = name
		}

		p.Symbol.Name = name
		params[i] = p
	}

	o.Params = *NewParams(params...)
	return o
}

// Fprint writes constants and instructions to given Writer in a human readable form.
func (o *CompiledFunction) Fprint(builtins *Builtins, w io.Writer, bc *Bytecode) {
	o.FprintLP(builtins, bc.Constants, "", w)
}

// FprintLP writes constants and instructions to given Writer in a human readable form with line prefix.
func (o *CompiledFunction) FprintLP(builtins *Builtins, constants Array, linePrefix string, w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s\n", o.ToString())
	_, _ = fmt.Fprintf(w, "%sLocals: %d\n", linePrefix, o.NumLocals)
	_, _ = fmt.Fprintf(w, "%sInstructions:\n", linePrefix)

	for _, line := range FormatInstructions(builtins, constants, o.Instructions, 0) {
		fmt.Fprintf(w, "%s\t%s\n", linePrefix, line)
	}

	if o.Free != nil {
		_, _ = fmt.Fprintf(w, "%sFree:%v\n", linePrefix, o.Free)
	}
	_, _ = fmt.Fprintf(w, "%sSourceMap:%v\n", linePrefix, o.SourceMap)
}

func (o *CompiledFunction) identical(other *CompiledFunction) bool {
	if o.FuncName != other.FuncName ||
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
		if required := o.Params.RequiredCount() - 1; args.Length() < required {
			return ErrWrongNumArguments.NewError(fmt.Sprintf("expected >= %d but got %d", required, args.Length()))
		}
	} else if args.Length() != o.Params.len {
		return ErrWrongNumArguments.NewError(fmt.Sprintf("expected %d but got %d", o.Params.len, args.Length()))
	}

	if o.Params.Typed() {
		var (
			l       = o.Params.len
			argType ObjectType
			t       ParamType
			accept  bool
			last    = o.Params.Items[l-1]
		)
		if last.Var {
			l--
		}

		for i := 0; i < l; i++ {
			argType = vm.ResolveType(args.GetOnly(i).Type())
			t = o.Params.Items[i].TypesSymbols
			if t != nil {
				if accept, err = t.Accept(vm, argType); err != nil {
					return
				} else if !accept {
					return NewArgumentTypeError(strconv.Itoa(i+1)+"st ("+o.Params.Items[i].Name+")", t.String(), argType.Name())
				}
			}
		}

		if last.Var {
			t = last.TypesSymbols
			args.WalkSkip(l, func(i int, arg Object) any {
				if accept, err = t.Accept(vm, arg.Type()); err == nil && !accept {
					err = NewArgumentTypeError(strconv.Itoa(i+1)+"st ("+o.Params.Items[i].Name+")", t.String(), arg.Type().Name())
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

func (o *CompiledFunction) ParamTypes(vm *VM) (types ParamsTypes, err error) {
	if o.Params.Typed() {
		types = make(ParamsTypes, o.Params.len)

		for i, p := range o.Params.Items {
			var ts ObjectTypes
			if len(p.TypesSymbols) > 0 {
				ts = make(ObjectTypes, len(p.TypesSymbols))
				for i2, symbol := range p.TypesSymbols {
					if typ, err := vm.GetSymbolValue(symbol); err != nil {
						return nil, err
					} else {
						ts[i2] = typ.(ObjectType)
					}
				}
			} else {
				ts = ObjectTypes{TAny}
			}
			types[i] = ts
		}
	} else {
		types = make(ParamsTypes, o.Params.len)
		for i := range types {
			types[i] = ObjectTypes{TAny}
		}
	}

	if o.Params.variadic {
		types[len(types)-1] = VarParamTypes(types[len(types)-1].Items())
	}
	return
}

func (o *CompiledFunction) Print(state *PrinterState) error {
	return state.WithoutRepr(func(s *PrinterState) error {
		return s.WriteString(ReprQuoteTyped("compiledFunction", o.HeaderString()))
	})
}

func hashData32(hash uint32, data []byte) uint32 {
	for _, c := range data {
		hash *= 16777619 // prime32
		hash ^= uint32(c)
	}
	return hash
}
