/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var input Word // must be stdin for now

var AstNodes [AstMaxNode]AstNode
var astNodeNext AstNodeIndex = AstMaxNode - 1

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

// TODO protocol should (?) be: callee allocates node, fills it in,
// returns index to caller. Caller increments count of child nodes,
// continues. When ready to return, caller writes accumulated count
// to its own allocated node and returns index to its caller, etc.

func program() AstNodeIndex {
	for {
		a := declaration()
		if IsError(Token(a)) {
			return a
		}
	}
}

func declaration() AstNodeIndex {
	t := GetToken(input)

	if t == F {
		return function()
	} else if t == V {
		return variable()
	} else if !IsError(t) {
		PrintErr("expected declaration, got %x", ERR_PARSE_ERR, ERR_CONTINUE, Word(t))
		return resync()
	} else {
		return AstNodeIndex(t)
	}
}

func variable() AstNodeIndex {
	// TODO
	return AstNodeIndex(0)
}

func function() AstNodeIndex {
	// TODO
	return AstNodeIndex(0)
}

func resync() AstNodeIndex {
	var t Token
	for t = GetToken(input); t != SEMI && t != TT_EOF; t = GetToken(input) {
		; // nothing
	}
	// TODO
	return AstNodeIndex(0)
}
