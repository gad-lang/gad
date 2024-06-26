// Copyright (c) 2022-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// EXPERIMENTAL code generation tool, use it at your own risk.

// Inspired by golang.org/x/sys/windows/cmd/mkwinsyscall package.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// We support (gad.Object) or (gad.Object, error) or (error) results.

const gadCallablePrefix = "//gad:callable"
const gadDot = "gad."

type converterFunc func(index int, argsName string, p *Param) string

var converters = map[string]any{
	"string":           "gad.ToGoString",
	"[]byte":           "gad.ToGoByteSlice",
	"int":              "gad.ToGoInt",
	"int64":            "gad.ToGoInt64",
	"uint64":           "gad.ToGoUint64",
	"float64":          "gad.ToGoFloat64",
	"rune":             "gad.ToGoRune",
	"bool":             "gad.ToGoBool",
	"gad.String":       "gad.ToString",
	"gad.Bytes":        "gad.ToBytes",
	"gad.Int":          "gad.ToInt",
	"gad.Uint":         "gad.ToUint",
	"gad.Float":        "gad.ToFloat",
	"gad.Char":         "gad.ToChar",
	"gad.Bool":         "gad.ToBool",
	"gad.Flag":         "gad.ToFlag",
	"gad.Array":        "gad.ToArray",
	"gad.Dict":         "gad.ToMap",
	"gad.CallerObject": "",
	"*gad.SyncMap":     "gad.ToSyncMap",
	"gad.Object": converterFunc(func(index int, argsName string, p *Param) string {
		return fmt.Sprintf("%s := %s.Args.Get(%d)", p.Name, argsName, index)
	}),
}

var builtinTypeAlias = map[string]string{
	"_":                "p", // p is reserved for pointer prefix
	"gad.Object":       "O",
	"gad.String":       "S",
	"gad.Bytes":        "B",
	"gad.Dict":         "M",
	"gad.SyncMap":      "M2",
	"gad.Array":        "A",
	"gad.Float":        "F",
	"gad.Int":          "I",
	"gad.Uint":         "U",
	"gad.Char":         "C",
	"gad.CallerObject": "Co",
	"gad.VM":           "Vm",
	"string":           "s",
	"bool":             "b",
	"flag":             "f",
	"byte":             "b1",
	"[]byte":           "b2",
	"int":              "i",
	"int64":            "i64",
	"uint64":           "u64",
	"float64":          "f64",
	"rune":             "r",
	"error":            "e",
}

var gadTypeNames = map[string]string{
	"gad.Object":       "object",
	"gad.String":       "str",
	"gad.Bytes":        "bytes",
	"gad.Dict":         "map",
	"gad.SyncMap":      "syncMap",
	"gad.Array":        "array",
	"gad.Float":        "float",
	"gad.Int":          "int",
	"gad.Uint":         "uint",
	"gad.Char":         "char",
	"gad.CallerObject": "CallerObject",
	"gad.Flag":         "gad.Flag",
	"flag":             "Flag",
	"string":           "str",
	"byte":             "char",
	"[]byte":           "bytes",
	"int64":            "int",
	"uint64":           "uint",
	"float64":          "float",
	"rune":             "char",
	"error":            "error",
	"*Time":            "time",
	"*Location":        "location",
}

var ordinals = [...]string{
	0: "th",
	1: "st",
	2: "nd",
	3: "rd",
	4: "th",
	5: "th",
	6: "th",
	7: "th",
	8: "th",
	9: "th",
}

func ordinalize(num int) string {
	suffix := ordinals[num%10]
	if vv := num % 100; vv >= 11 && vv <= 13 {
		suffix = "th"
	}
	return strconv.Itoa(num) + suffix
}

var (
	filename = flag.String("output", "", "output file name (standard output if omitted)")
	export   = flag.Bool("export", false, "export auto generated function names")
)

var packageName string

func packagename() string {
	return packageName
}

func gaddot() string {
	if packageName == "gad" {
		return ""
	}
	return gadDot
}

func trim(s string) string {
	return strings.Trim(s, " \t")
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mkcallable [flags] [path ...]\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) <= 0 {
		fmt.Fprintf(os.Stderr, "no files provided to parse\n")
		usage()
	}

	src, err := ParseFiles(flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	if err := src.Generate(&buf); err != nil {
		log.Fatal(err)
	}

	// data := buf.Bytes()

	data, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("error:%v\n%s", err, buf.Bytes())
	}
	if *filename == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = ioutil.WriteFile(*filename, data, 0644)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// ParseFiles parses files listed in files and extracts all directives listed in
