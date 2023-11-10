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
		if cf, ok := bc.Constants[i].(*CompiledFunction); ok {
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

type Params struct {
	Len int
	Var bool
}

func (p Params) String() string {
	s := strconv.Itoa(p.Len)
	if p.Var {
		s += ", variadic"
	}
	return s
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
			s[i] = param.Name + " (variadic)"
		}
	}
	return strings.Join(s, ", ")
}

// CompiledFunction holds the constants and instructions to pass VM.
type CompiledFunction struct {
	// number of local variabls including parameters NumLocals>=NumParams
	NumLocals    int
	Instructions []byte
	Free         []*ObjectPtr
	// SourceMap holds the index of instruction and token's position.
	SourceMap map[int]int

	Params      Params
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
	var s []string
	if o.Params.Len > 0 {
		s = append(s, " [params: "+o.Params.String()+"]")
	}
	if o.NamedParams.len > 0 {
		s = append(s, " [named params: "+o.NamedParams.String()+"]")
	}
	return "<compiledFunction" + strings.Join(s, "") + ">"
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
	if o.NumLocals != other.NumLocals ||
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
	return NewInvoker(c.VM, o).Invoke(c.Args, &c.NamedArgs)
}

func (o *CompiledFunction) SetNamedParams(params ...*NamedParam) {
	o.NamedParams = *NewNamedParams(params...)
}

func hashData32(hash uint32, data []byte) uint32 {
	for _, c := range data {
		hash *= 16777619 // prime32
		hash ^= uint32(c)
	}
	return hash
}
