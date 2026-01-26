// YAPL Semantic Analyzer - AST Data Structures
// Represents the parsed AST from Pass 2

package main

import "fmt"

// Program represents the entire parsed program
type Program struct {
	SourceFile string
	Structs    []*StructDef
	Constants  []*ConstDef
	Globals    []*VarDef
	Functions  []*FuncDef
	AsmDecls   []string // File-level inline assembly declarations
}

// StructDef represents a struct definition
type StructDef struct {
	Name   string
	Fields []*FieldDef
	Size   int
	Align  int
}

// FieldDef represents a struct field
type FieldDef struct {
	Name   string
	Type   *Type
	Offset int
}

// ConstDef represents a constant definition
type ConstDef struct {
	Name      string
	Type      *Type
	Value     int64  // for scalar constants
	ArrayLen  int    // for const arrays: 0 if not array, -1 if inferred
	InitBytes []byte // for const array initializers (string data + null)
}

// VarDef represents a global or local variable
type VarDef struct {
	Name      string
	Type      *Type
	Offset    int    // for globals: data offset; for locals: stack offset
	ArrayLen  int    // 0 if not an array, -1 if inferred from initializer
	IsParam   bool   // true if function parameter
	RegHint   string // for params: R1, R2, R3, or empty for stack
	InitBytes []byte // for string/array initializers (includes null terminator)
}

// FuncDef represents a function definition
type FuncDef struct {
	Name       string
	ReturnType *Type
	Params     []*VarDef
	Locals     []*VarDef
	FrameSize  int
	Body       []Stmt
	Line       int
}

// Type represents a YAPL type
type Type struct {
	Kind     TypeKind
	BaseType BaseType   // for Kind == TypeBase
	Pointee  *Type      // for Kind == TypePointer
	ElemType *Type      // for Kind == TypeArray
	ArrayLen int        // for Kind == TypeArray
	Name     string     // for Kind == TypeStruct
}

type TypeKind int

const (
	TypeInvalid TypeKind = iota
	TypeVoid
	TypeBase
	TypePointer
	TypeArray
	TypeStruct
)

type BaseType int

const (
	BaseInvalid BaseType = iota
	BaseUint8   // byte
	BaseInt16
	BaseUint16
	BaseBlock32
	BaseBlock64
	BaseBlock128
)

// Predefined types
var (
	VoidType    = &Type{Kind: TypeVoid}
	Uint8Type   = &Type{Kind: TypeBase, BaseType: BaseUint8}
	Int16Type   = &Type{Kind: TypeBase, BaseType: BaseInt16}
	Uint16Type  = &Type{Kind: TypeBase, BaseType: BaseUint16}
	Block32Type = &Type{Kind: TypeBase, BaseType: BaseBlock32}
	Block64Type = &Type{Kind: TypeBase, BaseType: BaseBlock64}
	Block128Type = &Type{Kind: TypeBase, BaseType: BaseBlock128}
)

func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	switch t.Kind {
	case TypeVoid:
		return "VOID"
	case TypeBase:
		return t.BaseType.String()
	case TypePointer:
		return "PTR:" + t.Pointee.String()
	case TypeArray:
		return fmt.Sprintf("[%d]%s", t.ArrayLen, t.ElemType.String())
	case TypeStruct:
		return "STRUCT:" + t.Name
	default:
		return "<invalid>"
	}
}

func (b BaseType) String() string {
	switch b {
	case BaseUint8:
		return "UINT8"
	case BaseInt16:
		return "INT16"
	case BaseUint16:
		return "UINT16"
	case BaseBlock32:
		return "BLOCK32"
	case BaseBlock64:
		return "BLOCK64"
	case BaseBlock128:
		return "BLOCK128"
	default:
		return "<invalid>"
	}
}

func (t *Type) Size() int {
	switch t.Kind {
	case TypeVoid:
		return 0
	case TypeBase:
		return t.BaseType.Size()
	case TypePointer:
		return 2 // all pointers are 16-bit
	case TypeArray:
		return t.ElemType.Size() * t.ArrayLen
	case TypeStruct:
		return 0 // need to look up in struct table
	default:
		return 0
	}
}

func (b BaseType) Size() int {
	switch b {
	case BaseUint8:
		return 1
	case BaseInt16, BaseUint16:
		return 2
	case BaseBlock32:
		return 4
	case BaseBlock64:
		return 8
	case BaseBlock128:
		return 16
	default:
		return 0
	}
}

func (t *Type) IsIntegral() bool {
	return t.Kind == TypeBase && (t.BaseType == BaseUint8 || t.BaseType == BaseInt16 || t.BaseType == BaseUint16)
}

