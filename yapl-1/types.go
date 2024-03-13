/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// Basic types. Don't use uint16 or uint8 directly because
// it will complicate self hosting if we pull in any library
// functions accidentally.

type Word uint16
type Byte uint8

// These are the only means of input and output, because they
// are the only means implemented in the WUT-4 emulator for
// eventual self hosting.
const (
	STDIN Word = Word(0)
	STDOUT Word = Word(1)
)

// Strings table. Strings are packed end to end with no terminators
// and no lengths. Lengths are stored in the symbol table. We use a
// Golang slice (will need a data structure for self hosting).
var Strtab []Byte

// Literals table. Literal values are stored here so they can be
// indexed with the low 12 bits of a token.
var Littab []Word

type Token Word

const ( // Token types
	TT_INVALID Token = 0x0000  // not allowed - internal error
	TT_ERROR Token = 0x1000    // error code in low 12 bits
	TT_IDENT Token = 0x2000    // length in low 12 bits
	TT_LIT Token = 0x3000      // Littab index in low 12 bits
	TT_KEY Token = 0x4000      // keyword code in low 12 bits
	TT_OP Token = 0x5000       // operator character in low 12 
	TT_PUNCT Token = 0x6000    // punctuation mark in low 12
	TT_OTHER Token = 0xF000    // currently used for EOF only
)

const ( // Defined tokens
	TT_OP_EQ Token = TT_OP|Token('=')
	TT_OP_PLUS Token = TT_OP|Token('+')
	TT_PUNCT_SEMI Token = TT_PUNCT|Token(';')
	TT_PUNCT_OPENBLK Token = TT_PUNCT|Token('{')
	TT_PUNCT_CLOSEBLK Token = TT_PUNCT|Token('}')

	TT_KEY_A Token = TT_KEY|Token('A') // output
	TT_KEY_B Token = TT_KEY|Token('B') // output
	TT_KEY_C Token = TT_KEY|Token('C') // output
	TT_KEY_D Token = TT_KEY|Token('D') // output
	TT_KEY_E Token = TT_KEY|Token('E') // else
	TT_KEY_F Token = TT_KEY|Token('F') // func
	TT_KEY_I Token = TT_KEY|Token('I') // if
	TT_KEY_Q Token = TT_KEY|Token('Q') // quit 
	TT_KEY_V Token = TT_KEY|Token('V') // var

	TT_OTHER_EOF Token = Token(E_EOF) // from io.go
)

const ( // Error subtypes
	ERR_LEX Token = 0x100     // 0x100 .. 0x1FF lexer errors
	ERR_PARSE Token = 0x200   // 0x200 .. 0x2FF parser errors
	ERR_TYPE Token = 0x300    // 0x300 .. 0x3FF type errors
	ERR_IR Token = 0x400      // 0x400 .. 0x4FF IR errors
	ERR_GEN Token = 0x500     // 0x500 .. 0x5FF code gen errors
)

const ( // Lexer errors
	ERR_LEX_INVAL Token = TT_ERROR|ERR_LEX|1   // 0x1101 invalid character
	ERR_LEX_IO Token = TT_ERROR|ERR_LEX|2      // 0x1102 i/o error on input
	ERR_LEX_UNEXP Token = TT_ERROR|ERR_LEX|3   // 0x1103 unexpected char
)

// Symbol table entry. 
type Syment struct {
	Val Word          // index of symbol in string table or lit value
	Len Byte          // Length of name (0 for literal values)
	Info Byte         // Type information
}

// In YAPL-1, there can only be 52 identifiers. All symbol table
// lookup is done by linear search.
var Symtab []Syment

// AST nodes use RJ's data oriented tree design. The code knows which
// node types have children. If a node has children, its first child
// node is immediately to its right. Its size is 1 + the size of all
// its children, so the next non-child node is at its index + size.

type Astnode struct { // AST node
	Val Word          // index of symbol table entry
	Size Word         // size of this node (with all subnodes)
}
