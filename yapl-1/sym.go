/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// Symbol table
var symtab [SYMTAB_MAX]Syment
var symtabNext SymIndex = 1 // Token value 0 is reserved; see lex.go

// Strings table. Intern strings here. The strings are packed
// end to end with no lengths and no terminators. Offsets and
// lengths are bit-packed elsewhere, e.g. the symbol table.
var strtab [STRTAB_MAX]Byte
var strtabNext StrIndex = 1 // We don't use [0] to help detect bugs

// Allocate the remainder of the string table as temporary byte
// storage.
//
// The tokenizer uses the space after the end of the string table for
// input. If the token is not a string, e.g. it's a bunch of digits,
// the tokenizer can call Discard() after the converting them to a
// value. If token should be kept as a string, the returned position
// becomes the value of the token prototype. After lookup, it may
// turn out the string already exists in the table, and the tokenizer
// again Discard()s the buffer. If the token is a new string, however,
// then SymEnter() commits the space. There's no need to copy the
// token prototype because it's in the right place.
func StrtabAllocate() StrIndex {
	if Word(STRTAB_MAX - strtabNext) < SYMLEN_MAX {
		return ErrorAsStrIndex(ERR_INT_NOSTR)
	}
	return strtabNext
}

// Return the number of bytes remaining between the end of the last
// string referenced from the symbol table and the end of the string
// table. The caller is responsible for avoiding overrun.
func StrtabRemaining() Word {
	if Word(STRTAB_MAX - strtabNext) < SYMLEN_MAX {
		return 0
	}
	return Word(STRTAB_MAX - strtabNext)
}

// Normally the lexer calls StrtabAllocate() and then SymEnter()
// to commit the next token or to discard it if it already has
// a symbol entry. But if the lexer just wants to discard the
// last token, it can call here. In the current design, this
// function doesn't actually have to anything; this may change.
func StrtabDiscard() {
}

// Create a symbol table entry. If the len is 0, val is an
// arbitrary constant value. If the len is > 0, then val is
// an offset in the interned string table. The return value
// is symbol table index if < 0xC000 or an error if >= 0xC000.
//
// If the symbol is already in the table, the existing definition
// is returned. Issues like "symbol already declared" are handled
// a higher level.
//
// If the len is nonzero (i.e. the call is to define a new string
// symbol), then val must be the result of a preceding call to
// StrtabAllocate(). If the symbol is not already in the table,
// the allocated string is committed and the table becomes ready
// to accept a new call to StrtabAllocate(). If the symbol is in
// the table, its index is returned and the symbol in the string
// buffer is discarded, again leaving the system ready for the next
// call to StrtabAllocate().
func SymEnter(val Word, len Byte) SymIndex {
	var symIndex SymIndex

	if len == 0 {
		symIndex = NumLookup(val)
	} else {
		symIndex = SymLookup(StrIndex(val), len)
	}
	if symIndex < symtabNext { // existing definition found
		return symIndex
	}
	if symtabNext >= SYMTAB_MAX {
		return ErrorAsSymIndex(ERR_INT_NOSYM)
	}
	result := symtabNext
	symtabNext++
	strtabNext += StrIndex(len)
	symtab[result].Val = val
	symtab[result].Len = len
	symtab[result].Info = 0
	return result
}

// Look up a value (not a interned string index) in the symbol table.
// Return the symbol table index of the constant, or NODEF.
func NumLookup(val Word) SymIndex {
	for i := SymIndex(0); i < symtabNext; i++ {
		if symtab[i].Len == 0 && symtab[i].Val == val {
			return i
		}
	}
	return ErrorAsSymIndex(ERR_SYM_NODEF)
}

// Experimental interface
func StrLookup(b Byte) SymIndex {
	n := StrtabAllocate()
	strtab[n] = b
	return SymLookup(n, 1)
}

