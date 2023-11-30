package gad

type ForEach struct {
	it                 Iterator
	args               Array
	startArgValueIndex int
	caller             VMCaller
	Key,
	Value Object
	err error
}

func NewForEach(it Iterator, args Array, startArgValueIndex int, caller VMCaller) *ForEach {
	return &ForEach{
		it:                 it,
		args:               args,
		startArgValueIndex: startArgValueIndex,
		caller:             caller,
	}
}

func (f *ForEach) Next() (ok bool) {
	if ok = f.it.Next(); ok {
		f.Key = f.it.Key()
		f.Value, f.err = f.it.Value()
	}
	return
}

func (f *ForEach) Call() (_ Object, err error) {
	if f.err != nil {
		return nil, f.err
	}

	f.args[f.startArgValueIndex] = f.Value
	f.args[f.startArgValueIndex+1] = f.Key

	return f.caller.Call()
}
