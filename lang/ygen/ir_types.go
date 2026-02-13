// YAPL Code Generator - IR Type Definitions
// Data structures for representing parsed IR

package main

// IRProgram represents the complete parsed IR
type IRProgram struct {
	SourceFile  string
	IsBootstrap bool     // set by #pragma bootstrap
	AsmDecls    []string // File-level inline assembly
	Structs    []*IRStruct
	Constants  []*IRConst
	Globals    []*IRData
	Functions  []*IRFunction
}

// IRStruct represents a struct definition
type IRStruct struct {
	Name   string
	Size   int
	Align  int
	Fields []*IRField
}

// IRField represents a struct field
type IRField struct {
	Name   string
	Offset int
	Type   string
}

// IRConst represents a constant
type IRConst struct {
	Name       string
	Visibility string // PUBLIC or STATIC
	Type       string // UINT8, INT16, UINT16
	Value      int64
}

// IRData represents global data
type IRData struct {
	Name       string
	Visibility string // PUBLIC or STATIC
	Type       string // WORD, BYTE, BYTES n, etc.
	Size       int
	Init       string // optional initializer
}

// IRFunction represents a function
type IRFunction struct {
	Name       string
	Visibility string
	ReturnType string
	Params     []*IRParam
	Locals     []*IRLocal
	FrameSize  int
	Instrs     []*IRInstr
}

// IRParam represents a function parameter
type IRParam struct {
	Name  string
	Type  string
	Index int
}

// IRLocal represents a local variable
type IRLocal struct {
	Name   string
	Type   string
	Offset int
}

// IRInstr represents an IR instruction
type IRInstr struct {
	LineNum int
	Op      string   // CONST.W, LOAD.W, ADD.W, LABEL, etc.
	Dest    string   // destination virtual register or empty
	Args    []string // operands
	Label   string   // for LABEL instruction
	Target  string   // for JUMP instructions
}

// Instruction categories for easier handling
const (
	// Data movement
	OpConstW   = "CONST.W"
	OpConstB   = "CONST.B"
	OpLoadW    = "LOAD.W"
	OpLoadB    = "LOAD.B"
	OpLoadBU   = "LOAD.BU"
	OpStoreW   = "STORE.W"
	OpStoreB   = "STORE.B"
	OpAddr     = "ADDR"
	OpParam    = "PARAM"
	OpSetParam = "SETPARAM"
	OpCopy     = "COPY"

	// Arithmetic
	OpAddW = "ADD.W"
	OpSubW = "SUB.W"
	OpMulW = "MUL.W"
	OpDivS = "DIV.S"
	OpDivU = "DIV.U"
	OpModS = "MOD.S"
	OpModU = "MOD.U"
	OpNegW = "NEG.W"

	// Bitwise
	OpAndW = "AND.W"
	OpOrW  = "OR.W"
	OpXorW = "XOR.W"
	OpNotW = "NOT.W"
	OpShlW = "SHL.W"
	OpShrW = "SHR.W"
	OpSarW = "SAR.W"

	// Comparison
	OpEqW  = "EQ.W"
	OpNeW  = "NE.W"
	OpLtS  = "LT.S"
	OpLeS  = "LE.S"
	OpGtS  = "GT.S"
	OpGeS  = "GE.S"
	OpLtU  = "LT.U"
	OpLeU  = "LE.U"
	OpGtU  = "GT.U"
	OpGeU  = "GE.U"

	// Control flow
	OpLabel  = "LABEL"
	OpJump   = "JUMP"
	OpJumpZ  = "JUMPZ"
	OpJumpNZ = "JUMPNZ"

	// Function calls
	OpArg    = "ARG"
	OpCall   = "CALL"
	OpReturn = "RETURN"

	// Special
	OpAsm = "ASM"
)

// Register constants
const (
	R0 = 0 // Zero / LINK
	R1 = 1 // Arg0 / Return value / Caller-saved
	R2 = 2 // Arg1 / Caller-saved
	R3 = 3 // Arg2 / Caller-saved
	R4 = 4 // Callee-saved temp
	R5 = 5 // Callee-saved temp
	R6 = 6 // Callee-saved temp
	R7 = 7 // Stack pointer (SP)
)

// RegName returns the assembly name for a register
func RegName(r int) string {
	return [...]string{"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7"}[r]
}
