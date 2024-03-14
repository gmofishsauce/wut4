/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// Basic types. Don't use uint16 or uint8 directly because
// it will complicate self hosting if we pull in any library
// functions accidentally.

type Word uint16
type Byte uint8
type Addr Word

// These are the only means of input and output, because they
// are the only means implemented in the WUT-4 emulator for
// eventual self hosting.
const (
	STDIN Word = Word(0)
	STDOUT Word = Word(1)
)

// There are four types of tokens: user defined symbols,
// language defined symbols ("keys"), error tokens, and
// "alt" tokens which are reserved for anything else that
// might be needed. The types are encoded in the high
// order 2 bits, leaving 14 bits to be used as the token
// type chooses.
const (
	TT_STR Token = 0x0000      // string valued symbols
	TT_NUM Token = 0x8000      // numeric valued symbols
	TT_KEY Token = 0x4000      // language defined symbols
	TT_ERR Token = 0xC000      // error tokens
)

const ( // language defined tokens
	TT_KEY_EQ Token = TT_KEY|Token('=')
	TT_KEY_PLUS Token = TT_KEY|Token('+')
	TT_KEY_SEMI Token = TT_KEY|Token(';')
	TT_KEY_OPENBLK Token = TT_KEY|Token('{')
	TT_KEY_CLOSEBLK Token = TT_KEY|Token('}')

	TT_KEY_A Token = TT_KEY|Token('A') // output
	TT_KEY_B Token = TT_KEY|Token('B') // output
	TT_KEY_C Token = TT_KEY|Token('C') // output
	TT_KEY_D Token = TT_KEY|Token('D') // output
	TT_KEY_E Token = TT_KEY|Token('E') // else
	TT_KEY_F Token = TT_KEY|Token('F') // func
	TT_KEY_I Token = TT_KEY|Token('I') // if
	TT_KEY_Q Token = TT_KEY|Token('Q') // quit 
	TT_KEY_V Token = TT_KEY|Token('V') // var
)

const ( // Error subtypes
	ERR_LEX Token = 0x100     // 0x100 .. 0x1FF lexer errors
	ERR_PARSE Token = 0x200   // 0x200 .. 0x2FF parser errors
	ERR_TYPE Token = 0x300    // 0x300 .. 0x3FF type errors
	ERR_IR Token = 0x400      // 0x400 .. 0x4FF IR errors
	ERR_GEN Token = 0x500     // 0x500 .. 0x5FF code gen errors
)

const TT_EOF Token = Token(E_EOF) // 0xFFFF io.go

const ( // Lexer errors
	ERR_LEX_INVAL Token = TT_ERR|ERR_LEX|1   // 0x1101 invalid character
	ERR_LEX_IO Token = TT_ERR|ERR_LEX|2      // 0x1102 i/o error on input
	ERR_LEX_UNEXP Token = TT_ERR|ERR_LEX|3   // 0x1103 unexpected char
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
	Sym Word          // index of symbol table entry
	Size Word         // size of this node (with all subnodes)
}
