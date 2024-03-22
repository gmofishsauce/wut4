/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// Basic types. Don't use uint16 or uint8 directly because
// it will complicate self hosting if we pull in any library
// functions accidentally.

type Word uint16
type Byte uint8
type Addr Word
type Bool bool

// These are the only means of input and output, because they
// are the only means implemented in the WUT-4 emulator for
// eventual self hosting.
const (
	STDIN Word = Word(0)
	STDOUT Word = Word(1)
)

// There are four types of tokens: user defined symbols like
// variable names and constant strings, language defined symbols
// ("keys"), numeric values, and error tokens. The types are
// encoded in the high order 2 bits, leaving 14 bits to be used
// for symbol table index (TT_STR, TT_KEY, and TT_NUM) or actual
// value (TT_ERR).
const (
	TT_USR Token = 0x0000      // user symbols from the source
	TT_KEY Token = 0x4000      // language defined symbols
	TT_NUM Token = 0x8000      // numeric valued symbols
	TT_ERR Token = 0xC000      // error tokens
)

func isUser(t Token) Bool {
	return (t&TT_USR) == TT_USR
}

func isKey(t Token) Bool {
	return (t&TT_KEY) == TT_KEY
}

func isNum(t Token) Bool {
	return (t&TT_NUM) == TT_NUM
}

func isError(t Token) Bool {
	return (t&TT_ERR) == TT_ERR
}

// Error types are encoded in the low 12 bits (could be 14 bits).
const ( // Error subtypes
	ERR_LEX Token = 0x100     // 0x100 .. 0x1FF lexer errors
	ERR_PARSE Token = 0x200   // 0x200 .. 0x2FF syntax errors
	ERR_TYPE Token = 0x300    // 0x300 .. 0x3FF type errors
	ERR_IR Token = 0x400      // 0x400 .. 0x4FF IR errors
	ERR_GEN Token = 0x500     // 0x500 .. 0x5FF code gen errors
	ERR_SYM Token = 0x600     // 0x600 .. 0x6FF symbol table errors
	ERR_INT Token = 0x700     // 0x700 .. 0x6FF internal errors
)

const TT_EOF Token = Token(E_EOF) // 0xFFFF io.go

const ( // Lexer errors
	ERR_LEX_INVAL Token = TT_ERR|ERR_LEX|1   // 0xC101 invalid character
	ERR_LEX_IO Token = TT_ERR|ERR_LEX|2      // 0xC102 i/o error on input
	ERR_LEX_UNEXP Token = TT_ERR|ERR_LEX|3   // 0xC103 unexpected char
)

const ( // Symbol table errors
	ERR_SYM_REDEF Token = TT_ERR|ERR_SYM|1   // 0xC601 symbol redefined
	ERR_SYM_NODEF Token = TT_ERR|ERR_SYM|2   // 0xC602 symbol undefined
)

const ( // internal errors, e.g. out of space
	ERR_INT_NOSTR Token = TT_ERR|ERR_INT|1   // 0xC701 string table full
	ERR_INT_NOSYM Token = TT_ERR|ERR_INT|2   // 0xC702 symbol table full
	ERR_INT_TOOBIG Token = TT_ERR|ERR_INT|3  // 0xC703 symbol or string too long
	ERR_INT_BUG Token = TT_ERR|ERR_INT|4     // 0xC704 unspecified internal error
	ERR_INT_INIT Token = TT_ERR|ERR_INT|5    // 0xC705 initialization error
)

// Error severities
const ERR_CONTINUE = Word(1)
const ERR_FATAL    = Word(2)

// All the language symbols in YAPL-1 are single bytes (characters).
// We create a symbol table entry for each one and we check that the
// entry has the expected constant value. The constant value is used
// to represent the token in parser and the AST.
func AddLangSymbol(symRaw Byte, constvalRaw Token) Word {
	sym := Byte(symRaw&0xFF)
	constval := Word(constvalRaw)

	pos := StrtabAllocate()
	strtab[pos] = sym // Every language symbol is 1 character in yapl-1
	result := SymEnter(false, pos, 1)
	if result != constval&0xFFF {
		Error(Word(ERR_INT_INIT), ERR_FATAL, Word(pos), Word(sym))
	}
	return pos
}

const A Token = TT_KEY|1
const B Token = TT_KEY|2
const C Token = TT_KEY|3
const D Token = TT_KEY|4

const E Token = TT_KEY|5
const F Token = TT_KEY|6
const I Token = TT_KEY|7
const Q Token = TT_KEY|8
const V Token = TT_KEY|9

const HASH Token = TT_KEY|10
const SEMI Token = TT_KEY|11
const EQU  Token = TT_KEY|12
const BOPEN Token = TT_KEY|13
const BCLOSE Token = TT_KEY|14
const PLUS Token = TT_KEY|15

func Init() {
	AddLangSymbol(Byte('A'), A)
	AddLangSymbol(Byte('B'), B)
	AddLangSymbol(Byte('C'), C)
	AddLangSymbol(Byte('D'), D)

	AddLangSymbol(Byte('E'), E)
	AddLangSymbol(Byte('F'), F)
	AddLangSymbol(Byte('I'), I)
	AddLangSymbol(Byte('Q'), Q)
	AddLangSymbol(Byte('V'), V)

	AddLangSymbol(Byte('#'), HASH)
	AddLangSymbol(Byte(';'), SEMI)
	AddLangSymbol(Byte('='), EQU)
	AddLangSymbol(Byte('{'), BOPEN)
	AddLangSymbol(Byte('}'), BCLOSE)
	AddLangSymbol(Byte('+'), PLUS)
}

