/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// XXX this is stupid, hide this in lexer
var input Word // must be stdin for now

var AstNodes [AstMaxNode]AstNode
var astNodeNext AstIndex = AstMaxNode - 1

const NO_NODE Word = 0

// Return the next AstNode, allocating from the end of the array toward
// the start. 0 is reserved to mean "no error and also no node".
func allocAstNode() AstIndex {
	result := astNodeNext
	if result < 1 {
		panic("out of AST nodes")
	}
	astNodeNext--
	return result
}

func Parse(in Word) AstIndex {
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

func program() AstIndex {
	var n Word;
	for result := declaration(); !IsError(result); result = declaration() {
		n++;
	}
	a := allocAstNode()
	AstNodes[a].Sym = StrLookup('P')
	AstNodes[a].Size = n
	return a
}

func declaration() AstIndex {
	t := GetToken(input)

	if IsError(t) {
		return ErrorAsAstIndex(t)
	} else if t == V {
		return variable()
	} else if t == F {
		return function()
	}

	return syntaxError("declaration", t)
}

// Variable declaration is parsed as a "var" token (already consumed
// by our caller) followed by an optional assignment expression. If
// the assignment is not there, parser injects " = 0".
func variable() AstIndex {
	v := GetToken(input)
	if IsError(v) {
		return ErrorAsAstIndex(v)
	}
	if !IsUsrTok(v) {
		return syntaxError("variable", v)
	}

	// We have the variable name in v
	t := GetToken(input)
	if IsError(t) {
		return ErrorAsAstIndex(t)
	} else if t == SEMI {
		// inject "= 0" for initialization
	} else if t == EQU {
		a := expr()
		if IsError(a) {
			return ErrorAsAstIndex(a)
		}
	}
	return 0 // TODO
}

func function() AstIndex {
	return 0 // TODO
}

func expr() AstIndex {
	return 0 // TODO
}

func IsAstError(a AstIndex) Bool {
	return IsError(a) || IsSyntaxErrorNode(a)
}

func IsSyntaxErrorNode(a AstIndex) Bool {
	return AstNodes[a].Kind == AstKindError
}

/*
// Consume the token. If it matches, don't create an AST node for it.
// Return an error token for errors, 0 for success (because no node
// was created) or 1 to indicate we allocated a syntax error node. In
// the latter case, we resynchronize so the caller doesn't have to.
func expect(exp Token) Word {
	t := GetToken(inFD)	
	if IsError(t) {
		return Word(t)
	} else if IsMatch(exp, t) {
		return 0
	}
	return syntaxError(TokenToString(exp), t)
}
*/

func syntaxError(msg string, bad Token) AstIndex {
	Printf("; syntax error: line %x: expected %s but got %s%n",
		LineNumber(), msg, TokenToString(bad))

	result := allocAstNode()
	AstNodes[result].Sym = ErrorAsSymIndex(TT_ERR)
	AstNodes[result].Size = 1
	AstNodes[result].Kind = AstKindError
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
