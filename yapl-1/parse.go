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

type AstNodeIndex Word
const AstMaxNode AstNodeIndex = 2048

var AstNodes [AstMaxNode]AstNode
var astNodeNext AstNodeIndex = AstMaxNode - 1
var Ast [AstMaxNode]AstNodeIndex

func allocNode() AstNodeIndex {
	result := astNodeNext
	if result < 0 {
		panic("out of AST nodes")
	}
	astNodeNext--
	return result
}

func Parse(in Word) AstNodeIndex {
	input = in
	return program()
}

func program() AstNodeIndex {
	// a program is a list of declarations
	var result AstNodeIndex

	for {
		// Get a declaration. If there is a parse error,
		// the result is an error node and the parse can
		// continue. If there is a lower level error, we
		// just stop.
		if n := declaration(); n >= AstMaxNode {
			Ast[allocNode()] = n		
			if result == 0 {
				result = n
			}
		}

		// Check for EOF (or a low level error).
		t := GetToken(input)
		if t == TT_EOF {
			return result
		} else if t >= TT_ERR {
			return AstNodeIndex(t)
		}

		// Neither, so go around again.
		PushbackToken(t)
	}
}

func declaration() AstNodeIndex {
	var result AstNodeIndex
	t := GetToken(STDIN)

	if t == F {
		// function declaration
		panic("TODO: fixme")
		//n := allocNode()
	} else if t == V {
		// variable declaration
		// expect(TYPE_VAR) FIXME pick up editing here
	} else if !isError(t) {
		// unexpected token, but not an error token
		// syntax error - consume and return error node
		// TODO error message improvement
		Printf("; Error: expected declaration, got %x%n", t)
	} else {
		// error token - low level error - just return it
		return AstNodeIndex(t)
	}
	return result
}
