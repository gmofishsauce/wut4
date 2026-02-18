// YAPL Parser - Abstract Syntax Tree
// AST node definitions for YAPL

package main

// SourceLoc represents a location in the source code
type SourceLoc struct {
	File string
	Line int
}

// Program is the root of the AST
type Program struct {
	Decls       []Decl
	IsBootstrap bool // set by #pragma bootstrap
}

// Decl is the interface for all declarations
type Decl interface {
	declNode()
	GetLoc() SourceLoc
}

// Stmt is the interface for all statements
type Stmt interface {
	stmtNode()
	GetLoc() SourceLoc
}

// Expr is the interface for all expressions
type Expr interface {
	exprNode()
	GetLoc() SourceLoc
	GetType() *Type
	SetType(*Type)
}

// ============================================================
// Declarations
// ============================================================

// ConstDecl represents a constant declaration
type ConstDecl struct {
	Name      string
	ConstType *Type
	Value     int64     // scalar value (for non-array constants)
	ArrayLen  int       // 0 if not array, -1 if inferred from initializer
	Init      Expr      // for array initializers (string literal)
	Loc       SourceLoc
}

func (d *ConstDecl) declNode()         {}
func (d *ConstDecl) GetLoc() SourceLoc { return d.Loc }

// VarDecl represents a variable declaration
type VarDecl struct {
	Name     string
	VarType  *Type
	ArrayLen int  // 0 if not array
	Init     Expr // nil if no initializer
	Loc      SourceLoc
}

func (d *VarDecl) declNode()         {}
func (d *VarDecl) GetLoc() SourceLoc { return d.Loc }

// FuncDecl represents a function declaration
type FuncDecl struct {
	Name       string
	ReturnType *Type
	Params     []*Param
	Locals     []LocalDecl // var and const declarations
	Body       []FuncStmt  // statements and labels
	Loc        SourceLoc
}

func (d *FuncDecl) declNode()         {}
func (d *FuncDecl) GetLoc() SourceLoc { return d.Loc }

// Param represents a function parameter
type Param struct {
	Name      string
	ParamType *Type
	Loc       SourceLoc
}

// LocalDecl is a local declaration (var or const)
type LocalDecl interface {
	localDeclNode()
	GetLoc() SourceLoc
}

func (d *ConstDecl) localDeclNode() {}
func (d *VarDecl) localDeclNode()   {}

// FuncStmt is a statement or label at function body level
type FuncStmt interface {
	funcStmtNode()
	GetLoc() SourceLoc
}

// StructDecl represents a struct declaration
type StructDecl struct {
	Name   string
	Fields []*FieldDecl
	Size   int // computed during analysis
	Align  int // computed during analysis
	Loc    SourceLoc
}

func (d *StructDecl) declNode()         {}
func (d *StructDecl) GetLoc() SourceLoc { return d.Loc }

// AsmDecl represents an inline assembly declaration at file level
type AsmDecl struct {
	AsmText string    // The raw assembly text (without quotes)
	Loc     SourceLoc
}

func (d *AsmDecl) declNode()         {}
func (d *AsmDecl) GetLoc() SourceLoc { return d.Loc }

// FieldDecl represents a struct field
type FieldDecl struct {
	Name      string
	FieldType *Type
	ArrayLen  int // 0 if not array
	Offset    int // computed during analysis
	Loc       SourceLoc
}

// ============================================================
// Statements
// ============================================================

// ExprStmt represents an expression statement
type ExprStmt struct {
	X   Expr // nil for empty statement ";"
	Loc SourceLoc
}

func (s *ExprStmt) stmtNode()         {}
func (s *ExprStmt) funcStmtNode()     {}
func (s *ExprStmt) GetLoc() SourceLoc { return s.Loc }

// Block represents a block of statements
type Block struct {
	Stmts []Stmt
	Loc   SourceLoc
}

func (s *Block) stmtNode()         {}
func (s *Block) funcStmtNode()     {}
func (s *Block) GetLoc() SourceLoc { return s.Loc }

// IfStmt represents an if statement
type IfStmt struct {
	Cond Expr
	Then Stmt
	Else Stmt // nil if no else clause
	Loc  SourceLoc
}

func (s *IfStmt) stmtNode()         {}
func (s *IfStmt) funcStmtNode()     {}
func (s *IfStmt) GetLoc() SourceLoc { return s.Loc }

// WhileStmt represents a while statement
type WhileStmt struct {
	Cond Expr
	Body Stmt
	Loc  SourceLoc
}

func (s *WhileStmt) stmtNode()         {}
func (s *WhileStmt) funcStmtNode()     {}
func (s *WhileStmt) GetLoc() SourceLoc { return s.Loc }

// ForStmt represents a for statement
type ForStmt struct {
	Init Expr // nil if omitted
	Cond Expr // nil if omitted
	Post Expr // nil if omitted
	Body Stmt
	Loc  SourceLoc
}

func (s *ForStmt) stmtNode()         {}
func (s *ForStmt) funcStmtNode()     {}
func (s *ForStmt) GetLoc() SourceLoc { return s.Loc }

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Value Expr // nil for void return
	Loc   SourceLoc
}

func (s *ReturnStmt) stmtNode()         {}
func (s *ReturnStmt) funcStmtNode()     {}
func (s *ReturnStmt) GetLoc() SourceLoc { return s.Loc }

// BreakStmt represents a break statement
type BreakStmt struct {
	Loc SourceLoc
}

func (s *BreakStmt) stmtNode()         {}
func (s *BreakStmt) funcStmtNode()     {}
func (s *BreakStmt) GetLoc() SourceLoc { return s.Loc }

