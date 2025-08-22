package gad

import (
	"fmt"
	"io"

	"github.com/gad-lang/gad/token"
)

// Falser represents an Falser object.
type Falser interface {
	// IsFalsy returns true if value is falsy otherwise false.
	IsFalsy() bool
}

// Object represents an object in the VM.
type Object interface {
	Falser

	// OpDotName should return the name of the type.
	Type() ObjectType

	// ToString should return a string of the type's value.
	ToString() string

	// Equal checks equality of objects.
	Equal(right Object) bool
}

type Representer interface {
	Repr(vm *VM) (string, error)
}

type ObjectRepresenter interface {
	Object
	Representer
}

type ObjectType interface {
	Object
	CallerObject
	fmt.Stringer
	Name() string
	Getters() Dict
	Setters() Dict
	Methods() Dict
	Fields() Dict
	New(vm *VM, fields Dict) (Object, error)
	IsChildOf(t ObjectType) bool
}

type Objector interface {
	Object
	Fields() Dict
}

// Copier wraps the Copy method to create a single copy of the object.
type Copier interface {
	Object
	Copy() Object
}

// DeepCopier wraps the Copy method to create a deep copy of the object.
type DeepCopier interface {
	Object
	DeepCopy(vm *VM) (Object, error)
}

// IndexDeleter wraps the IndexDelete method to delete an index of an object.
type IndexDeleter interface {
	Object
	IndexDelete(vm *VM, key Object) error
}

// IndexGetter wraps the IndexGet method to get index value.
type IndexGetter interface {
	Object
	// IndexGet should take an index Object and return a result Object or an
	// error for indexable objects. Indexable is an object that can take an
	// index and return an object. Returned error stops VM execution if not
	// handled with an error handler and VM.Run returns the same error as
	// wrapped. If Object is not indexable, ErrNotIndexable should be returned
	// as error.
	IndexGet(vm *VM, index Object) (value Object, err error)
}

// IndexSetter wraps the IndexSet method to set index value.
type IndexSetter interface {
	Object
	// IndexSet should take an index Object and a value Object for index
	// assignable objects. Index assignable is an object that can take an index
	// and a value on the left-hand side of the assignment statement. If Object
	// is not index assignable, ErrNotIndexAssignable should be returned as
	// error. Returned error stops VM execution if not handled with an error
	// handler and VM.Run returns the same error as wrapped.
	IndexSet(vm *VM, index, value Object) error
}

type IndexGetSetter interface {
	IndexGetter
	IndexSetter
}

type Indexer interface {
	IndexGetter
	IndexSetter
	IndexDeleter
}

// LengthGetter wraps the Len method to get the number of elements of an object.
type LengthGetter interface {
	Object
	Length() int
}

// CallerObject is an interface for objects that can be called with Call
// method.
type CallerObject interface {
	Object
	Call(c Call) (Object, error)
}

type CallerObjectWithStaticMethods interface {
	GetGoMethods() []*Caller
}

// CallerObjectWithParamTypes is an interface for objects that can be called with Call
// method with parameters with types.
type CallerObjectWithParamTypes interface {
	CallerObject
	ParamTypes(vm *VM) (MultipleObjectTypes, error)
}

// CanCallerObject is an interface for objects that can be objects implements
// this CallerObject interface.
// Note if CallerObject implements this interface, CanCall() is called for check
// if object is callable.
type CanCallerObject interface {
	CallerObject
	// CanCall returns true if type can be called with Call() method.
	// VM returns an error if one tries to call a noncallable object.
	CanCall() bool
}

type CanCallerObjectTypesValidation interface {
	CallerObject
	ValidateParamTypes(vm *VM, args Args) (err error)
	CanValidateParamTypes() bool
}

type CanCallerObjectMethodsEnabler interface {
	CallerObject
	MethodsDisabled() bool
}

