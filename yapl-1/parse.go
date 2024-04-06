/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var input Word // must be stdin for now

var AstNodes [AstMaxNode]AstNode
var astNodeNext AstNodeIndex = AstMaxNode - 1

func allocAstNode() AstNodeIndex {
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

// Protocol: callee evaluates children, keeps count of directs, at
// the end allocates a node for itself, fills in N, and returns index
// to caller. Which adds callee to its count continues, allocates
// its own node, returns, etc.

// Syntax errors cause the result to be an Error node of some type.
// Error nodes are legal AST nodes and do not cause IsError() to
// return true, so the parse may continue. Errors are typically
// returned for non-syntax type errors,

func program() AstNodeIndex {
	var n Word;
	for result := declaration(); !IsError(Word(result)); result = declaration() {
		n++;
	}
	a := allocAstNode()
	AstNodes[a].Sym = 0
	AstNodes[a].Size = n
	return a
}

func declaration() AstNodeIndex {
	var t Token
	var n Word
	for t = GetToken(input); !IsError(Word(t)) && t != SEMI; t = GetToken(input) {
		n++
		a := allocAstNode()
		AstNodes[a].Sym = TokenAsSymIndex(t)
		AstNodes[a].Size = 1
		AstNodes[a].Info = Word(t)&0xC000 // XXX placeholder
	}
	if IsError(Word(t)) {
		return AstNodeIndex(t)
	}
	// ourselves:
	a := allocAstNode() 
	AstNodes[a].Sym = TokenAsSymIndex(V)
	AstNodes[a].Size = 1 + n
	AstNodes[a].Info = Word(V)&0xC000
	return a
}

/*
func declaration() AstNodeIndex {
	t := GetToken(input)

	if t == F {
		return function()
	} else if t == V {
		return variable()
	} else if !IsError(t) {
		PrintErr("expected declaration, got %x", ERR_PARSE_ERR, ERR_CONTINUE, Word(t))
		return syntaxError(t) // creates error AST node, also calls resync()
	} else {
		return AstNodeIndex(t)
	}
}
*/

/*
func variable() AstNodeIndex {
	t := GetToken(input)
	var result AstNodeIndex

	if IsError(t) {
		result = ErrorAsAstIndex(AsError(t))
	} else if IsUserTok(t) {
		result := allocAstNode()
		AstNodes[result].Sym = Word(t&0x0FFF) // XXX correct, but needs a better way
		AstNodes[result].Size = 1
		t = GetToken(input)
		if t == SEMI {
			// we're done here
		} else if t == EQU {
		}
	} else {
		result = syntaxError(t)
	}
	return result
}

func function() AstNodeIndex {
	// TODO
	return AstNodeIndex(0)
}

func syntaxError(t Token) AstNodeIndex {
	result := allocAstNode()
	AstNodes[result].Sym = Word(ERR&0x0FFF) // XXX correct, but needs a better way
	AstNodes[result].Size = 1
	resync()
	return result
}

// Consume tokens through the next semicolon or closing brace.
// If we hit EOF, push it back so our caller(s) will see it.
func resync() {
	var t Token
	for t = GetToken(input); t != SEMI && t != BCLOSE && t != TT_EOF; t = GetToken(input) {
		; // nothing
	}
	if t == TT_EOF {
		// Leave it for the caller to consume
		PushbackToken(t)
	}
}
*/
