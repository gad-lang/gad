package gad

const (
	PrintStateOptionIndent           = "indent"
	PrintStateOptionMaxDepth         = "maxDepth"
	PrintStateOptionRaw              = "raw"
	PrintStateOptionZeros            = "zeros"
	PrintStateOptionAnonymous        = "anonymous"
	PrintStateOptionSortKeys         = "sortKeys"
	PrintStateOptionIndexes          = "indexes"
	PrintStateOptionTypesAsFullNames = "typesAsFullNames"
	PrintStateOptionRepr             = "repr"
	PrintStateOptionBytesToHex       = "bytesToHex"
	PrintStateOptionQuoteStr         = "quoteStr"
)

type PrintStateOptionSortType uint8

const (
	PrintStateOptionSortTypeAuto PrintStateOptionSortType = iota
	PrintStateOptionSortTypeAscending
	PrintStateOptionSortTypeDescending
)

type PrinterStateOptions map[string]Object

func (o PrinterStateOptions) Dict() Dict {
	return Dict(o)
}

func (o PrinterStateOptions) Indent() (v string, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionIndent]; ok {
		switch vt := vo.(type) {
		case Flag:
			if vt {
				v = "\t"
			}
		default:
			v = vo.ToString()
		}
	}
	return
}

func (o PrinterStateOptions) DefaultIndent() {
	o.SetIndent(Yes)
}

func (o PrinterStateOptions) SetIndent(v Object) {
	o[PrintStateOptionIndent] = v
}

func (o PrinterStateOptions) WithIndent() PrinterStateOptions {
	o.SetIndent(Yes)
	return o
}

func (o PrinterStateOptions) MaxDepth() (v int64, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionMaxDepth]; ok {
		var vi Int
		vi, _ = vo.(Int)
		v = int64(vi)
	}
	return
}

func (o PrinterStateOptions) SetMaxDepth(v int64) {
	o[PrintStateOptionMaxDepth] = Int(v)
}

func (o PrinterStateOptions) Raw() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionRaw]; ok {
		var vb Bool
		vb, _ = vo.(Bool)
		v = bool(vb)
	}
	return
}

func (o PrinterStateOptions) SetRaw(v bool) {
	o[PrintStateOptionRaw] = Bool(v)
}

func (o PrinterStateOptions) WithRaw() {
	o.SetRaw(true)
}

func (o PrinterStateOptions) Zeros() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionZeros]; ok {
		var vb Bool
		vb, _ = vo.(Bool)
		v = bool(vb)
	}
	return
}

func (o PrinterStateOptions) SetZeros(v bool) {
	o[PrintStateOptionZeros] = Bool(v)
}

func (o PrinterStateOptions) WithZeros() PrinterStateOptions {
	o.SetZeros(true)
	return o
}

func (o PrinterStateOptions) Anonymous() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionAnonymous]; ok {
		var vb Bool
		vb, _ = vo.(Bool)
		v = bool(vb)
	}
	return
}

func (o PrinterStateOptions) SetAnonymous(v bool) {
	o[PrintStateOptionAnonymous] = Bool(v)
}

func (o PrinterStateOptions) WithAnonymous() PrinterStateOptions {
	o.SetAnonymous(true)
	return o
}

func (o PrinterStateOptions) Indexes() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionIndexes]; ok {
		v = !vo.IsFalsy()
	}
	return
}

func (o PrinterStateOptions) SetIndexes(v bool) {
	o[PrintStateOptionIndexes] = Bool(v)
}

func (o PrinterStateOptions) WithIndexes() PrinterStateOptions {
	o.SetIndexes(true)
	return o
}

func (o PrinterStateOptions) SortKeys() (v PrintStateOptionSortType, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionIndexes]; ok {
		var vi Int
		vi, _ = vo.(Int)
		v = PrintStateOptionSortType(vi)
	}
	return
}

func (o PrinterStateOptions) SetSortKeys(v PrintStateOptionSortType) {
	o[PrintStateOptionIndexes] = Int(v)
}

func (o PrinterStateOptions) SetTypesAsFullNames(v bool) {
	o[PrintStateOptionTypesAsFullNames] = Bool(v)
}

func (o PrinterStateOptions) TypesAsFullNames() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionTypesAsFullNames]; ok {
		v = !vo.IsFalsy()
	}
	return
}

func (o PrinterStateOptions) IsTypesAsFullNames() (v bool) {
	v, _ = o.TypesAsFullNames()
	return
}

func (o PrinterStateOptions) WithTypesAsFullNames() PrinterStateOptions {
	o.SetTypesAsFullNames(true)
	return o
}

func (o PrinterStateOptions) Repr() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionRepr]; ok {
		v = !vo.IsFalsy()
	}
	return
}

func (o PrinterStateOptions) SetRepr(v bool) {
	o[PrintStateOptionRepr] = Bool(v)
}

func (o PrinterStateOptions) WithRepr() PrinterStateOptions {
	o.SetRepr(true)
	return o
}

func (o PrinterStateOptions) BytesToHex() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionBytesToHex]; ok {
		var vb Bool
		vb, _ = vo.(Bool)
		v = bool(vb)
	}
	return
}

func (o PrinterStateOptions) IsBytesToHex() (is bool) {
	is, _ = o.BytesToHex()
	return
}

func (o PrinterStateOptions) SetBytesToHex(v bool) {
	o[PrintStateOptionBytesToHex] = Bool(v)
}

func (o PrinterStateOptions) WithBytesToHex() {
	o.SetBytesToHex(true)
}

func (o PrinterStateOptions) QuoteStr() (v bool, ok bool) {
	var vo Object
	if vo, ok = o[PrintStateOptionQuoteStr]; ok {
		var vb Bool
		vb, _ = vo.(Bool)
		v = bool(vb)
	}
	return
}

func (o PrinterStateOptions) IsQuoteStr() (is bool) {
	is, _ = o.QuoteStr()
	return
}

func (o PrinterStateOptions) SetQuoteStr(v bool) {
	o[PrintStateOptionQuoteStr] = Bool(v)
}

func (o PrinterStateOptions) WithQuoteStr() {
	o.SetQuoteStr(true)
}

func (o PrinterStateOptions) Backup(key string) (restore func()) {
	v, ok := o[key]
	return func() {
		if ok {
			o[key] = v
		} else {
			delete(o, key)
		}
	}
}

func (o PrinterStateOptions) WithBackup(key string, value Object) (restore func()) {
	old, ok := o[key]

	if value == nil && ok {
		delete(o, key)
	} else {
		o[key] = value
	}

	return func() {
		if ok {
			o[key] = old
		} else {
			delete(o, key)
		}
	}
}
