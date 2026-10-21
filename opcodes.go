// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// Opcode represents a single byte operation code.
type Opcode byte

func (o Opcode) String() string {
	if int(o) < len(OpcodeNames) {
		return OpcodeNames[o]
	}
	return ""
}

const (
	OpCallFlagVarArgs OpCallFlag = 1 << iota
	OpCallFlagNamedArgs
	OpCallFlagNamedArgsVar
)

type OpCallFlag byte

func (f OpCallFlag) Has(other OpCallFlag) bool {
	return (f & other) != 0
}

// List of opcodes
const (
	OpNoOp Opcode = iota
	OpConstant
	OpCall
	OpGetGlobal
	OpSetGlobal
	OpGetLocal
	OpSetLocal
	OpGetBuiltin
	OpBinary
	OpUnary
	OpSelfAssign
	OpEqual
	OpNotEqual
	OpJump
	OpJumpFalsy
	OpAndJump
	OpOrJump
	OpDict
	OpArray
	OpSliceIndex
	OpGetIndex
	OpSetIndex
	OpNil
	OpStdIn
	OpStdOut
	OpStdErr
	OpDotName
	OpDotFile
	OpIsMain
	OpNotIsMain
	OpModule
	OpGlobals
	OpPop
	OpGetFree
	OpSetFree
	OpGetLocalPtr
	OpGetFreePtr
	OpClosure
	OpIterInit
	OpIterNext
	OpIterNextElse
	OpIterKey
	OpIterValue
	OpLoadModule
	OpInitModule
	OpSetupTry
	OpSetupCatch
	OpSetupFinally
	OpThrow
	OpFinalizer
	OpReturn
	OpSetReturn
	OpSetReturnModule
	OpDefineLocal
	OpTrue
	OpFalse
	OpYes
	OpNo
	OpCallName
	OpJumpNil
	OpJumpNotNil
	OpKeyValueArray
	OpKeyValue
	OpCallee
	OpArgs
	OpNamedArgs
	OpTextWriter
	OpIsNil
	OpNotIsNil
	OpNamedParamsVar
	OpNamedParamValue
	OpComputedValue
	OpAddMethod
	OpExtendModule
	OpExtendModuleConst
	OpToRawStr
	OpMakeTypeUnion
	OpAddMethodOverride
	OpAssign
	OpEnv
	OpEnvGet
	OpEnvSet
	OpDelete
	OpSelfAssignN
	OpInterfaceBind
	OpAssignTransform
	OpPushSource
	OpPopSource
	OpFiles
	OpInterfaceExtends
	OpVarPtr
	OpAddrOfIndex
	OpMakeTypeUnionMeta
	// OpAssignTransformStrict is `obj :> type`: the transform into a class, an
	// enum value or an interface only (any other target is an error that points
	// to `::>`, which is OpAssignTransform). Appended so no opcode value shifts.
	OpAssignTransformStrict
	// OpJumpNotNo is `a ?! b`: when the top of the stack is not `no` it jumps
	// (keeping it), else it pops it and goes on to b. Appended so no opcode
	// value shifts.
	OpJumpNotNo
	// OpInterfaceDefaults binds the defaults of an interface's fields that are
	// no constant, computed where it is declared: the stack holds the
	// interface, then pairs of a field's index and its default. Appended so no
	// opcode value shifts.
	OpInterfaceDefaults
	// OpBroadcast is `a .op b`: op applied item by item, an array of the
	// results (Julia's broadcast); its flag 1 descends into nested arrays
	// (`a.op(b; nested)`). Appended so no opcode value shifts.
	OpBroadcast
)

