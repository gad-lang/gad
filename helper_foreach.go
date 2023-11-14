package gad

type ForEach struct {
	it                 Iterator
	args               Array
	startArgValueIndex int
	caller             VMCaller
	k, v               Object
	err                error
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
		f.k = f.it.Key()
		f.v, f.err = f.it.Value()
	}
	return
}

func (f *ForEach) Call() (_ Object, err error) {
	if f.err != nil {
		return nil, f.err
	}

	f.args[f.startArgValueIndex] = f.v
	f.args[f.startArgValueIndex+1] = f.k

	return f.caller.Call()
}

func (f *ForEach) Key() Object {
	return f.k
}

func (f *ForEach) Value() Object {
	return f.v
}