// ContinueStmt represents a continue statement
type ContinueStmt struct {
	Loc SourceLoc
}

func (s *ContinueStmt) stmtNode()         {}
func (s *ContinueStmt) funcStmtNode()     {}
func (s *ContinueStmt) GetLoc() SourceLoc { return s.Loc }

// GotoStmt represents a goto statement
type GotoStmt struct {
	Label string
	Loc   SourceLoc
}

func (s *GotoStmt) stmtNode()         {}
func (s *GotoStmt) funcStmtNode()     {}
func (s *GotoStmt) GetLoc() SourceLoc { return s.Loc }

// LabelStmt represents a label (only at function body level)
type LabelStmt struct {
	Label string
	Loc   SourceLoc
}

func (s *LabelStmt) funcStmtNode()     {} // LabelStmt is FuncStmt but not Stmt
func (s *LabelStmt) GetLoc() SourceLoc { return s.Loc }

// AsmStmt represents an inline assembly statement within a function
type AsmStmt struct {
	AsmText string    // The raw assembly text (without quotes)
	Loc     SourceLoc
}

func (s *AsmStmt) stmtNode()         {}
func (s *AsmStmt) funcStmtNode()     {}
func (s *AsmStmt) GetLoc() SourceLoc { return s.Loc }

// ============================================================
// Expressions
// ============================================================

// baseExpr provides common fields for expressions
type baseExpr struct {
	ExprType *Type
	Loc      SourceLoc
}

func (e *baseExpr) GetLoc() SourceLoc  { return e.Loc }
func (e *baseExpr) GetType() *Type     { return e.ExprType }
func (e *baseExpr) SetType(t *Type)    { e.ExprType = t }

// BinaryOp represents binary operators
type BinaryOp int

const (
	OpInvalid BinaryOp = iota
	// Arithmetic
	OpAdd // +
	OpSub // -
	OpMul // *
	OpDiv // /
	OpMod // %
	// Bitwise
	OpAnd // &
	OpOr  // |
	OpXor // ^
	OpShl // <<
	OpShr // >>
	// Logical
	OpLAnd // &&
	OpLOr  // ||
	// Comparison
	OpEq // ==
	OpNe // !=
	OpLt // <
	OpGt // >
	OpLe // <=
	OpGe // >=
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
	case OpLAnd:
		return "LAND"
	case OpLOr:
		return "LOR"
	case OpEq:
		return "EQ"
	case OpNe:
		return "NE"
	case OpLt:
		return "LT"
	case OpGt:
		return "GT"
	case OpLe:
		return "LE"
	case OpGe:
		return "GE"
	default:
		return "INVALID"
	}
}

// UnaryOp represents unary operators
type UnaryOp int

const (
	UnaryInvalid UnaryOp = iota
	UnaryNeg             // -
	UnaryNot             // ~ (bitwise)
	UnaryLNot            // ! (logical)
	UnaryDeref           // @ (dereference)
	UnaryAddr            // & (address-of)
	UnarySizeof          // sizeof expr
)

func (op UnaryOp) String() string {
	switch op {
	case UnaryNeg:
		return "NEG"
	case UnaryNot:
		return "NOT"
	case UnaryLNot:
		return "LNOT"
	case UnaryDeref:
		return "DEREF"
	case UnaryAddr:
		return "ADDR"
	case UnarySizeof:
		return "SIZEOF"
	default:
		return "INVALID"
	}
}

// BinaryExpr represents a binary expression
type BinaryExpr struct {
	baseExpr
	Op    BinaryOp
	Left  Expr
	Right Expr
}

func (e *BinaryExpr) exprNode() {}

// AssignExpr represents an assignment expression
type AssignExpr struct {
	baseExpr
	LHS Expr
	RHS Expr
}

func (e *AssignExpr) exprNode() {}

// UnaryExpr represents a unary expression
type UnaryExpr struct {
	baseExpr
	Op      UnaryOp
	Operand Expr
}

func (e *UnaryExpr) exprNode() {}

// CastExpr represents a type cast expression
type CastExpr struct {
	baseExpr
	TargetType *Type
	Operand    Expr
}

func (e *CastExpr) exprNode() {}

// CallExpr represents a function call expression
type CallExpr struct {
	baseExpr
	Func Expr
	Args []Expr
}

func (e *CallExpr) exprNode() {}

// IndexExpr represents an array subscript expression
type IndexExpr struct {
	baseExpr
	Array Expr
	Index Expr
}

func (e *IndexExpr) exprNode() {}

// FieldExpr represents field access (. or ->)
type FieldExpr struct {
	baseExpr
	Object  Expr
	Field   string
	IsArrow bool // true for ->, false for .
}

func (e *FieldExpr) exprNode() {}

// IdentExpr represents an identifier expression
type IdentExpr struct {
	baseExpr
	Name string
	// Resolved during semantic analysis - not set by parser
}

func (e *IdentExpr) exprNode() {}

// LitKind represents literal kinds
type LitKind int

const (
	LitInt LitKind = iota
	LitString
)

// LiteralExpr represents a literal expression
type LiteralExpr struct {
	baseExpr
	Kind   LitKind
	IntVal int64
	StrVal string
}

func (e *LiteralExpr) exprNode() {}

// SizeofTypeExpr represents sizeof(type)
type SizeofTypeExpr struct {
	baseExpr
	TargetType *Type
}

func (e *SizeofTypeExpr) exprNode() {}

// ArrayInitExpr represents a numeric array initializer: { expr, expr, ... }
type ArrayInitExpr struct {
	baseExpr
	Elems []Expr
}

func (e *ArrayInitExpr) exprNode() {}