// OpcodeNames are string representation of opcodes.
var OpcodeNames = [...]string{
	OpNoOp:                  "NOOP",
	OpConstant:              "CONSTANT",
	OpCall:                  "CALL",
	OpGetGlobal:             "GETGLOBAL",
	OpSetGlobal:             "SETGLOBAL",
	OpGetLocal:              "GETLOCAL",
	OpSetLocal:              "SETLOCAL",
	OpGetBuiltin:            "GETBUILTIN",
	OpBinary:                "BINARY",
	OpUnary:                 "UNARY",
	OpSelfAssign:            "SELFASSIGN",
	OpEqual:                 "EQUAL",
	OpNotEqual:              "NOTEQUAL",
	OpJump:                  "JUMP",
	OpJumpFalsy:             "JUMPFALSY",
	OpAndJump:               "ANDJUMP",
	OpOrJump:                "ORJUMP",
	OpDict:                  "DICT",
	OpArray:                 "ARRAY",
	OpSliceIndex:            "SLICEINDEX",
	OpGetIndex:              "GETINDEX",
	OpSetIndex:              "SETINDEX",
	OpNil:                   "NIL",
	OpStdIn:                 "STDIN",
	OpStdOut:                "STDOUT",
	OpStdErr:                "STDERR",
	OpDotName:               "DOTNAME",
	OpDotFile:               "DOTFILE",
	OpIsMain:                "ISMAIN",
	OpNotIsMain:             "NOTISMAIN",
	OpModule:                "MODULE",
	OpGlobals:               "GLOBALS",
	OpPop:                   "POP",
	OpGetFree:               "GETFREE",
	OpSetFree:               "SETFREE",
	OpGetLocalPtr:           "GETLOCALPTR",
	OpGetFreePtr:            "GETFREEPTR",
	OpClosure:               "CLOSURE",
	OpIterInit:              "ITERINIT",
	OpIterNext:              "ITERNEXT",
	OpIterNextElse:          "ITERNEXTELSE",
	OpIterKey:               "ITERKEY",
	OpIterValue:             "ITERVALUE",
	OpLoadModule:            "LOADMODULE",
	OpInitModule:            "INITMODULE",
	OpReturn:                "RETURN",
	OpSetReturn:             "SETRETURN",
	OpSetReturnModule:       "SETRETURNMODULE",
	OpSetupTry:              "SETUPTRY",
	OpSetupCatch:            "SETUPCATCH",
	OpSetupFinally:          "SETUPFINALLY",
	OpThrow:                 "THROW",
	OpFinalizer:             "FINALIZER",
	OpDefineLocal:           "DEFINELOCAL",
	OpTrue:                  "TRUE",
	OpFalse:                 "FALSE",
	OpYes:                   "YES",
	OpNo:                    "NO",
	OpCallName:              "CALLNAME",
	OpJumpNil:               "JUMPNIL",
	OpJumpNotNil:            "JUMPNOTNIL",
	OpKeyValueArray:         "KVARRAY",
	OpKeyValue:              "KV",
	OpCallee:                "CALLEE",
	OpArgs:                  "ARGS",
	OpNamedArgs:             "NAMEDARGS",
	OpIsNil:                 "ISNIL",
	OpNotIsNil:              "NOTISNIL",
	OpNamedParamsVar:        "NAMEDPARAMSVAR",
	OpNamedParamValue:       "NPARAMVALUE",
	OpComputedValue:         "COMPUTEDVALUE",
	OpAddMethod:             "ADDMETHOD",
	OpExtendModule:          "EXTENDMODULE",
	OpExtendModuleConst:     "EXTENDMODULECONST",
	OpToRawStr:              "TORAWSTR",
	OpMakeTypeUnion:         "MAKETYPEUNION",
	OpAddMethodOverride:     "ADDMETHODOVERRIDE",
	OpAssign:                "ASSIGN",
	OpEnv:                   "ENV",
	OpEnvGet:                "ENVGET",
	OpEnvSet:                "ENVSET",
	OpDelete:                "DELETE",
	OpSelfAssignN:           "SELFASSIGNN",
	OpInterfaceBind:         "INTERFACEBIND",
	OpAssignTransform:       "ASSIGNT",
	OpPushSource:            "PUSHSOURCE",
	OpPopSource:             "POPSOURCE",
	OpFiles:                 "FILES",
	OpInterfaceExtends:      "INTERFACEEXTENDS",
	OpVarPtr:                "VARPTR",
	OpAddrOfIndex:           "ADDROFINDEX",
	OpMakeTypeUnionMeta:     "MAKETYPEUNIONMETA",
	OpAssignTransformStrict: "ASSIGNTS",
	OpJumpNotNo:             "JUMPNOTNO",
	OpInterfaceDefaults:     "INTERFACEDEFAULTS",
	OpBroadcast:             "BROADCAST",
}