// gad:callable comments. It returns *Source if successful.
func ParseFiles(files []string) (*Source, error) {
	src := &Source{
		Funcs:           make([]*Fn, 0),
		GoImports:       []Pkg{},
		ExternalImports: []Pkg{},
	}
	for _, file := range files {
		if err := src.ParseFile(file); err != nil {
			return nil, err
		}
	}
	if gaddot() != "" {
		src.ExternalImports = append(src.ExternalImports, Pkg{Path: "github.com/gad-lang/gad"})
	}
	err := src.checkConverters()
	return src, err
}

// Pkg represents a import package in Source.
type Pkg struct {
	Alias string
	Path  string
}

// HelperImport is a helper used in template returning import path
// and alias if exists.
func (p *Pkg) HelperImport() string {
	if p.Alias == "" {
		return strconv.Quote(p.Path)
	}
	return p.Alias + " " + strconv.Quote(p.Path)
}

// Source functions and imports.
type Source struct {
	Funcs           []*Fn
	GoImports       []Pkg
	ExternalImports []Pkg
	ExtendedOnly    bool
}

// Generate output source file from a source set src.
func (src *Source) Generate(w io.Writer) error {
	funcMap := template.FuncMap{
		"packagename": packagename,
		"gaddot":      gaddot,
	}
	t := template.Must(template.New("main").Funcs(funcMap).Parse(srcTemplate))
	err := t.Execute(w, src)
	if err != nil {
		return errors.New("failed to execute template: " + err.Error())
	}
	return nil
}

// GoImport adds a go runtime package to be imported in output.
func (src *Source) GoImport(alias, path string) {
	src.GoImports = append(src.GoImports, Pkg{Alias: alias, Path: path})
	sortPackages(src.GoImports)
}

// ExternalImport adds an external package to be imported in output.
func (src *Source) ExternalImport(alias, path string) {
	src.ExternalImports = append(src.ExternalImports, Pkg{Alias: alias, Path: path})
	sortPackages(src.ExternalImports)
}

func sortPackages(pkgs []Pkg) {
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Path < pkgs[j].Path
	})
}

// ParseFile adds additional file path to a source set src.
func (src *Source) ParseFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	for s.Scan() {
		t := trim(s.Text())
		if !strings.HasPrefix(t, gadCallablePrefix) {
			continue
		}
		tt := t[len(gadCallablePrefix):]
		if tt[0] == ':' {
			if strings.HasPrefix(tt, ":import") {
				if err := src.parseImport(tt[7:]); err != nil {
					return err
				}
			} else if strings.HasPrefix(tt, ":convert") {
				if err := src.parseConvert(tt[8:]); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unknown directive '%s'", t)
			}
			continue
		}
		if !(tt[0] == ' ' || tt[0] == '\t') {
			continue
		}
		f, err := newFn(tt[1:])
		if err != nil {
			return err
		}
		src.Funcs = append(src.Funcs, f)
	}
	if err := s.Err(); err != nil {
		return err
	}

	// get package name
	fset := token.NewFileSet()
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	pkg, err := parser.ParseFile(fset, "", file, parser.PackageClauseOnly)
	if err != nil {
		return err
	}
	packageName = pkg.Name.Name

	return nil
}

func (src *Source) parseImport(s string) error {
	path := trim(s)
	alias := ""

	if path == "" {
		// ignore empty import directive
		return nil
	}

	// Two ways to define imports
	// 1.
	// //gad:callable:import "path/to/package"
	//
	// 2.
	// //gad:callable:import alias "path/to/package"

	// Check first char is " to determine if alias is provided
	if path[0] == '"' {
		p, err := strconv.Unquote(path)
		if err != nil {
			return fmt.Errorf("cannot unquote %v, line:%s", err, path)
		}
		path = p
	} else {
		parts := strings.Fields(path)
		if len(parts) != 2 {
			return fmt.Errorf("invalid import directive, line: %s", path)
		}
		alias, path = parts[0], parts[1]
		p, err := strconv.Unquote(path)
		if err != nil {
			return fmt.Errorf("cannot unquote %v, line:%s", err, path)
		}
		path = p
	}

	if exist, err := src.checkImports(alias, path); err != nil {
		return err
	} else if exist {
		return nil
	}

	// Simply check dot(.) in path to determine if it is an external package.
	if strings.Count(path, ".") > 0 {
		src.ExternalImport(alias, path)
	} else {
		src.GoImport(alias, path)
	}
	return nil
}

