package gad

import "time"

func init() {
	BuiltinObjects.AddMethod(BuiltinInt, NewFunction(
		"goTimeToInt",
		func(c Call) (o Object, err error) {
			var (
				arg = &Arg{Name: "v"}
				get = GoTimeArg(arg)
			)

			if err = c.Args.Destructure(arg); err != nil {
				return
			}

			var (
				t       = get()
				unit, _ = c.NamedArgs.MustGetValueOrNil("unit").(Char)
			)

			switch unit {
			case 'n':
				return Int(t.UnixNano()), nil
			case 'm':
				return Int(t.UnixMicro()), nil
			case 'l':
				return Int(t.UnixMilli()), nil
			default:
				return Int(t.Unix()), nil
			}
		},
		FunctionWithUsage(`converts ReflectValue of Go time.Time or *time.Time to Unix time value elapsed since January 1, 1970 UTC`),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("v").Type(TReflectTimeType).Usage("go reflect time object")
		}),
		FunctionWithNamedParams(func(newParam func(name string) *NamedParamBuilder) {
			newParam("unit").Type(TChar).Usage(`
Available values:

'n'
	the number of nano seconds.
'm'
	the number of micro seconds.
'l'
	the number of milli seconds.
default
	the number of seconds.
`)
		}),
	))
}

var TReflectTimeType = ReflectTypeFor[time.Time]()

func GoTimeArg(arg *Arg) (get func() *time.Time) {
	arg.TypeAssertion = TypeAssertionFromTypes(TReflectTimeType)
	return func() *time.Time {
		return arg.Value.(*ReflectStruct).PtrValue().Interface().(*time.Time)
	}
}
