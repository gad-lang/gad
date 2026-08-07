package gad

// SourceFileType is the object type of SourceFileObject (gad.SourceFileObject):
// the parsed source returned by gad.parse / gad.parseFile alongside its
// statements. It exposes the raw source (indexable, sliceable, convertible to
// bytes) and the `path` and `type` attributes.
var SourceFileType = registerGadNamespaceType(BuiltinSourceFile, "SourceFileObject", (*SourceFileObject)(nil))

// SourceFileObject wraps a parsed source file: its path (or generated name), the
// selected SourceKind, the raw source bytes and the parsed statements. Indexing
// by position returns the byte at that offset as a char; slicing returns the
// byte range as bytes; the value converts to bytes and exposes the `path`,
// `type` and `stmts` attributes.
type SourceFileObject struct {
	Path   string
	Kind   SourceKind
	Source []byte
	Stmts  StmtsObject
}

var (
	_ Object         = (*SourceFileObject)(nil)
	_ IndexGetter    = (*SourceFileObject)(nil)
	_ LengthGetter   = (*SourceFileObject)(nil)
	_ Slicer         = (*SourceFileObject)(nil)
	_ BytesConverter = (*SourceFileObject)(nil)
)

func (*SourceFileObject) Type() ObjectType { return SourceFileType }

func (o *SourceFileObject) ToString() string { return string(o.Source) }

func (o *SourceFileObject) IsFalsy() bool { return len(o.Source) == 0 }

// Equal reports whether right is the same SourceFileObject value (path, kind and
// source content).
func (o *SourceFileObject) Equal(right Object) bool {
	r, ok := right.(*SourceFileObject)
	if !ok || r.Path != o.Path || r.Kind != o.Kind || len(r.Source) != len(o.Source) {
		return false
	}
	return string(r.Source) == string(o.Source)
}

// Length implements LengthGetter (len(sourceFile) is the source byte count).
func (o *SourceFileObject) Length() int { return len(o.Source) }

// ToBytes implements BytesConverter so bytes(sourceFile) yields the raw source.
func (o *SourceFileObject) ToBytes() (Bytes, error) {
	cp := make(Bytes, len(o.Source))
	copy(cp, o.Source)
	return cp, nil
}

// Slice implements Slicer: sourceFile[low:high] returns the byte range as bytes.
func (o *SourceFileObject) Slice(low, high int) Object {
	cp := make(Bytes, high-low)
	copy(cp, o.Source[low:high])
	return cp
}

// IndexGet returns the byte at an integer offset as a char, or the `path` /
// `type` attributes by name.
func (o *SourceFileObject) IndexGet(_ *VM, index Object) (Object, error) {
	switch v := index.(type) {
	case Int, Uint:
		i, err := stmtsIndex(index, len(o.Source))
		if err != nil {
			return nil, err
		}
		return Char(o.Source[i]), nil
	case Str, RawStr:
		switch v.ToString() {
		case "path":
			return Str(o.Path), nil
		case "type":
			return sourceKindEnumValue(o.Kind), nil
		case "stmts":
			return o.Stmts, nil
		}
		return nil, ErrInvalidIndex.NewError(v.ToString())
	}
	return nil, NewIndexTypeError("int|uint|str", index.Type().Name())
}

// sourceKindEnumValue returns the gad.SourceType member for a SourceKind.
func sourceKindEnumValue(k SourceKind) Object {
	name := "GAD"
	switch k {
	case SourceKindGadt:
		name = "TEMPLATE"
	case SourceKindGadx:
		name = "GADX"
	}
	if v, ok := SourceTypeEnum.Values[name]; ok {
		return v
	}
	return Int(k)
}
