package main

const (
	MAGIC_NUMBER = 0xDDD1
	MAGIC_WOF    = 0xDDD2
	HEADER_SIZE  = 16
)

/* WOF section identifiers */
const (
	SEC_UNDEF    = 0
	SEC_CODE_WOF = 1
	SEC_DATA_WOF = 2
)

/* WOF symbol visibility */
const (
	VIS_LOCAL  = 0
	VIS_GLOBAL = 1
)

/* Relocation types */
const (
	R_LDI_CODE    = 0x01 /* 2-word LUI+ADI referencing code-space address */
	R_LDI_DATA    = 0x02 /* 2-word LUI+ADI referencing data-space address */
	R_JAL         = 0x03 /* 2-word LUI+JAL referencing code-space address */
	R_WORD16_CODE = 0x04 /* 16-bit word in data section, code-space address */
	R_WORD16_DATA = 0x05 /* 16-bit word in data section, data-space address */
)

type Relocation struct {
	inDataSeg bool   /* true = reference is in data section */
	rtype     uint8
	offset    uint16 /* byte offset within section */
	symName   string /* resolved to index at write time */
}

/* Token types */
const (
	TOK_EOF = iota
	TOK_NEWLINE
	TOK_LABEL       /* identifier followed by colon */
	TOK_IDENT       /* identifier */
	TOK_NUMBER      /* numeric constant */
	TOK_STRING      /* quoted string */
	TOK_COMMA       /* , */
	TOK_LPAREN      /* ( */
	TOK_RPAREN      /* ) */
	TOK_PLUS        /* + */
	TOK_MINUS       /* - */
	TOK_STAR        /* * */
	TOK_SLASH       /* / */
	TOK_AMP         /* & */
	TOK_PIPE        /* | */
	TOK_TILDE       /* ~ */
	TOK_LSHIFT      /* << */
	TOK_RSHIFT      /* >> */
	TOK_DOLLAR      /* $ */
)

/* Segments */
const (
	SEG_CODE = iota
	SEG_DATA
)

/* Instruction formats */
const (
	FMT_BASE    = iota /* base instructions with imm7 */
	FMT_LUI            /* load upper immediate */
	FMT_BRX            /* branch instructions */
	FMT_JAL            /* jump and link */
	FMT_XOP            /* 3-operand extended */
	FMT_YOP            /* 2-operand extended */
	FMT_ZOP            /* 1-operand extended */
	FMT_VOP            /* 0-operand extended */
)

/* Directive types */
const (
	DIR_ALIGN = iota
	DIR_BYTES
	DIR_WORDS
	DIR_SPACE
	DIR_CODE
	DIR_DATA
	DIR_SET
	DIR_BOOTSTRAP
)

type Token struct {
	typ    int
	text   string
	value  int
	line   int
	column int
}

type Symbol struct {
	name    string
	value   int
	defined bool
	segment int
}

type Instruction struct {
	mnemonic string
	format   int
	opcode   uint16
	numOps   int
}

type Statement struct {
	line      int
	label     string
	directive int
	hasDir    bool
	instr     string
	hasInstr  bool
	args      []string
	numArgs   int
}

type Assembler struct {
	symbols        []Symbol
	numSymbols     int
	codePC         int
	dataPC         int
	currentSeg     int
	codeBuf        []byte
	dataBuf        []byte
	codeSize       int
	dataSize       int
	codeCap        int
	dataCap        int
	pass           int
	errors         int
	inputFile      string
	outputFile     string
	bootstrapMode  bool
	seenCode       bool
	objectMode     bool
	relocations    []Relocation
	lastExternalRef string /* set by expr parser when external symbol used */
}

type Disassembler struct {
	codeBuf  []byte
	dataBuf  []byte
	codeSize int
	dataSize int
}
