package main

/* Token types */
const (
	TOK_EOF = iota
	TOK_LABEL
	TOK_IDENT
	TOK_NUMBER
	TOK_STRING
	TOK_COMMA
	TOK_COLON
	TOK_SEMICOLON
	TOK_PLUS
	TOK_MINUS
	TOK_STAR
	TOK_SLASH
	TOK_LPAREN
	TOK_RPAREN
	TOK_NEWLINE
)

/* Token structure */
type Token struct {
	typ    int
	value  string
	intval int
	line   int
	col    int
}

/* Assembler state */
type Assembler struct {
	/* Source file info */
	filename string
	lines    []string
	lineNum  int

	/* Current position */
	codeDollar int /* Current position in code segment */
	dataDollar int /* Current position in data segment */
	inCodeSeg  int /* 1 if in code segment, 0 if in data segment */

	/* Output buffers */
	codeBytes []byte /* Code segment bytes */
	dataBytes []byte /* Data segment bytes */

	/* Symbol tables */
	labels  map[string]int /* Label -> address */
	symbols map[string]int /* Symbols from .set directive */

	/* Forward references */
	fixups []Fixup /* List of forward references to resolve */

	/* Error tracking */
	errors []string
}

/* Fixup for forward references */
type Fixup struct {
	addr     int    /* Address to fix up */
	label    string /* Label being referenced */
	line     int    /* Line number for error reporting */
	isInCode int    /* 1 if in code segment, 0 if in data segment */
}

/* Instruction definition */
type InstrDef struct {
	name    string
	opcode  uint16
	format  int  /* Instruction format */
	numOps  int  /* Number of operands */
	hasImm  int  /* Has immediate value */
	immBits int  /* Number of immediate bits */
}

/* Instruction formats */
const (
	FMT_BASE = iota /* Base instructions: LDW, LDB, STW, STB, ADI */
	FMT_LUI         /* Load upper immediate */
	FMT_BRX         /* Branch conditional */
	FMT_JAL         /* Jump and link */
	FMT_XOP         /* 3-operand extended */
	FMT_YOP         /* 2-operand extended */
	FMT_ZOP         /* 1-operand extended */
	FMT_VOP         /* 0-operand extended */
)

/* Register names */
var regNames = map[string]int{
	"r0": 0, "r1": 1, "r2": 2, "r3": 3,
	"r4": 4, "r5": 5, "r6": 6, "r7": 7,
	"link": 0, /* LINK is mapped to r0 in many contexts */
}

/* Branch condition codes */
var branchCodes = map[string]int{
	"br":     0,
	"brl":    1,
	"brz":    2,
	"breq":   2,
	"brnz":   3,
	"brneq":  3,
	"brc":    4,
	"bruge":  4,
	"brnc":   5,
	"brult":  5,
	"brsge":  6,
	"brslt":  7,
}

/* Special register names */
var sprNames = map[string]int{
	"link":    0,
	"flags":   1,
	"cyclo":   6,
	"cychi":   7,
	"irr":     8,
	"icr":     9,
	"idr":     10,
	"isr":     11,
	"context": 15,
}
