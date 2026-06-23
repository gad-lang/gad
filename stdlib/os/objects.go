package os

import (
	"strings"
	"syscall"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/token"
)

const (
	// Exactly one of ORo, OWo, or ORW must be specified.
	ORo FileFlag = syscall.O_RDONLY // open the file read-only.
	OWo FileFlag = syscall.O_WRONLY // open the file write-only.
	ORW FileFlag = syscall.O_RDWR   // open the file read-write.
	// The remaining values may be or'ed in to control behavior.
	OAppend      FileFlag = syscall.O_APPEND // append data to the file when writing.
	OCreate      FileFlag = syscall.O_CREAT  // create a new file if none exists.
	OIfNotExists FileFlag = syscall.O_EXCL   // used with OCreate, file must not exist.
	OSync        FileFlag = syscall.O_SYNC   // open for synchronous I/O.
	OTrunc       FileFlag = syscall.O_TRUNC  // truncate regular writable file when opened.
)

type FileFlag uint64

func (f FileFlag) IsFalsy() bool {
	return f == 0
}

func (f FileFlag) Type() gad.ObjectType {
	return TFileFlag
}

func (f FileFlag) ToString() string {
	return f.String()
}

func (f FileFlag) Equal(right gad.Object) bool {
	if r, ok := right.(FileFlag); ok {
		return f == r
	}
	return false
}

func (f *FileFlag) Set(flag FileFlag) *FileFlag    { *f = *f | flag; return f }
func (f *FileFlag) Clear(flag FileFlag) *FileFlag  { *f = *f &^ flag; return f }
func (f *FileFlag) Toggle(flag FileFlag) *FileFlag { *f = *f ^ flag; return f }
func (f FileFlag) Has(flag FileFlag) bool          { return f&flag != 0 }
func (f FileFlag) String() string {
	var s []string
	if f.Has(ORo) {
		s = append(s, "ro")
	}
	if f.Has(OWo) {
		s = append(s, "wo")
	}
	if f.Has(ORW) {
		s = append(s, "rw")
	}
	if f.Has(OAppend) {
		s = append(s, "append")
	}
	if f.Has(OCreate) {
		s = append(s, "create")
	}
	if f.Has(OIfNotExists) {
		s = append(s, "if_not_exists")
	}
	if f.Has(OSync) {
		s = append(s, "sync")
	}
	if f.Has(OTrunc) {
		s = append(s, "trunc")
	}
	return strings.Join(s, "|")
}

func (f *FileFlag) Parse(str string) {
	for _, s := range strings.Split(str, "|") {
		if m := FileModeByName[s]; m > 0 {
			f.Set(m)
		}
	}
}

// The per-operator methods delegate to binOp, which forwards to the integer
// bit operations and re-wraps the result as a FileFlag.
func (f FileFlag) BinOpAdd(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Add, right)
}
func (f FileFlag) BinOpSub(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Sub, right)
}
func (f FileFlag) BinOpMul(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Mul, right)
}
func (f FileFlag) BinOpQuo(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Quo, right)
}
func (f FileFlag) BinOpRem(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Rem, right)
}
func (f FileFlag) BinOpAnd(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.And, right)
}
func (f FileFlag) BinOpOr(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Or, right)
}
func (f FileFlag) BinOpXor(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Xor, right)
}
func (f FileFlag) BinOpAndNot(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.AndNot, right)
}
func (f FileFlag) BinOpShl(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Shl, right)
}
func (f FileFlag) BinOpShr(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Shr, right)
}
func (f FileFlag) BinOpLess(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Less, right)
}
func (f FileFlag) BinOpLessEq(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.LessEq, right)
}
func (f FileFlag) BinOpGreater(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.Greater, right)
}
func (f FileFlag) BinOpGreaterEq(vm *gad.VM, right gad.Object) (gad.Object, error) {
	return f.binOp(vm, token.GreaterEq, right)
}

func (f FileFlag) binOp(vm *gad.VM, tok token.Token, right gad.Object) (ret gad.Object, err error) {
try:
	switch v := right.(type) {
	case gad.Int:
		right = FileFlag(v)
		goto try
	case gad.Uint:
		right = FileFlag(v)
		goto try
	case FileFlag:
		if ret, err = gad.BinaryOp(vm, tok, gad.Int(f), gad.Int(right.(FileFlag))); err == nil {
			if r2, ok := ret.(gad.Int); ok {
				ret = FileFlag(r2)
			}
		}
		return
	default:
		return nil, gad.NewOperandTypeError(
			tok.String(),
			f.Type().Name(),
			right.Type().Name(),
		)
	}
}

var FileModeByName = map[string]FileFlag{
	"ro":            ORo,
	"wo":            OWo,
	"rw":            ORW,
	"append":        OAppend,
	"create":        OCreate,
	"if_not_exists": OIfNotExists,
	"sync":          OSync,
	"trunc":         OTrunc,
}

var TFileFlag = gad.NewType("FileFlag").
	WithConstructor(&gad.Function{
		Value: NewFileMode,
	}).
	WithStatic(gad.Dict{
		"RO":          ORo,
		"WO":          OWo,
		"RW":          ORW,
		"Append":      OAppend,
		"Create":      OCreate,
		"IfNotExists": OIfNotExists,
		"Sync":        OSync,
		"Trunc":       OTrunc,
	})