func (src *Source) checkImports(alias, path string) (bool, error) {
	all := make([]Pkg, len(src.ExternalImports)+len(src.GoImports))
	copy(all, src.ExternalImports)
	copy(all[len(src.ExternalImports):], src.GoImports)

	for _, p := range all {
		if p.Path == path {
			if p.Alias == alias {
				return true, nil
			}
			return true, fmt.Errorf("double import with different alias, path: %s", path)
		}
	}
	return false, nil
}

func (src *Source) parseConvert(s string) error {
	// Examples
	//
	// //gad:callable:convert *Time ToTime
	//
	// //gad:callable:convert *string ToStringPointer
	//

	s = trim(s)
	parts := strings.Fields(s)
	if len(parts) != 2 {
		if len(parts) == 0 {
			// ignore empty lines
			return nil
		}
		return fmt.Errorf("invalid convert directive, line: %s", s)
	}
	typeName, converter := parts[0], parts[1]
	converters[typeName] = converter
	return nil
}

func (src *Source) checkConverters() error {
	for _, fn := range src.Funcs {
		for _, p := range fn.Params {
			if _, ok := converters[p.Type]; ok {
				continue
			}
			if !strings.HasSuffix(p.Type, "VM") {
				if _, ok := converters[gadDot+p.Type]; !ok {
					return fmt.Errorf("converter is not found for type: %s", p.Type)
				}
			}
		}
	}
	return nil
}

// Param is function parameter
type Param struct {
	Named bool
	Name  string
	Type  string
	fn    *Fn
	idx   int
}

// IsError determines if p parameter is used to return error.
func (p *Param) IsError() bool {
	return p.Type == "error"
}

// HelperAssignVar is a helper function used in template to create variable
// assignment with appropriate converter.
func (p *Param) HelperAssignVar() string {
	conv := converters[p.Type]
	if conv == nil {
		conv = converters[gadDot+p.Type]
		if conv == nil {
			conv = "CONVERTER_NOT_FOUND"
		}
	}
	if conv != nil {
		if fn, ok := conv.(converterFunc); ok {
			return fn(p.idx, p.fn.argsName, p)
		}
	}
	if gaddot() == "" {
		conv = strings.TrimPrefix(conv.(string), gadDot)
	}

	gadTypeName := p.gadTypeName()

	return fmt.Sprintf(`%s, ok := %s(%s[%d])
		if !ok {
			return %sNil, %sNewArgumentTypeError("%s", "%s", %s[%d].Type().Name())
		}`,
		p.Name, conv, p.fn.argsName, p.idx,
		gaddot(), gaddot(), ordinalize(p.idx+1), gadTypeName, p.fn.argsName, p.idx,
	)
}

// HelperAssignVarEx is a helper function used in template to create variable
// assignment with appropriate converter for extended API.
func (p *Param) HelperAssignVarEx() string {
	conv := converters[p.Type]
	if conv == nil {
		if strings.Contains(p.Type, "VM") {
			return p.Name + ":= c.VM"
		} else {
			conv = converters[gadDot+p.Type]
			if conv == nil {
				conv = "CONVERTER_NOT_FOUND"
			}
		}
	}
	if conv != nil {
		if fn, ok := conv.(converterFunc); ok {
			return fn(p.idx, p.fn.argsName, p)
		}
	}
	if gaddot() == "" {
		conv = strings.TrimPrefix(conv.(string), gadDot)
	}

	gadTypeName := p.gadTypeName()

	if p.Named {
		return ""
	}

	if conv, ok := conv.(string); ok && conv == "" {
		return fmt.Sprintf(`%s, ok := %s.Args.Get(%d).(%s)
		if !ok {
			return %sNil, %sNewArgumentTypeError("%s", "%s", %s.Args.Get(%d).Type().Name())
		}`,
			p.Name, p.fn.argsName, p.idx, gadTypeName,
			gaddot(), gaddot(), ordinalize(p.idx+1), gadTypeName, p.fn.argsName, p.idx,
		)
	}

	return fmt.Sprintf(`%s, ok := %s(%s.Args.Get(%d))
		if !ok {
			return %sNil, %sNewArgumentTypeError("%s", "%s", %s.Args.Get(%d).Type().Name())
		}`,
		p.Name, conv, p.fn.argsName, p.idx,
		gaddot(), gaddot(), ordinalize(p.idx+1), gadTypeName, p.fn.argsName, p.idx,
	)
}

