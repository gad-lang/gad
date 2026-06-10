package gad

func init() {
	AddMethod(TIterator, NewFunction(
		"",
		func(c Call) (o Object, err error) {
			if err = c.Args.CheckLen(1); err != nil {
				return
			}
			_, o, err = ToStateIterator(c.VM, c.Args.GetOnly(0), &c.NamedArgs)
			return
		},
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("iterable").Type(TAny).Usage("An iterable object")
		}),
	))

	TZipIterator.WithConstructor(
		&Function{
			Value: func(c Call) (o Object, err error) {
				var it = make([]Iterator, c.Args.Length())
				c.Args.Walk(func(i int, arg Object) any {
					if _, it[i], err = ToIterator(c.VM, arg, &c.NamedArgs); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return
				}

				o = IteratorObject(ZipIterator(it...))
				return
			},
		})
}
