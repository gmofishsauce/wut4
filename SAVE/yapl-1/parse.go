/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var AstNodes [AstMaxNode]AstNode
var astNodeNext AstIndex = AstMaxNode - 1

const NO_NODE Word = 0

// Return the next AstNode, allocating from the end of the array toward
// the start. 0 is reserved to mean "no error and also no node".
func allocAstNode() AstIndex {
	result := astNodeNext
	Assert(result >= 1, "out of AST nodes")
	astNodeNext--
	return result
}

// Return a ("conceptually abstract") snapshot of the node allocation
// offset that can be passed to AllocatedSince() in the future to find
// the number of nodes allocated.
func Snapshot() AstIndex {
	return astNodeNext
}

// Return the number of nodes allocated since the argument snapshot.
func AllocatedSince(snapshot AstIndex) Word {
	result := snapshot - astNodeNext
	Assert(result < AstMaxNode, "bad snapshot in AllocatedSince()")
	return Word(result)
}

func MakeAstNode(sym SymIndex, size Word, kind Byte, xtra Byte) AstIndex {
	result := allocAstNode()
	AstNodes[result].Sym = sym
	AstNodes[result].Size = size
	AstNodes[result].Kind = kind
	AstNodes[result].Xtra = xtra
	return result
}

func Parse() AstIndex {
	return program()
}

// Protocol: some callee evaluates children. It determines count of
// children, adds 1 for itself and a node for itself, and returns to
// its caller, repeat.

// Syntax errors cause the result to be an Error node of some type.
// Error nodes are legal AST nodes and do not cause IsError() to
// return true, so the parse may continue. Error returns indicate
// something non-continuable like EOF or an I/O error.

func program() AstIndex {
	var result AstIndex

	pos := Snapshot()
	for result := declaration(); !IsError(result); result = declaration() {
		// nothing
	}
	if IsError(result) {
		return result
	}
	return MakeAstNode(TokenAsSymIndex(P), 1+AllocatedSince(pos), 0, 0)
}

func declaration() AstIndex {
	t := GetToken()

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
	v := GetToken()
	if IsError(v) {
		return ErrorAsAstIndex(v)
	}
	if !IsUsrTok(v) {
		return syntaxError("variable", v)
	}

	// We have the variable decl in v. We could shape this AST subtree
	// with the assignment operator at the top, the newly declared
	// variable as the left child, and the expression as the right
	// child. But this is a bit messy to implement, so we create a
	// node for v, a node for the assignment operation, and finally
	// a node for the expression. We may end up revisiting this.

	decl := MakeAstNode(TokenAsSymIndex(v), 1, AstKindUsr, AstXtraDecl)
	t := GetToken()
	if IsError(t) {
		return ErrorAsAstIndex(t)
	} else if t == SEMI {
		return decl
	} else if t == EQU {
		return assign()
	}
	return syntaxError("semicolon or assignment", t)
}

func function() AstIndex {
	return 0 // TODO
}

func assign() AstIndex {
	return 0 // TODO
}

func IsAstError(a AstIndex) Bool {
	return IsError(a) || IsSyntaxErrorNode(a)
}

func IsSyntaxErrorNode(a AstIndex) Bool {
	return AstNodes[a].Kind == AstKindError
}

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
	for t = GetToken(); t != SEMI && t != BCLOSE && t != TT_EOF; t = GetToken() {
		// nothing
	}
	if t == TT_EOF {
		// Leave it for the caller to consume
		PushbackToken(t)
	}
}