func (p *Param) gadTypeName() string {
	n := gadTypeNames[p.Type]
	if n == "" {
		n = gadTypeNames[gadDot+p.Type]
		if n == "" {
			return p.Type
		}
	}
	return n
}

// join concatenates parameters ps into a string with sep separator.
// For parameter is converted into string by applying fn to it
// before conversion.
func join(ps []*Param, fn func(*Param) string, sep string) string {
	if len(ps) == 0 {
		return ""
	}
	a := make([]string, 0)
	for _, p := range ps {
		a = append(a, fn(p))
	}
	return strings.Join(a, sep)
}

// Rets describes function return parameters.
type Rets struct {
	Name         string
	Type         string
	ReturnsError bool
}

// ToParams converts r into slice of *Param.
func (r *Rets) ToParams() []*Param {
	ps := make([]*Param, 0)
	if r.Name != "" {
		ps = append(ps, &Param{Name: r.Name, Type: r.Type})
	}
	if r.ReturnsError {
		ps = append(ps, &Param{Name: "err", Type: "error"})
	}
	return ps
}

// List returns source code of syscall return parameters.
func (r *Rets) List() string {
	s := join(r.ToParams(), func(p *Param) string { return p.Type }, ", ")
	if len(s) > 0 {
		s = "(" + s + ")"
	}
	return s
}

// Fn describes callable function.
type Fn struct {
	Name          string
	Params        []*Param
	Rets          *Rets
	fnName        string
	argsName      string
	retName       string
	namedArgsName string
	errName       string
	src           string
	vmarg         bool
}

func newFn(s string) (*Fn, error) {
	s = trim(s)
	f := &Fn{
		Rets: &Rets{},
		src:  s,
	}
	// function name and args
	prefix, body, s, found := extractSection(s, '(', ')')
	if !found || prefix == "" {
		return nil, fmt.Errorf("could not extract function name and parameters from %q", f.src)
	}
	f.Name = prefix
	var err error
	f.Params, err = extractParams(body, f)
	if err != nil {
		return nil, err
	}

	if len(f.Params) > 0 && strings.HasSuffix(f.Params[0].Type, "VM") {
		f.vmarg = true

		for _, param := range f.Params[1:] {
			param.idx--
		}
	}

	// return values
	_, body, s, found = extractSection(s, '(', ')')
	if found {
		r, err := extractParams(body, f)
		if err != nil {
			return nil, err
		}
		switch len(r) {
		case 0:
		case 1:
			if r[0].IsError() {
				f.Rets.ReturnsError = true
			} else {
				f.Rets.Name = r[0].Name
				f.Rets.Type = r[0].Type
			}
		case 2:
			if !r[1].IsError() {
				return nil, fmt.Errorf("only last error is allowed as second return value in %q", f.src)
			}
			f.Rets.ReturnsError = true
			f.Rets.Name = r[0].Name
			f.Rets.Type = r[0].Type
		default:
			return nil, fmt.Errorf("too many return values in %q", f.src)
		}
	}
	if s != "" {
		return nil, fmt.Errorf("extra arguments in %q", f.src)
	}
	f.genFuncName()
	f.setVarNames()
	return f, nil
}

// ParamList returns source code for function f parameters.
func (f *Fn) ParamList() string {
	return join(f.Params, func(p *Param) string { return p.Type }, ", ")
}

// FuncName returns the Fn's Name field.
func (f *Fn) FuncName() string { return f.Name }

// FnName returns the fn parameter name.
func (f *Fn) FnName() string { return f.fnName }

// ArgsName returns the args parameter name.
func (f *Fn) ArgsName() string { return f.argsName }

// RetName returns the ret variable name.
func (f *Fn) RetName() string { return f.retName }

// ErrName returns the err variable name.
func (f *Fn) ErrName() string { return f.errName }

// SourceString returns the source string of the function read from comment.
func (f *Fn) SourceString() string { return f.src }

// HelperCheckNumArgs is an helper used in template to return code block to
// check number of arguments.
func (f *Fn) HelperCheckNumArgs() string {
	return fmt.Sprintf(`if len(%s)!=%d {
			return %sNil, %sErrWrongNumArguments.NewError("want=%d got=" + strconv.Itoa(len(%s)))
	    }`, f.argsName, len(f.Params), gaddot(), gaddot(), len(f.Params), f.argsName)
}

