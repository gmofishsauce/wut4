/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var input Word // must be stdin for now

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

// TODO protocol should (?) be: callee allocates node, fills it in,
// returns index to caller. Caller places node in (index) in tree.
// At end, caller allocates its own node, writes length of children,
// and returns index to caller.

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
		parseErr(ERR_PARSE, ERR_CONTINUE, 0, 0) // TODO inadequate
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

func parseErr(code Error, sev Word, print1 Word, print2 Word) {
	PrintErr(code, sev, print1, print2)
}

func resync() AstNodeIndex {
	var t Token
	for t = GetToken(input); t != SEMI && t != TT_EOF; t = GetToken(input) {
		; // nothing
	}
	// TODO
	return AstNodeIndex(0)
}