// OpcodeOperands is the number of operands.
var OpcodeOperands = [...][]int{
	OpNoOp:                  {},
	OpConstant:              {2},    // constant index
	OpCall:                  {1, 1}, // number of arguments, named arguments
	OpGetGlobal:             {2},    // constant index
	OpSetGlobal:             {2},    // constant index
	OpGetLocal:              {1},    // local variable index
	OpSetLocal:              {1},    // local variable index
	OpGetBuiltin:            {2},    // builtin index
	OpBinary:                {1},    // operator
	OpUnary:                 {1},    // operator
	OpSelfAssign:            {1},    // operator
	OpEqual:                 {},
	OpNotEqual:              {},
	OpIsNil:                 {},
	OpNotIsNil:              {},
	OpJump:                  {2}, // position
	OpJumpFalsy:             {2}, // position
	OpAndJump:               {2}, // position
	OpOrJump:                {2}, // position
	OpDict:                  {2}, // number of keys and values
	OpArray:                 {2}, // number of items
	OpSliceIndex:            {},
	OpGetIndex:              {1}, // number of selectors
	OpSetIndex:              {},
	OpNil:                   {},
	OpStdIn:                 {},
	OpStdOut:                {},
	OpStdErr:                {},
	OpDotName:               {},
	OpDotFile:               {},
	OpIsMain:                {},
	OpNotIsMain:             {},
	OpModule:                {},
	OpGlobals:               {},
	OpPop:                   {},
	OpGetFree:               {1},    // index
	OpSetFree:               {1},    // index
	OpGetLocalPtr:           {1},    // index
	OpGetFreePtr:            {1},    // index
	OpClosure:               {2, 1}, // constant index, item count
	OpIterInit:              {},
	OpIterNext:              {},
	OpIterNextElse:          {2, 2}, // true pos, false pos
	OpIterKey:               {},
	OpIterValue:             {},
	OpLoadModule:            {2},    // store index
	OpInitModule:            {1, 1}, // number of arguments, named arguments
	OpSetReturn:             {1},    // local variable index
	OpSetReturnModule:       {},
	OpReturn:                {1}, // number of items (0 or 1)
	OpSetupTry:              {2, 2},
	OpSetupCatch:            {},
	OpSetupFinally:          {},
	OpThrow:                 {1}, // 0:re-throw (system), 1:throw <expression>
	OpFinalizer:             {1}, // up to error handler index
	OpDefineLocal:           {1},
	OpTrue:                  {},
	OpFalse:                 {},
	OpYes:                   {},
	OpNo:                    {},
	OpCallName:              {1, 1}, // number of arguments, flags
	OpJumpNil:               {2},    // position
	OpJumpNotNil:            {2},    // position
	OpKeyValueArray:         {2},    // number of keys and values
	OpCallee:                {},
	OpArgs:                  {},
	OpNamedArgs:             {},
	OpKeyValue:              {1}, // 0: whitout value, 1: with value
	OpNamedParamsVar:        {},
	OpNamedParamValue:       {},
	OpComputedValue:         {},
	OpAddMethod:             {1, 1}, // 0: number of selectors, 1: number of funcs
	OpExtendModule:          {},
	OpExtendModuleConst:     {},
	OpToRawStr:              {},
	OpMakeTypeUnion:         {1},    // number of member types on the stack
	OpAddMethodOverride:     {1, 1}, // 0: number of selectors, 1: number of funcs
	OpAssign:                {},     // obj :: type -> obj (or throw if not assignable)
	OpEnv:                   {},     // push the VM env object
	OpEnvGet:                {},     // pop key (str), push env[key]
	OpEnvSet:                {2},    // constant index of key; pop value, env[key]=value
	OpDelete:                {},     // pop keys array and `this`; delete this[k] for each k
	OpSelfAssignN:           {1, 1}, // operator, n; fused spread self-assign over n stack items
	OpInterfaceBind:         {2},    // n; pop n context-func values + the interface, push the bound interface
	OpAssignTransform:       {},     // obj ::> type -> converted obj (or throw if not convertible)
	OpPushSource:            {2},    // constant index of the source name (a Str); push it on the source-name stack
	OpPopSource:             {},     // pop the source-name stack
	OpFiles:                 {},     // push the source-name stack as a read-only Array
	OpInterfaceExtends:      {2},    // n; pop n parent values (interfaces or arrays of them) + the interface, push it extending them
	OpVarPtr:                {},     // pop a variable cell (from OpGetLocalPtr/OpGetFreePtr), push a `ptr` to it (`&x`)
	OpAddrOfIndex:           {},     // pop key and target, push a `ptr` to target[key] (`&a.b`, `&a[i]`)
	OpMakeTypeUnionMeta:     {1},    // number of member types on the stack, under the metadata
	OpAssignTransformStrict: {},     // obj :> type -> obj transformed into a class/enum value/interface (or throw)
	OpJumpNotNo:             {2},    // position
	OpInterfaceDefaults:     {2},    // number of pairs (field index, default) above the interface
	OpBroadcast:             {1, 1}, // operator, nested (0 or 1)
}

// ReadOperands reads operands from the bytecode. Given operands slice is used to
// fill operands and is returned to allocate less.
func ReadOperands(numOperands []int, ins []byte, operands []int) ([]int, int) {
	operands = operands[:0]
	var offset int
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(ins[offset+1])|int(ins[offset])<<8)
		}
		offset += width
	}
	return operands, offset
}