// HelperCheckNumArgsEx is an helper used in template to return code block to
// check number of arguments for extended API.
func (f *Fn) HelperCheckNumArgsEx() string {
	var numArgs int
	for i, p := range f.Params {
		if i == 0 && strings.HasSuffix(p.Type, "VM") {
			continue
		}
		if !p.Named {
			numArgs++
		}
	}
	return fmt.Sprintf(`if err := %s.Args.CheckLen(%d); err!=nil {
			return %sNil, err
	    }`, f.argsName, numArgs, gaddot())
}

// HelperCheckNamedArgs is an helper used in template to return code block to
// check named arguments.
func (f *Fn) HelperCheckNamedArgs() string {
	if f.namedArgsName == "" {
		return ""
	}
	var (
		s     []string
		names []string
	)
	for _, p := range f.Params {
		if p.Named {
			conv := converters[p.Type]
			if conv == nil {
				conv = converters[gadDot+p.Type]
				if conv == nil {
					conv = "CONVERTER_NOT_FOUND"
				}
			}
			if conv != nil {
				if fn, ok := conv.(converterFunc); ok {
					return fn(p.idx, p.fn.argsName, p)
				}
			}
			if gaddot() == "" {
				conv = strings.TrimPrefix(conv.(string), gadDot)
			}

			if cs, ok := conv.(string); ok && cs == "" {
				conv = fmt.Sprintf("v.(%s)", p.Type)
			} else {
				conv = fmt.Sprintf("%s(v)", cs)
			}

			names = append(names, p.Name+"_")
			s = append(s, fmt.Sprintf(`
			%s %[4]s
			%[1]s_ = &%[3]sNamedArgVar{
				Name:        %[1]q,
				TypeAssertion: %[3]sNewTypeAssertion(%[3]sTypeAssertionHandlers{
					%[4]q: func(v %[3]sObject) (ok bool) {
						%[1]s, ok = %s
						return
					},
				}),
			}`, p.Name, conv, gaddot(), p.gadTypeName()))
		}
	}
	return fmt.Sprintf(`
		var (
			%s
		);
		if err := %s.NamedArgs.Get(%s); err!=nil {
			return %sNil, err
		}`, strings.Join(s, ""), f.argsName, strings.Join(names, ", "), gaddot())
}

// HelperCall is an helper used in template to return function call block with
// assignments of result variables.
func (f *Fn) HelperCall() string {
	const retPrefix = "\n        " // just for formatting reasons.
	var (
		left string
		ret  = retPrefix + f.retName + " = " + gaddot() + "Nil"
	)
	if rets := f.Rets.ToParams(); len(rets) > 0 {
		switch len(rets) {
		case 1:
			if f.Rets.ReturnsError {
				left = f.errName + " = "
				ret = retPrefix + f.retName + " = " + gaddot() + "Nil"
			} else {
				left = f.retName + " = "
				ret = ""
			}
		case 2:
			left = f.retName + ", " + f.errName + " = "
			ret = ""
		}
	}

	return fmt.Sprintf(`%s%s(%s)%s`,
		left,
		f.fnName,
		join(f.Params, func(p *Param) string {
			return p.Name
		}, ", "),
		ret,
	)
}

func (f *Fn) setVarNames() {
	names := map[string]struct{}{}
	for _, p := range f.Params {
		names[p.Name] = struct{}{}
		if p.Named && f.namedArgsName == "" {
			f.namedArgsName = genVarName("na", names)
		}
	}

	// Check parameter names to create unique name for default variable names.
	f.fnName = genVarName("fn", names)
	f.argsName = genVarName("c", names)
	f.retName = genVarName("ret", names)
	f.errName = genVarName("err", names)
}

func genVarName(prefix string, names map[string]struct{}) string {
	name := prefix
	i := 0
	for {
		_, ok := names[name]
		if !ok {
			break
		}
		name = prefix + strconv.Itoa(i)
		i++
	}
	return name
}

