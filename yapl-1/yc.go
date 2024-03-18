/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// Initialization creates symbol table entries for all the operators,
// significant punctuation, and keywords, so that AST nodes can be
// basically symbol table references.

func Parse(stdin Word) {
}

func Dump() {
}

func main() {
	Init()
	printOops(4.669)
	printS("bad")
	printX(17)
	Printf("PRINTF%n")

	/*
	for tk := GetToken(STDIN); tk != TT_EOF; tk = GetToken(STDIN) {
		Printf("0x%x%n", GetToken(STDIN))
	}
	*/
	Dump()
}
