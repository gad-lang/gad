package giom

import (
	"strconv"
	"strings"

	"github.com/gad-lang/gad"
)

var (
	BuiltinEscape = &gad.Function{
		FuncName: "giom.escape",
		Module:   ModuleSpec,
		Value: func(call gad.Call) (_ gad.Object, err error) {
			if err = call.Args.CheckLen(1); err != nil {
				return
			}

			var (
				value = call.Args.GetOnly(0)
				s     string
			)

			switch t := value.(type) {
			case gad.RawStr:
				s = string(t)
			case gad.Str:
				s = string(t)
			default:
				var v gad.Object
				if v, err = call.VM.Builtins.Call(gad.BuiltinStr, call); err != nil {
					return
				}
				s = string(v.(gad.Str))
			}

			return gad.RawStr(s), nil
		},
	}

	AttrFunc = func(vm *gad.VM, name, value gad.Object) (ret gad.RawStr, err error) {
		var (
			toRawStr = vm.Builtins.ArgsInvoker(gad.BuiltinRawStr, gad.Call{VM: vm})
		)

		if value.IsFalsy() {
			return
		}

		if _, ok := name.(gad.RawStr); !ok {
			if name, err = toRawStr(name); err != nil {
				return
			}
		}

		switch t := value.(type) {
		case gad.Array:
			var b strings.Builder
			for _, o := range t {
				if o.IsFalsy() {
					continue
				}
				if _, ok := o.(gad.RawStr); !ok {
					if o, err = toRawStr(o); err != nil {
						return
					}
				}
				b.WriteString(string(o.(gad.RawStr)))
				b.WriteString(" ")
			}
			value = gad.RawStr(strings.TrimSpace(b.String()))
		case gad.RawStr:
		case gad.Flag:
			if t {
				return gad.RawStr(name.ToString()), nil
			}
			return "", nil
		default:
			if value, err = toRawStr(value); err != nil {
				return
			}
		}
		return gad.RawStr(name.ToString() + "=" + strconv.Quote(value.ToString())), nil
	}

	BuiltinAttr = &gad.Function{
		FuncName: "giom.attr",
		Module:   ModuleSpec,
		Value: func(call gad.Call) (ret gad.Object, err error) {
			if err = call.Args.CheckLen(2); err != nil {
				return
			}

			ret, err = AttrFunc(call.VM, call.Args.GetOnly(0), call.Args.GetOnly(1))
			return
		},
	}

	BuiltinAttrs = &gad.Function{
		FuncName: "giom.attrs",
		Module:   ModuleSpec,
		Value: func(call gad.Call) (_ gad.Object, err error) {
			var (
				b      strings.Builder
				rs     gad.RawStr
				keys   []string
				d      = make(gad.Dict)
				class  []string
				style  []string
				filter func(arr gad.Array) (ret gad.Array)
			)

			filter = func(arr gad.Array) (ret gad.Array) {
				for _, v := range arr {
					switch t := v.(type) {
					case *gad.KeyValue:
						if !t.K.IsFalsy() {
							ret = append(ret, t.V)
						}
					case gad.Array:
						ret = append(ret, filter(t)...)
					default:
						if !t.IsFalsy() {
							ret = append(ret, t)
						}
					}
				}
				return
			}

			cb := func(na *gad.KeyValue) (err error) {
				var k string
				switch t := na.K.(type) {
				case gad.Bool:

				case gad.Str:
					k = string(t)
				case gad.RawStr:
					k = string(t)
				default:
					var s gad.Str
					if s, err = gad.ToStr(call.VM, t); err != nil {
						return
					}
					k = string(s)
				}
				switch k {
				case "class":
					if !na.V.IsFalsy() {
						switch t := na.V.(type) {
						case gad.Str:
							if len(t) > 0 {
								class = append(class, string(t))
							}
						case gad.RawStr:
							if len(t) > 0 {
								class = append(class, string(t))
							}
						case *gad.KeyValue:
							if !t.K.IsFalsy() {
								var arr gad.Array
								switch t := t.V.(type) {
								case gad.Array:
									arr = t
								default:
									arr = gad.Array{t}
								}
								for _, o := range filter(arr) {
									var s gad.Str
									if s, err = gad.ToStr(call.VM, o); err != nil {
										return
									}
									class = append(class, string(s))
								}
							}
						case gad.Array:
							for _, o := range filter(t) {
								var s gad.Str
								if s, err = gad.ToStr(call.VM, o); err != nil {
									return
								}
								class = append(class, string(s))
							}
						}
					}
				case "style":
					switch t := na.V.(type) {
					case gad.Str:
						if len(t) > 0 {
							style = append(style, string(t))
						}
					case gad.RawStr:
						if len(t) > 0 {
							style = append(style, string(t))
						}
					case *gad.KeyValue:
						if !t.K.IsFalsy() {
							var arr gad.Array
							switch t := t.V.(type) {
							case gad.Array:
								arr = t
							default:
								arr = gad.Array{t}
							}
							for _, o := range filter(arr) {
								var s gad.Str
								if s, err = gad.ToStr(call.VM, o); err != nil {
									return
								}
								style = append(style, string(s))
							}
						}
					case gad.Array:
						for _, o := range filter(t) {
							var s gad.Str
							if s, err = gad.ToStr(call.VM, o); err != nil {
								return
							}
							style = append(style, string(s))
						}
					case gad.Dict:
						for key, o := range t {
							var s gad.Str
							if s, err = gad.ToStr(call.VM, o); err != nil {
								return
							}
							style = append(style, key+":"+string(s))
						}
					}
				default:
					v := na.V
					switch t := v.(type) {
					case *gad.KeyValue:
						if t.K.IsFalsy() {
							return
						}
						v = t.V
					}
					if _, ok := d[k]; !ok {
						keys = append(keys, k)
					}
					d[k] = v
				}
				return err
			}

			err = call.NamedArgs.Walk(func(na *gad.KeyValue) (err error) {
				var v gad.Object = na
				switch t := na.K.(type) {
				case gad.Bool:
					if !t {
						return
					}
					v = na.V
				}
				switch t := v.(type) {
				case gad.Array:
					return gad.ItemsOfCb(call.VM, &gad.NamedArgs{}, cb, na.V)
				case *gad.KeyValue:
					return cb(t)
				case gad.KeyValueArray:
					for _, value := range t {
						if err = cb(value); err != nil {
							return
						}
					}
					return
				default:
					return cb(na)
				}
			})

			if err != nil {
				return
			}

			for _, key := range keys {
				if rs, err = AttrFunc(call.VM, gad.Str(key), d[key]); err == nil && rs != "" {
					b.WriteString(" " + string(rs))
				}
			}

			if len(class) > 0 {
				b.WriteString(" class=")
				b.WriteString(strconv.Quote(strings.Join(class, " ")))
			}

			if len(style) > 0 {
				b.WriteString(" style=")
				b.WriteString(strconv.Quote(strings.Join(style, "; ")))
			}

			return gad.RawStr(b.String()), nil
		},
	}

	BuiltinTextWrite = &gad.Function{
		FuncName: "giom.write",
		Module:   ModuleSpec,
		Value: func(call gad.Call) (_ gad.Object, err error) {
			return call.VM.Builtins.Call(gad.BuiltinWrite, call)
		},
	}
)

// AppendBuiltins registers the giom module as a non-loadable builtin namespace,
// making giom.escape, giom.attr, giom.attrs, and giom.write available globally.
func AppendBuiltins(b *gad.Builtins) *gad.Builtins {
	mod := newModule()
	b.Set(ModuleSpec.Name, mod)
	for k, v := range mod {
		b.Set(ModuleSpec.Name+"."+k, v)
	}
	return b
}
