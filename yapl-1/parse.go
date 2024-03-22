/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var input Word // must be stdin for now

// AST nodes use RJ's data oriented tree design. The code knows which
// node types have children. If a node has children, its first child
// node is immediately to its right. Its size is 1 + the size of all
// its children, so the next non-child node is at its index + size.
// It's tempting to try for having the size be a Byte, but I'm afraid
// a Block may have more than 255 Statement-like children. So it goes.

type AstNode struct { // AST node
	Sym Word          // index of symbol table entry
	Size Word         // size of this node (with all subnodes)
}

const AstMaxNode = 2048
type AstNodeIndex Word
var ast [AstMaxNode]AstNode

func Parse(in Word) AstNodeIndex {
	input = in
	return program()
}

func program() AstNodeIndex {
	// a program is a list of declarations
	var root AstNodeIndex = 1
	for {
		declaration()
		t := GetToken(input)
		if t == TT_EOF {
			return root
		} else if t >= TT_ERR {
			return AstNodeIndex(t)
		}
		PushbackToken(t)
	}
}

func declaration() AstNodeIndex {
	return AstMaxNode - 1
}
