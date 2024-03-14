/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

type Token Word

// Strings table. Intern strings here. The strings are packed
// end to end with no lengths and no terminators. Offsets and
// lengths are bit-packed elsewhere, e.g. the symbol table.
const STRTAB_MAX Word = 8192
var Strtab [STRTAB_MAX]Byte
var strtabNext Word

// Literals table. Intern numeric values.
const LITTAB_MAX Word = 4096
var Littab [LITTAB_MAX]Word
var littabNext Word

func internStr(sym Addr) Token {
	return 0
}

func internLit(lit Word) Token {
	return 0
}


func SymEnter(redefOK bool, val Word, len Byte) Word {
	return 0
	/*
	if SymLookup(HMMM, val is not a token!?, len) {
	}
	if len != 0 {
		
	}
	Symtab = append(Symtab, Syment{
	*/
}

// Look up a symbol. The token must be a STR or NUM
// type, and the token is looked up in the appropriate
// table.
func SymLookup(t Token, len Byte) Word {
	return Word(0)
}
