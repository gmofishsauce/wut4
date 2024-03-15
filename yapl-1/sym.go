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
var Symtab [SYMTAB_MAX]Syment
var symtabNext Word

// Strings table. Intern strings here. The strings are packed
// end to end with no lengths and no terminators. Offsets and
// lengths are bit-packed elsewhere, e.g. the symbol table.
const STRTAB_MAX Word = 8192
var Strtab [STRTAB_MAX]Byte
var strtabNext Word

// Allocate the remainder of the string table for token input. The
// tokenizer uses the string table for input. If the token is not
// a string, e.g. it's a bunch of digits, the tokenizer can call
// Discard() after the converting them to a value. If token should
// be kept as a string, the returned position becomes the value of
// the token prototype. After lookup, it may turn out the string
// already exists in the table, and the tokenizer again Discard()s
// the buffer. If the token is a new string, however, then there's
// no need to copy it for interning - it's in the right place.
func StrtabAllocate() (pos Word, len Word) {
	if STRTAB_MAX - strtabNext < SYMLEN_MAX {
		return Word(ERR_INT_NOSTR), 0
	}
	return strtabNext, STRTAB_MAX - strtabNext
}

func StrtabDiscard() {
	// actually there's nothing to do here
}

func StrtabCommit(len Byte) {
	strtabNext += Word(len)
}

// Create a symbol table entry. The ultimate maximum number
// of possible symbol table entries is 0x3FFF because this
// would fill most of the 64k address space anyway. Negative
// values are error returns, positive values are indices.
func SymEnter(redefOK bool, val Word, len Byte) Word {
	var symIndex Word
	var result Word

	if len == 0 {
		symIndex = NumLookup(val)
	} else {
		symIndex = SymLookup(val, len)
	}

	if symIndex < symtabNext { // existing definition found
		if redefOK {
			result = symIndex
		} else {
			result = Word(ERR_SYM_REDEF)
		}
	} else {
		result = symtabNext
		symtabNext++
		Symtab[result].Val = val
		Symtab[result].Len = len
		Symtab[result].Info = 0 // caller's problem
	}
	return result
}

// Look up a constant (non string) in the symbol table.
// Return the symbol table index of the constant.
func NumLookup(val Word) Word {
	for i := Word(0); i < symtabNext; i++ {
		if Symtab[i].Len == 0 && Symtab[i].Val == val {
			return i
		}
	}
	// This constant value should be added to the symbol table
	return Word(ERR_SYM_NODEF)
}

// Look up a symbol in the symbol table. val is index in the
// interned string table. len is its length.
func SymLookup(val Word, len Byte) Word {
	if Word(len) > SYMLEN_MAX {
		// internal error
		return Word(ERR_INT_TOOBIG)
	}
	for i := Word(0); i < symtabNext; i++ {
		if Symtab[i].Len == len {
			failed := false
			s := Symtab[i].Val // Strtab index
			for j := Word(0); j < Word(len); j++ {
				if Strtab[s+j] != Strtab[val+j] {
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