// Look up a symbol in the symbol table. val is index in the
// interned string table. len is its length. We only look at
// symbol table entries with len != 0; their Val fields are
// comparable string intern table indices.
func SymLookup(val StrIndex, len Byte) SymIndex {
	wLen := Word(len)
	if wLen == 0 {
		return ErrorAsSymIndex(ERR_INT_BUG)
	}
	if wLen > SYMLEN_MAX {
		// internal error
		return ErrorAsSymIndex(ERR_INT_TOOBIG)
	}
	for i := SymIndex(0); i < symtabNext; i++ {
		if symtab[i].Len == len {
			failed := false
			s := symtab[i].Val // strtab index
			for j := Word(0); j < Word(len); j++ {
				if strtab[s+j] != strtab[Word(val)+j] {
					failed = true
					break
				}
			}
			if !failed {
				return i
			}
		}
	}
	return ErrorAsSymIndex(ERR_SYM_NODEF)
}

// The argument can be a token (the caller doesn't have to mask
// off the high order 2 bits). If the argument is an error we
// say so.
func PrintSym(n SymIndex) {
	if IsError(n) {
		Printf("; SymIndex TT_ERR %x%n", n)
	} else {
		n &^= SymIndex(ErrBase)
		Printf("; SymIndex %x ", n)
		if symtab[n].Len == 0 {
			Printf("num %x%n", symtab[n].Val)
		} else {
			r := symtab[n].Val
			Printf("sym %c%n", strtab[r])
		}
	}
}

var lastKeySymIndex SymIndex

func isKeySym(si SymIndex) Bool {
	return si <= lastKeySymIndex
}

// All the language symbols in YAPL-1 are single bytes (characters).
// We create a symbol table entry for each one and we check that the
// entry has the expected constant value. The constant value is used
// to represent the token in parser and the AST.
func AddLangSymbol(symRaw Byte, t Token) SymIndex {
	sym := Byte(symRaw&0xFF)
	constval := TokenAsSymIndex(t)

	// We can only do lookups on strings that have a string table index.
	pos := StrtabAllocate()
	strtab[pos] = sym
	result := SymEnter(Word(pos), 1)
	if Word(result) >= ErrBase {
		PrintErr("defining sym %x", ERR_INT_INIT, ERR_FATAL, Word(sym))
	}
	if result != constval {
		PrintErr("defining sym %x", ERR_INT_INIT, ERR_FATAL, Word(sym))
	}
	lastKeySymIndex = result
	return result
}

// All symbols defined by the language:

const A Token = Token(TT_KEY|1) // Result variables displayed by emulator
const B Token = Token(TT_KEY|2)
const C Token = Token(TT_KEY|3)
const D Token = Token(TT_KEY|4)

const E Token = Token(TT_KEY|5) // Keywords
const F Token = Token(TT_KEY|6)
const I Token = Token(TT_KEY|7)
const P Token = Token(TT_KEY|8)
const Q Token = Token(TT_KEY|9)
const V Token = Token(TT_KEY|10)

const HASH Token = Token(TT_KEY|11)	// Punctuation
const SEMI Token = Token(TT_KEY|12)
const EQU  Token = Token(TT_KEY|13)
const BOPEN Token = Token(TT_KEY|14)
const BCLOSE Token = Token(TT_KEY|15)
const PLUS Token = Token(TT_KEY|16)
const ERR Token = Token(TT_KEY|17)

func Init() {
	AddLangSymbol(Byte('A'), A)
	AddLangSymbol(Byte('B'), B)
	AddLangSymbol(Byte('C'), C)
	AddLangSymbol(Byte('D'), D)

	AddLangSymbol(Byte('E'), E)
	AddLangSymbol(Byte('F'), F)
	AddLangSymbol(Byte('I'), I)
	AddLangSymbol(Byte('P'), P)
	AddLangSymbol(Byte('Q'), Q)
	AddLangSymbol(Byte('V'), V)

	AddLangSymbol(Byte('#'), HASH)
	AddLangSymbol(Byte(';'), SEMI)
	AddLangSymbol(Byte('='), EQU)
	AddLangSymbol(Byte('{'), BOPEN)
	AddLangSymbol(Byte('}'), BCLOSE)
	AddLangSymbol(Byte('+'), PLUS)

	AddLangSymbol(Byte('?'), ERR)
}