type MethodCaller interface {
	CallerObject

	// AddCallerMethod add caller method from argument types.
	// the argTypes param is a list of supported types for arguments.
	//
	// Examples:
	//  - fn(str, decimal) => MultipleObjectTypes{{TStr},{TDecimal}}
	//  - fn(str|int, decimal) => MultipleObjectTypes{{TStr,Int},{TDecimal}}
	AddCallerMethod(vm *VM, argTypes MultipleObjectTypes, handler CallerObject, override bool) error
	CallerMethods() *MethodArgType
	// CallerMethodWithValidationCheckOfArgs return a method and validation check flag from args.
	// In same cases this method is most fast then `MethodWithValidationCheckOfArgTypes`
	CallerMethodWithValidationCheckOfArgs(args Args) (method CallerObject, validationCheck bool)
	// CallerMethodWithValidationCheckOfArgsTypes return a method from knowed args types with validation check flag
	CallerMethodWithValidationCheckOfArgsTypes(types []ObjectType) (method CallerObject, validationCheck bool)
	// CallerMethodOfArgs return a method from arguments types whitout validation check flag.
	CallerMethodOfArgs(args Args) (method CallerObject)
	// CallerMethodOfArgsTypes return a method from arguments types whitout validation check flag.
	CallerMethodOfArgsTypes(types []ObjectType) (method CallerObject)
	HasCallerMethods() bool
	// Caller return the caller without methods
	Caller() CallerObject
}

// NameCallerObject is an interface for objects that can be called with CallName
// method to call a method of an object. Objects implementing this interface can
// reduce allocations by not creating a callable object for each method call.
type NameCallerObject interface {
	Object
	CallName(name string, c Call) (Object, error)
}

type ToArrayAppenderObject interface {
	Object
	AppendToArray(arr Array) Array
}

type ItemsGetterCallback func(i int, item *KeyValue) (err error)

// ItemsGetter is an interface for returns pairs of fields or keys with same values.
type ItemsGetter interface {
	Object
	Items(vm *VM, cb ItemsGetterCallback) (err error)
}

// KeysGetter is an interface for returns keys or fields names.
type KeysGetter interface {
	Object
	Keys() (arr Array)
}

// ValuesGetter is an interface for returns values.
type ValuesGetter interface {
	Object
	Values() (arr Array)
}

// Sorter is an interface for return sorted values.
type Sorter interface {
	Object

	// Sort sorts object. if `update`, sort self and return then, other else sorts a self copy object.
	Sort(vm *VM, less CallerObject) (Object, error)
}

// ReverseSorter is an interface for return reverse sorted values.
type ReverseSorter interface {
	Object

	// SortReverse sorts object reversely. if `update`, sort self and return then, other else sorts a self copy object.
	SortReverse(vm *VM) (Object, error)
}

type Iterabler interface {
	Object

	// Iterate should return an Iterator for the type.
	Iterate(vm *VM, na *NamedArgs) Iterator
}

type CanIterabler interface {
	Iterabler
	// CanIterate should return whether the Object can be Iterated.
	CanIterate() bool
}

type Filterabler interface {
	Object
	Filter(vm *VM, args Array, handler VMCaller) (Object, error)
}

type CanFilterabler interface {
	CanFilter() bool
}

type Mapabler interface {
	Object
	// Map map object.
	// If update, update self object.
	// If len(args) is 1, must a value, otherwise value and key
	Map(c Call, update bool, keyValue Array, handler VMCaller) (Object, error)
}

type CanMapeabler interface {
	CanMap() bool
}

type Reducer interface {
	Object
	Reduce(vm *VM, initialValue Object, args Array, handler VMCaller) (Object, error)
}

type CanReducer interface {
	CanReduce() bool
}

type Slicer interface {
	LengthGetter
	Slice(low, high int) Object
}

type ToIterfaceConverter interface {
	ToInterface() any
}

type ToIterfaceVMConverter interface {
	ToInterface(*VM) any
}

type Niler interface {
	Object
	IsNil() bool
}

type Appender interface {
	Object
	AppendObjects(vm *VM, arr ...Object) (Object, error)
}

type Adder interface {
	Object
	Append(vm *VM, arr ...Object) (err error)
}

// BytesConverter is to bytes converter
type BytesConverter interface {
	Object
	ToBytes() (Bytes, error)
}

type UserDataStorage interface {
	Object
	UserData() Indexer
}

type BinaryOperatorHandler interface {
	// BinaryOp handles +,-,*,/,%,<<,>>,<=,>=,<,> operators.
	// Returned error stops VM execution if not handled with an error handler
	// and VM.Run returns the same error as wrapped.
	BinaryOp(vm *VM, tok token.Token, right Object) (Object, error)
}

type Writer interface {
	Object
	io.Writer
	GoWriter() io.Writer
}

type Reader interface {
	Object
	io.Reader
	GoReader() io.Reader
}

type ReadWriter interface {
	Writer
	Reader
}

type ToReaderConverter interface {
	Reader() Reader
}

type ToWriterConverter interface {
	Writer() Writer
}

type CanCloser interface {
	CanClose() bool
}

type IterationDoner interface {
	IterationDone(vm *VM) error
}

type CanIterationDoner interface {
	CanIterationDone() bool
}
