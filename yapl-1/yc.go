/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// Initialization creates symbol table entries for all the operators,
// significant punctuation, and keywords, so that AST nodes can be
// basically symbol table references.

func Init(fd Word) {
	LexInit(fd)
	SymInit()
}

func main() {
	Init(STDIN)
	for a := Parse(); a < AstMaxNode; a++ {
		Printf("AstNode %x Sym ", Word(a))
		PrintSym(AstNodes[a].Sym)
	}
}