func (t *Type) IsSigned() bool {
	return t.Kind == TypeBase && t.BaseType == BaseInt16
}

func (t *Type) IsPointer() bool {
	return t.Kind == TypePointer
}

// Stmt represents a statement
type Stmt interface {
	stmtNode()
	GetLine() int
}

type baseStmt struct {
	Line int
}

func (s *baseStmt) stmtNode()    {}
func (s *baseStmt) GetLine() int { return s.Line }

// ExprStmt is an expression statement
type ExprStmt struct {
	baseStmt
	X Expr
}

// ReturnStmt is a return statement
type ReturnStmt struct {
	baseStmt
	Value Expr // nil for void return
}

// IfStmt is an if statement
type IfStmt struct {
	baseStmt
	Cond Expr
	Then []Stmt
	Else []Stmt // nil if no else
}

// WhileStmt is a while statement
type WhileStmt struct {
	baseStmt
	Cond Expr
	Body []Stmt
}

// ForStmt is a for statement
type ForStmt struct {
	baseStmt
	Init Expr   // nil if omitted
	Cond Expr   // nil if omitted
	Post Expr   // nil if omitted
	Body []Stmt
}

// GotoStmt is a goto statement
type GotoStmt struct {
	baseStmt
	Label string
}

// LabelStmt is a label
type LabelStmt struct {
	baseStmt
	Label string
}

// BreakStmt is a break statement
type BreakStmt struct {
	baseStmt
}

// ContinueStmt is a continue statement
type ContinueStmt struct {
	baseStmt
}

// AsmStmt is an inline assembly statement
type AsmStmt struct {
	baseStmt
	AsmText string // The raw assembly text
}

// Expr represents an expression
type Expr interface {
	exprNode()
	GetType() *Type
	SetType(*Type)
}

type baseExpr struct {
	ExprType *Type
}

func (e *baseExpr) exprNode()         {}
func (e *baseExpr) GetType() *Type    { return e.ExprType }
func (e *baseExpr) SetType(t *Type)   { e.ExprType = t }

// LiteralExpr is a literal value
type LiteralExpr struct {
	baseExpr
	IntVal int64
	StrVal string
	IsStr  bool
}

// IdentExpr is an identifier reference
type IdentExpr struct {
	baseExpr
	Name string
}

// BinaryExpr is a binary operation
type BinaryExpr struct {
	baseExpr
	Op    BinaryOp
	Left  Expr
	Right Expr
}

type BinaryOp int

const (
	OpAdd BinaryOp = iota
	OpSub
	OpMul
	OpDiv
	OpMod
	OpAnd
	OpOr
	OpXor
	OpShl
	OpShr
	OpEq
	OpNe
	OpLt
	OpLe
	OpGt
	OpGe
	OpLAnd
	OpLOr
)

func (op BinaryOp) String() string {
	switch op {
	case OpAdd:
		return "ADD"
	case OpSub:
		return "SUB"
	case OpMul:
		return "MUL"
	case OpDiv:
		return "DIV"
	case OpMod:
		return "MOD"
	case OpAnd:
		return "AND"
	case OpOr:
		return "OR"
	case OpXor:
		return "XOR"
	case OpShl:
		return "SHL"
	case OpShr:
		return "SHR"
	case OpEq:
		return "EQ"
	case OpNe:
		return "NE"
	case OpLt:
		return "LT"
	case OpLe:
		return "LE"
	case OpGt:
		return "GT"
	case OpGe:
		return "GE"
	case OpLAnd:
		return "LAND"
	case OpLOr:
		return "LOR"
	default:
		return "?"
	}
}

// UnaryExpr is a unary operation
type UnaryExpr struct {
	baseExpr
	Op      UnaryOp
	Operand Expr
}

type UnaryOp int

const (
	OpNeg UnaryOp = iota // -
	OpNot               // ~
	OpLNot              // !
	OpAddr              // &
	OpDeref             // @
)

// AssignExpr is an assignment
type AssignExpr struct {
	baseExpr
	LHS Expr
	RHS Expr
}

// CallExpr is a function call
type CallExpr struct {
	baseExpr
	Func string
	Args []Expr
}

// IndexExpr is array indexing
type IndexExpr struct {
	baseExpr
	Array Expr
	Index Expr
}

// FieldExpr is struct field access
type FieldExpr struct {
	baseExpr
	Object  Expr
	Field   string
	IsArrow bool // true for ->, false for .
}

// CastExpr is a type cast
type CastExpr struct {
	baseExpr
	Target  *Type
	Operand Expr
}

// SizeofExpr is sizeof(type)
type SizeofExpr struct {
	baseExpr
	Target *Type
}
