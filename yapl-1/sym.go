/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

type Token Word

// There are some maximum size constants below. The ultimate max of these
// max sizes is 0xC000 because error codes are the values 0xC000 - 0xFFFF.
// In fact, likely none of these can be greater than 0x8000, if even that, 
// because the tables in this file need to be stored simultaneously in a
// 64k address space during the first pass. The symbol length max, now 15,
// cannot ultimately exceed 255.

// Symbol table entry. Values may be string table offsets in which case
// the length is relevant, or they maybe constant values where the length
// is 0.
type Syment struct {
	Val Word          // index in string table or lit value
	Len Byte          // Length of name or 0 for lit value
	Info Byte         // Type information
}

const SYMLEN_MAX Word = 16
const SYMTAB_MAX Word = 4096
var symtab [SYMTAB_MAX]Syment
var symtabNext Word = 1 // We don't use [0] to help detect bugs

// Strings table. Intern strings here. The strings are packed
// end to end with no lengths and no terminators. Offsets and
// lengths are bit-packed elsewhere, e.g. the symbol table.
const STRTAB_MAX Word = 8192
var strtab [STRTAB_MAX]Byte
var strtabNext Word = 1 // We don't use [0] to help detect bugs

// Allocate the remainder of the string table as temporary byte
// storage.
//
// The okenizer uses the space after the end of the string table for
// input. If the token is not a string, e.g. it's a bunch of digits,
// the tokenizer can call Discard() after the converting them to a
// value. If token should be kept as a string, the returned position
// becomes the value of the token prototype. After lookup, it may
// turn out the string already exists in the table, and the tokenizer
// again Discard()s the buffer. If the token is a new string, however,
// then SymEnter() commits the space. There's no need to copy the
// token prototype because it's in the right place.
func StrtabAllocate() Word {
	if STRTAB_MAX - strtabNext < SYMLEN_MAX {
		return Word(ERR_INT_NOSTR)
	}
	return strtabNext
}

// Return the number of bytes remaining between the end of the last
// string referenced from the symbol table and the end of the string
// table. The caller is responsible for avoiding overrun.
func StrtabRemaining() Word {
	if STRTAB_MAX - strtabNext < SYMLEN_MAX {
		return 0
	}
	return STRTAB_MAX - strtabNext
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
// If redefOK is true, then finding an existing defininition
// causes that existing definition, if any, to be returned.
// Otherwise, ERR_SYM_REDEF is returned. If a new symbol table
// entry is allocated, the Info field of the new entry is set
// to 0.
//
// If the len is nonzero (i.e. the call is to define a new
// string symbol), then val must be the result of a preceding
// call to StrtabAllocate(). The allocated string is committed
// and the symbol table becomes ready to accept a new call to
// StrtabAllocate().
func SymEnter(redefOK bool, val Word, len Byte) Word {
	var symIndex Word

	if len == 0 {
		symIndex = NumLookup(val)
	} else {
		symIndex = SymLookup(val, len)
	}
	if symIndex < symtabNext { // existing definition found
		if redefOK {
			return symIndex
		}
		return  Word(ERR_SYM_REDEF)
	}
	if symtabNext >= SYMTAB_MAX {
		return Word(ERR_INT_NOSYM)
	}
	result := symtabNext
	symtabNext++
	strtabNext += Word(len)
	symtab[result].Val = val
	symtab[result].Len = len
	symtab[result].Info = 0
	return result
}

// Look up a value (not a interned string index) in the symbol table.
// Return the symbol table index of the constant, or NODEF.
func NumLookup(val Word) Word {
	for i := Word(0); i < symtabNext; i++ {
		if symtab[i].Len == 0 && symtab[i].Val == val {
			return i
		}
	}
	return Word(ERR_SYM_NODEF)
}

// Look up a symbol in the symbol table. val is index in the
// interned string table. len is its length. We only look at
// symbol table entries with len != 0; their Val fields are
// comparable string intern table indices.
func SymLookup(val Word, len Byte) Word {
	wLen := Word(len)
	if wLen == 0 {
		return Word(ERR_INT_BUG)
	}
	if wLen > SYMLEN_MAX {
		// internal error
		return Word(ERR_INT_TOOBIG)
	}
	for i := Word(0); i < symtabNext; i++ {
		if symtab[i].Len == len {
			failed := false
			s := symtab[i].Val // strtab index
			for j := Word(0); j < Word(len); j++ {
				if strtab[s+j] != strtab[val+j] {
					failed = true
					break
				}
			}
			if !failed {
				return i
			}
		}
	}
	return Word(ERR_SYM_NODEF)
}