func (f *Fn) genFuncName() {
	if f.Name != "func" {
		return
	}
	var b strings.Builder
	if *export {
		b.WriteString("FuncP")
	} else {
		b.WriteString("funcP")
	}

	gen := func(params []*Param) {
		for _, param := range params {
			if strings.HasPrefix(param.Type, "*") { // pointer
				b.WriteString("p")
			}

			n := builtinTypeAlias[param.Type]
			if n == "" {
				n = builtinTypeAlias[gadDot+param.Type]
			}
			if n == "" {
				i := strings.Index(param.Type, ".")
				typ := strings.TrimPrefix(param.Type[i+1:], "*")
				if i > -1 {
					n = builtinTypeAlias[typ]
				}
				if n == "" {
					n = typ + "_"
				}
			}
			b.WriteString(n)
			if param.Named {
				b.WriteString("_")
				b.WriteString(param.Name)
				b.WriteString("_")
			}
		}
	}
	gen(f.Params)
	b.WriteString("R")
	gen(f.Rets.ToParams())
	f.Name = b.String()
}

// extractSection extracts text out of string s starting after start
// and ending just before end. found return value will indicate success,
// and prefix, body and suffix will contain correspondent parts of string s.
func extractSection(s string, start, end rune) (prefix, body, suffix string, found bool) {
	s = trim(s)
	if strings.HasPrefix(s, string(start)) {
		// no prefix
		body = s[1:]
	} else {
		a := strings.SplitN(s, string(start), 2)
		if len(a) != 2 {
			return "", "", s, false
		}
		prefix = a[0]
		body = a[1]
	}
	a := strings.SplitN(body, string(end), 2)
	if len(a) != 2 {
		return "", "", "", false
	}
	return prefix, a[0], a[1], true
}

// extractParams parses s to extract function parameters.
func extractParams(s string, f *Fn) ([]*Param, error) {
	s = trim(s)
	if s == "" {
		return nil, nil
	}
	a := strings.Split(s, ",")
	ps := make([]*Param, len(a))
	for i := range ps {
		var (
			s2    = trim(a[i])
			b     []string
			named bool
		)
		if strings.ContainsRune(s2, '=') {
			named = true
			b = strings.Split(s2, "=")
			if len(b) != 2 {
				if len(b) != 2 {
					return nil, fmt.Errorf("could not extract function parameter from %q", s2)
				}
			}
			for i2, s3 := range b {
				b[i2] = trim(s3)
			}
		} else {
			b = strings.Split(s2, " ")
			if len(b) != 2 {
				b = strings.Split(s2, "\t")
				if len(b) != 2 {
					return nil, fmt.Errorf("could not extract function parameter from %q", s2)
				}
			}
		}
		p := &Param{
			Named: named,
			Name:  trim(b[0]),
			Type:  trim(b[1]),
			fn:    f,
			idx:   i,
		}

		ps[i] = p
		if strings.Contains(ps[i].Type, "...") {
			return nil, fmt.Errorf("variadic parameter is not supported from %q", s2)
		}
	}
	return ps, nil
}

const srcTemplate = `
{{define "main"}}// Code generated by 'go generate'; DO NOT EDIT.

package {{packagename}}

import ({{range .GoImports}}
	{{.HelperImport}}{{end}}
	{{range .ExternalImports}}
	{{.HelperImport}}{{end}}
)

{{range .Funcs}}{{template "funcbody" .}}{{end}}

{{end}}

{{define "gadcallparamsEx"}}({{.ArgsName}} {{gaddot}}Call){{end}}

{{define "checknumargsEx"}}{{.HelperCheckNumArgsEx}}{{end}}

{{define "checkNamedArgs"}}{{.HelperCheckNamedArgs}}{{end}}

{{define "assignvarsEx"}}{{range .Params}}
		{{.HelperAssignVarEx}}{{end}}
{{end}}


{{define "funcbody"}}
// {{.FuncName}} is a generated function to make {{gaddot}}CallableFunc.
// Source: {{.SourceString}}
func {{.FuncName}}({{.FnName}} func({{.ParamList}}) {{.Rets.List}}) {{gaddot}}CallableFunc {
	return func{{template "gadcallparamsEx" .}} {{template "gadresults" .}} {
		{{template "checknumargsEx" .}}
		{{template "checkNamedArgs" .}}
		{{template "assignvarsEx" .}}
		{{template "call" .}}
		return
	}
}
{{end}}

{{define "gadcallparams"}}({{.ArgsName}} ...{{gaddot}}Object){{end}}

{{define "gadresults"}}({{.RetName}} {{gaddot}}Object, {{.ErrName}} error){{end}}

{{define "checknumargs"}}{{.HelperCheckNumArgs}}{{end}}

{{define "assignvars"}}{{range .Params}}
		{{.HelperAssignVar}}{{end}}
{{end}}

{{define "call"}}{{.HelperCall}}{{end}}
`
