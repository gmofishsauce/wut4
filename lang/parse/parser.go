// YAPL Parser - Main Parser
// Recursive descent parser with panic-mode error recovery

package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser parses tokens into an AST
type Parser struct {
	tokens    *TokenReader
	symtab    *SymbolTable
	funcScope *FuncScope // current function scope, nil if at global level
	errors    []string
	panicMode bool // in error recovery mode
}

// NewParser creates a new parser
func NewParser(tokens *TokenReader) *Parser {
	return &Parser{
		tokens:    tokens,
		symtab:    NewSymbolTable(),
		funcScope: nil,
		errors:    nil,
		panicMode: false,
	}
}

// Parse parses the token stream and returns the AST
func (p *Parser) Parse() (*Program, *SymbolTable, []string) {
	prog := &Program{
		Decls: make([]Decl, 0),
	}

	for !p.tokens.AtEOF() {
		decl := p.parseDeclaration()
		if decl != nil {
			prog.Decls = append(prog.Decls, decl)
		}
	}

	return prog, p.symtab, p.errors
}

// Error handling

func (p *Parser) error(format string, args ...interface{}) {
	tok := p.tokens.Peek()
	msg := fmt.Sprintf("%s:%d: %s", tok.File, tok.Line, fmt.Sprintf(format, args...))
	p.errors = append(p.errors, msg)
	p.panicMode = true
}

func (p *Parser) errorAt(loc SourceLoc, format string, args ...interface{}) {
	msg := fmt.Sprintf("%s:%d: %s", loc.File, loc.Line, fmt.Sprintf(format, args...))
	p.errors = append(p.errors, msg)
	p.panicMode = true
}

// synchronize skips tokens until a synchronization point
func (p *Parser) synchronize() {
	p.panicMode = false
	for !p.tokens.AtEOF() {
		tok := p.tokens.Peek()
		// Synchronize at declaration keywords
		if tok.Category == TokKEY {
			switch tok.Value {
			case "const", "var", "func", "struct":
				return
			}
		}
		// Synchronize at semicolons and closing braces
		if tok.IsPunct(";") || tok.IsPunct("}") {
			p.tokens.Next() // consume the sync token
			return
		}
		p.tokens.Next()
	}
}

// synchronizeStmt synchronizes within a function body
func (p *Parser) synchronizeStmt() {
	p.panicMode = false
	for !p.tokens.AtEOF() {
		tok := p.tokens.Peek()
		// Synchronize at statement keywords
		if tok.Category == TokKEY {
			switch tok.Value {
			case "if", "while", "for", "return", "break", "continue", "goto", "var", "const":
				return
			}
		}
		// Synchronize at semicolons, closing braces, or labels
		if tok.IsPunct(";") {
			p.tokens.Next()
			return
		}
		if tok.IsPunct("}") {
			return // don't consume, let caller handle
		}
		// Check for label (identifier followed by colon)
		if tok.Category == TokID {
			// Peek ahead to see if it's a label
			// We can't easily do this, so just return and let the label be parsed
			return
		}
		p.tokens.Next()
	}
}

// currentLoc returns the current source location
func (p *Parser) currentLoc() SourceLoc {
	tok := p.tokens.Peek()
	return SourceLoc{File: tok.File, Line: tok.Line}
}

// ============================================================
// Type Parsing
// ============================================================

// parseType parses a type specifier
func (p *Parser) parseType() *Type {
	tok := p.tokens.Peek()

	// Pointer type: @ TypeSpecifier or @ void
	if tok.IsPunct("@") {
		p.tokens.Next()
		nextTok := p.tokens.Peek()
		if nextTok.IsKeyword("void") {
			p.tokens.Next()
			return NewPointerType(TypeVoidType)
		}
		pointee := p.parseType()
		if pointee == nil {
			return nil
		}
		return NewPointerType(pointee)
	}

	// Base types
	if tok.Category == TokKEY {
		switch tok.Value {
		case "byte", "uint8":
			p.tokens.Next()
			return TypeUint8Type
		case "int16":
			p.tokens.Next()
			return TypeInt16Type
		case "uint16":
			p.tokens.Next()
			return TypeUint16Type
		case "block32":
			p.tokens.Next()
			return TypeBlock32Type
		case "block64":
			p.tokens.Next()
			return TypeBlock64Type
		case "block128":
			p.tokens.Next()
			return TypeBlock128Type
		case "void":
			p.tokens.Next()
			return TypeVoidType
		case "struct":
			// struct identifier
			p.tokens.Next()
			nameTok := p.tokens.Peek()
			if nameTok.Category != TokID {
				p.error("expected struct name")
				return nil
			}
			p.tokens.Next()
			return NewStructType(nameTok.Value)
		}
	}

	// Struct type by name (identifier that's a struct)
	if tok.Category == TokID {
		// Check if it's a known struct
		if p.symtab.LookupStruct(tok.Value) != nil {
			p.tokens.Next()
			return NewStructType(tok.Value)
		}
		// Could be a forward reference - allow it for now
		// Semantic analysis will catch undefined types
		p.tokens.Next()
		return NewStructType(tok.Value)
	}

	p.error("expected type, got %s %q", tok.Category, tok.Value)
	return nil
}

// parseReturnType parses a return type (type or void)
func (p *Parser) parseReturnType() *Type {
	return p.parseType()
}

// isTypeName returns true if the token starts a type
func (p *Parser) isTypeName() bool {
	tok := p.tokens.Peek()
	if tok.IsPunct("@") {
		return true
	}
	if tok.Category == TokKEY {
		switch tok.Value {
		case "byte", "uint8", "int16", "uint16", "void",
			"block32", "block64", "block128", "struct":
			return true
		}
	}
	// Check for struct name
	if tok.Category == TokID {
		return p.symtab.LookupStruct(tok.Value) != nil
	}
	return false
}

// ============================================================
// Declaration Parsing
// ============================================================

// parseDeclaration parses a top-level declaration
func (p *Parser) parseDeclaration() Decl {
	tok := p.tokens.Peek()

	if tok.IsKeyword("const") {
		return p.parseConstDecl()
	}
	if tok.IsKeyword("var") {
		return p.parseVarDecl(true) // global
	}
	if tok.IsKeyword("func") {
		return p.parseFuncDecl()
	}
	if tok.IsKeyword("struct") {
		return p.parseStructDecl()
	}

	p.error("expected declaration, got %s %q", tok.Category, tok.Value)
	p.synchronize()
	return nil
}

// parseConstDecl parses: const TypeSpecifier identifier = ConstExpr ;
func (p *Parser) parseConstDecl() *ConstDecl {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'const'

	constType := p.parseType()
	if constType == nil {
		p.synchronize()
		return nil
	}

	nameTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected identifier in const declaration")
		p.synchronize()
		return nil
	}

	if _, err := p.tokens.ExpectPunct("="); err != nil {
		p.error("expected '=' after constant name")
		p.synchronize()
		return nil
	}

	// The value should be a literal (lexer has already folded constant expressions)
	valTok := p.tokens.Next()
	if valTok.Category != TokLIT {
		p.error("expected constant value")
		p.synchronize()
		return nil
	}

	value := p.parseLiteralValue(valTok)

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after constant declaration")
		p.synchronize()
	}

	decl := &ConstDecl{
		Name:      nameTok.Value,
		ConstType: constType,
		Value:     value,
		Loc:       loc,
	}

	// Add to symbol table
	if p.funcScope != nil {
		if err := p.funcScope.AddLocalConst(nameTok.Value, value, loc); err != nil {
			p.errorAt(loc, "%v", err)
		}
	} else {
		if err := p.symtab.DefineConst(nameTok.Value, value, loc); err != nil {
			p.errorAt(loc, "%v", err)
		}
	}

	return decl
}

// parseVarDecl parses a variable declaration
func (p *Parser) parseVarDecl(isGlobal bool) *VarDecl {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'var'

	varType := p.parseType()
	if varType == nil {
		p.synchronize()
		return nil
	}

	nameTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected identifier in variable declaration")
		p.synchronize()
		return nil
	}

	var arrayLen int
	var init Expr

	// Check for array dimension
	if p.tokens.Peek().IsPunct("[") {
		p.tokens.Next() // consume '['

		// Array dimension should be a literal (already folded by lexer)
		dimTok := p.tokens.Next()
		if dimTok.Category != TokLIT {
			p.error("expected constant array dimension")
			p.synchronize()
			return nil
		}
		arrayLen = int(p.parseLiteralValue(dimTok))

		if _, err := p.tokens.ExpectPunct("]"); err != nil {
			p.error("expected ']' after array dimension")
			p.synchronize()
			return nil
		}
	}

	// Check for initializer
	if p.tokens.Peek().IsPunct("=") {
		p.tokens.Next() // consume '='

		if arrayLen > 0 {
			// Array initializer: { expr, expr, ... } or string literal
			init = p.parseArrayInit()
		} else {
			init = p.parseExpression()
		}
	}

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after variable declaration")
		p.synchronize()
	}

	decl := &VarDecl{
		Name:     nameTok.Value,
		VarType:  varType,
		ArrayLen: arrayLen,
		Init:     init,
		Loc:      loc,
	}

	// Add to symbol table
	if isGlobal {
		if err := p.symtab.DefineGlobalVar(nameTok.Value, varType, arrayLen, loc); err != nil {
			p.errorAt(loc, "%v", err)
		}
	} else if p.funcScope != nil {
		if err := p.funcScope.AddLocal(nameTok.Value, varType, arrayLen, p.symtab.Structs, loc); err != nil {
			p.errorAt(loc, "%v", err)
		}
	}

	return decl
}

// parseArrayInit parses an array initializer
func (p *Parser) parseArrayInit() Expr {
	tok := p.tokens.Peek()

	// String literal
	if tok.Category == TokLIT && strings.HasPrefix(tok.Value, "\"") {
		p.tokens.Next()
		return &LiteralExpr{
			baseExpr: baseExpr{Loc: SourceLoc{File: tok.File, Line: tok.Line}},
			Kind:     LitString,
			StrVal:   tok.Value,
		}
	}

	// Brace-enclosed list: { expr, expr, ... }
	// For now, just parse as a single expression (simplified)
	// Full array initializer support would require a list
	return p.parseExpression()
}

// parseStructDecl parses a struct declaration
func (p *Parser) parseStructDecl() *StructDecl {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'struct'

	nameTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected identifier after 'struct'")
		p.synchronize()
		return nil
	}

	if _, err := p.tokens.ExpectPunct("{"); err != nil {
		p.error("expected '{' in struct declaration")
		p.synchronize()
		return nil
	}

	fields := make([]*FieldDecl, 0)
	for !p.tokens.Peek().IsPunct("}") && !p.tokens.AtEOF() {
		field := p.parseStructField()
		if field != nil {
			fields = append(fields, field)
		}
		if p.panicMode {
			p.synchronizeStmt()
		}
	}

	if _, err := p.tokens.ExpectPunct("}"); err != nil {
		p.error("expected '}' after struct fields")
	}

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after struct declaration")
	}

	decl := &StructDecl{
		Name:   nameTok.Value,
		Fields: fields,
		Loc:    loc,
	}

	// Add to symbol table and compute layout
	def, err2 := p.symtab.DefineStruct(nameTok.Value, fields, loc)
	if err2 != nil {
		p.errorAt(loc, "%v", err2)
	} else {
		decl.Size = def.Size
		decl.Align = def.Align
		// Update field offsets
		for i, f := range def.Fields {
			fields[i].Offset = f.Offset
		}
	}

	return decl
}

// parseStructField parses a struct field
func (p *Parser) parseStructField() *FieldDecl {
	loc := p.currentLoc()

	fieldType := p.parseType()
	if fieldType == nil {
		return nil
	}

	nameTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected field name")
		return nil
	}

	var arrayLen int
	if p.tokens.Peek().IsPunct("[") {
		p.tokens.Next()
		dimTok := p.tokens.Next()
		if dimTok.Category != TokLIT {
			p.error("expected constant array dimension")
			return nil
		}
		arrayLen = int(p.parseLiteralValue(dimTok))
		if _, err := p.tokens.ExpectPunct("]"); err != nil {
			p.error("expected ']'")
		}
	}

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after field")
	}

	return &FieldDecl{
		Name:      nameTok.Value,
		FieldType: fieldType,
		ArrayLen:  arrayLen,
		Loc:       loc,
	}
}

// parseFuncDecl parses a function declaration
func (p *Parser) parseFuncDecl() *FuncDecl {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'func'

	returnType := p.parseReturnType()
	if returnType == nil {
		p.synchronize()
		return nil
	}

	nameTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected function name")
		p.synchronize()
		return nil
	}

	if _, err := p.tokens.ExpectPunct("("); err != nil {
		p.error("expected '(' after function name")
		p.synchronize()
		return nil
	}

	// Create function symbol
	funcSym, err2 := p.symtab.DefineFunc(nameTok.Value, returnType, loc)
	if err2 != nil {
		p.errorAt(loc, "%v", err2)
		// Create a dummy symbol for error recovery
		funcSym = &Symbol{
			Name:     nameTok.Value,
			Kind:     SymFunc,
			Type:     returnType,
			Loc:      loc,
			Params:   make([]*ParamSymbol, 0),
			Locals:   make([]*LocalSymbol, 0),
			Labels:   make(map[string]*LabelSymbol),
		}
	}

	// Enter function scope
	p.funcScope = NewFuncScope(funcSym)

	// Parse parameters
	params := make([]*Param, 0)
	if !p.tokens.Peek().IsPunct(")") {
		for {
			param := p.parseParam()
			if param != nil {
				params = append(params, param)
			}
			if !p.tokens.Peek().IsPunct(",") {
				break
			}
			p.tokens.Next() // consume ','
		}
	}

	if _, err := p.tokens.ExpectPunct(")"); err != nil {
		p.error("expected ')' after parameters")
	}

	// Parse function body
	if _, err := p.tokens.ExpectPunct("{"); err != nil {
		p.error("expected '{' to start function body")
		p.synchronize()
		p.funcScope = nil
		return nil
	}

	// Parse local declarations
	locals := make([]LocalDecl, 0)
	for p.tokens.Peek().IsKeyword("var") || p.tokens.Peek().IsKeyword("const") {
		if p.tokens.Peek().IsKeyword("var") {
			decl := p.parseVarDecl(false)
			if decl != nil {
				locals = append(locals, decl)
			}
		} else {
			decl := p.parseConstDecl()
			if decl != nil {
				locals = append(locals, decl)
			}
		}
		if p.panicMode {
			p.synchronizeStmt()
		}
	}

	// Parse statements
	body := make([]FuncStmt, 0)
	stmtIndex := 0
	for !p.tokens.Peek().IsPunct("}") && !p.tokens.AtEOF() {
		stmt := p.parseFuncStmt(stmtIndex)
		if stmt != nil {
			body = append(body, stmt)
			stmtIndex++
		}
		if p.panicMode {
			p.synchronizeStmt()
		}
	}

	if _, err := p.tokens.ExpectPunct("}"); err != nil {
		p.error("expected '}' at end of function")
	}

	// Finalize function scope
	p.funcScope.Finalize()

	decl := &FuncDecl{
		Name:       nameTok.Value,
		ReturnType: returnType,
		Params:     params,
		Locals:     locals,
		Body:       body,
		Loc:        loc,
	}

	// Exit function scope
	p.funcScope = nil

	return decl
}

// parseParam parses a function parameter
func (p *Parser) parseParam() *Param {
	loc := p.currentLoc()

	paramType := p.parseType()
	if paramType == nil {
		return nil
	}

	nameTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected parameter name")
		return nil
	}

	param := &Param{
		Name:      nameTok.Value,
		ParamType: paramType,
		Loc:       loc,
	}

	// Add to function scope
	if p.funcScope != nil {
		if err := p.funcScope.AddParam(nameTok.Value, paramType, loc); err != nil {
			p.errorAt(loc, "%v", err)
		}
	}

	return param
}

// ============================================================
// Statement Parsing
// ============================================================

// parseFuncStmt parses a statement or label at function body level
func (p *Parser) parseFuncStmt(stmtIndex int) FuncStmt {
	tok := p.tokens.Peek()

	// Check for label: identifier followed by ':'
	if tok.Category == TokID {
		// Look ahead for ':'
		// We need to be careful here - save position conceptually
		// Actually, just check if next-next is ':'
		// For simplicity, peek at current and check
		// This is a bit awkward with our simple token reader

		// Try parsing as label
		name := tok.Value
		p.tokens.Next() // consume identifier
		if p.tokens.Peek().IsPunct(":") {
			p.tokens.Next() // consume ':'
			loc := SourceLoc{File: tok.File, Line: tok.Line}

			// Add to function scope
			if p.funcScope != nil {
				if err := p.funcScope.AddLabel(name, stmtIndex, loc); err != nil {
					p.errorAt(loc, "%v", err)
				}
			}

			return &LabelStmt{
				Label: name,
				Loc:   loc,
			}
		}
		// Not a label - it's an expression starting with identifier
		// We need to "unread" the identifier - but we can't easily
		// Instead, parse it as an expression statement starting with this identifier
		return p.parseExprStmtStartingWith(tok)
	}

	// Parse a regular statement - all Stmt types also implement FuncStmt
	stmt := p.parseStatement()
	if stmt == nil {
		return nil
	}
	// Type assertion: all concrete Stmt types implement FuncStmt
	if fs, ok := stmt.(FuncStmt); ok {
		return fs
	}
	return nil
}

// parseStatement parses a statement
func (p *Parser) parseStatement() Stmt {
	tok := p.tokens.Peek()

	if tok.IsPunct("{") {
		return p.parseBlock()
	}
	if tok.IsPunct(";") {
		loc := p.currentLoc()
		p.tokens.Next()
		return &ExprStmt{X: nil, Loc: loc}
	}
	if tok.IsKeyword("if") {
		return p.parseIfStmt()
	}
	if tok.IsKeyword("while") {
		return p.parseWhileStmt()
	}
	if tok.IsKeyword("for") {
		return p.parseForStmt()
	}
	if tok.IsKeyword("return") {
		return p.parseReturnStmt()
	}
	if tok.IsKeyword("break") {
		return p.parseBreakStmt()
	}
	if tok.IsKeyword("continue") {
		return p.parseContinueStmt()
	}
	if tok.IsKeyword("goto") {
		return p.parseGotoStmt()
	}

	// Expression statement
	return p.parseExprStmt()
}

// parseBlock parses a block: { statements }
func (p *Parser) parseBlock() *Block {
	loc := p.currentLoc()
	p.tokens.Next() // consume '{'

	stmts := make([]Stmt, 0)
	for !p.tokens.Peek().IsPunct("}") && !p.tokens.AtEOF() {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		if p.panicMode {
			p.synchronizeStmt()
		}
	}

	if _, err := p.tokens.ExpectPunct("}"); err != nil {
		p.error("expected '}'")
	}

	return &Block{Stmts: stmts, Loc: loc}
}

// parseIfStmt parses: if ( expr ) stmt [else stmt]
func (p *Parser) parseIfStmt() *IfStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'if'

	if _, err := p.tokens.ExpectPunct("("); err != nil {
		p.error("expected '(' after 'if'")
		return nil
	}

	cond := p.parseExpression()

	if _, err := p.tokens.ExpectPunct(")"); err != nil {
		p.error("expected ')' after if condition")
	}

	then := p.parseStatement()

	var els Stmt
	if p.tokens.Peek().IsKeyword("else") {
		p.tokens.Next()
		els = p.parseStatement()
	}

	return &IfStmt{
		Cond: cond,
		Then: then,
		Else: els,
		Loc:  loc,
	}
}

// parseWhileStmt parses: while ( expr ) stmt
func (p *Parser) parseWhileStmt() *WhileStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'while'

	if _, err := p.tokens.ExpectPunct("("); err != nil {
		p.error("expected '(' after 'while'")
		return nil
	}

	cond := p.parseExpression()

	if _, err := p.tokens.ExpectPunct(")"); err != nil {
		p.error("expected ')' after while condition")
	}

	body := p.parseStatement()

	return &WhileStmt{
		Cond: cond,
		Body: body,
		Loc:  loc,
	}
}

// parseForStmt parses: for ( [expr] ; [expr] ; [expr] ) stmt
func (p *Parser) parseForStmt() *ForStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'for'

	if _, err := p.tokens.ExpectPunct("("); err != nil {
		p.error("expected '(' after 'for'")
		return nil
	}

	var init, cond, post Expr

	// Init
	if !p.tokens.Peek().IsPunct(";") {
		init = p.parseExpression()
	}
	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' in for statement")
	}

	// Condition
	if !p.tokens.Peek().IsPunct(";") {
		cond = p.parseExpression()
	}
	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' in for statement")
	}

	// Post
	if !p.tokens.Peek().IsPunct(")") {
		post = p.parseExpression()
	}

	if _, err := p.tokens.ExpectPunct(")"); err != nil {
		p.error("expected ')' after for clauses")
	}

	body := p.parseStatement()

	return &ForStmt{
		Init: init,
		Cond: cond,
		Post: post,
		Body: body,
		Loc:  loc,
	}
}

// parseReturnStmt parses: return [expr] ;
func (p *Parser) parseReturnStmt() *ReturnStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'return'

	var value Expr
	if !p.tokens.Peek().IsPunct(";") {
		value = p.parseExpression()
	}

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after return")
	}

	return &ReturnStmt{Value: value, Loc: loc}
}

// parseBreakStmt parses: break ;
func (p *Parser) parseBreakStmt() *BreakStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'break'

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after break")
	}

	return &BreakStmt{Loc: loc}
}

// parseContinueStmt parses: continue ;
func (p *Parser) parseContinueStmt() *ContinueStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'continue'

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after continue")
	}

	return &ContinueStmt{Loc: loc}
}

// parseGotoStmt parses: goto identifier ;
func (p *Parser) parseGotoStmt() *GotoStmt {
	loc := p.currentLoc()
	p.tokens.Next() // consume 'goto'

	labelTok, err := p.tokens.ExpectID()
	if err != nil {
		p.error("expected label after 'goto'")
		return nil
	}

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after goto")
	}

	return &GotoStmt{Label: labelTok.Value, Loc: loc}
}

// parseExprStmt parses: expr ;
func (p *Parser) parseExprStmt() *ExprStmt {
	loc := p.currentLoc()
	expr := p.parseExpression()

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after expression")
	}

	return &ExprStmt{X: expr, Loc: loc}
}

// parseExprStmtStartingWith handles the case where we've already consumed the first identifier
func (p *Parser) parseExprStmtStartingWith(idTok Token) *ExprStmt {
	loc := SourceLoc{File: idTok.File, Line: idTok.Line}

	// Build identifier expression
	ident := &IdentExpr{
		baseExpr: baseExpr{Loc: loc},
		Name:     idTok.Value,
	}

	// Continue parsing as postfix then rest of expression
	expr := p.parsePostfixRest(ident)
	expr = p.parseBinaryRest(expr, 0)

	if _, err := p.tokens.ExpectPunct(";"); err != nil {
		p.error("expected ';' after expression")
	}

	return &ExprStmt{X: expr, Loc: loc}
}

// ============================================================
// Expression Parsing
// ============================================================

// parseExpression parses an expression
func (p *Parser) parseExpression() Expr {
	return p.parseAssignment()
}

// parseAssignment parses assignment (right-associative)
func (p *Parser) parseAssignment() Expr {
	expr := p.parseLogicalOr()

	if p.tokens.Peek().IsPunct("=") {
		loc := p.currentLoc()
		p.tokens.Next()
		rhs := p.parseAssignment()
		return &AssignExpr{
			baseExpr: baseExpr{Loc: loc},
			LHS:      expr,
			RHS:      rhs,
		}
	}

	return expr
}

// parseLogicalOr parses || (left-associative)
func (p *Parser) parseLogicalOr() Expr {
	expr := p.parseLogicalAnd()

	for p.tokens.Peek().IsPunct("||") {
		loc := p.currentLoc()
		p.tokens.Next()
		right := p.parseLogicalAnd()
		expr = &BinaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       OpLOr,
			Left:     expr,
			Right:    right,
		}
	}

	return expr
}

// parseLogicalAnd parses && (left-associative)
func (p *Parser) parseLogicalAnd() Expr {
	expr := p.parseComparison()

	for p.tokens.Peek().IsPunct("&&") {
		loc := p.currentLoc()
		p.tokens.Next()
		right := p.parseComparison()
		expr = &BinaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       OpLAnd,
			Left:     expr,
			Right:    right,
		}
	}

	return expr
}

// parseComparison parses comparison operators (non-associative)
func (p *Parser) parseComparison() Expr {
	expr := p.parseAdditive()

	tok := p.tokens.Peek()
	var op BinaryOp
	switch {
	case tok.IsPunct("=="):
		op = OpEq
	case tok.IsPunct("!="):
		op = OpNe
	case tok.IsPunct("<="):
		op = OpLe
	case tok.IsPunct(">="):
		op = OpGe
	case tok.IsPunct("<"):
		op = OpLt
	case tok.IsPunct(">"):
		op = OpGt
	default:
		return expr
	}

	loc := p.currentLoc()
	p.tokens.Next()
	right := p.parseAdditive()
	return &BinaryExpr{
		baseExpr: baseExpr{Loc: loc},
		Op:       op,
		Left:     expr,
		Right:    right,
	}
}

// parseAdditive parses + - | ^ (left-associative)
func (p *Parser) parseAdditive() Expr {
	expr := p.parseMultiplicative()

	for {
		tok := p.tokens.Peek()
		var op BinaryOp
		switch {
		case tok.IsPunct("+"):
			op = OpAdd
		case tok.IsPunct("-"):
			op = OpSub
		case tok.IsPunct("|") && !p.tokens.Peek().IsPunct("||"):
			// Need to check it's not ||
			op = OpOr
		case tok.IsPunct("^"):
			op = OpXor
		default:
			return expr
		}

		// Double-check for || (already consumed one |)
		if op == OpOr {
			p.tokens.Next()
			if p.tokens.Peek().IsPunct("|") {
				// It was ||, put back conceptually - but we can't
				// This is a parsing issue; let's handle differently
				// Actually, IsPunct("||") should match "||" as a single token
				// So this case shouldn't happen
			}
		} else {
			p.tokens.Next()
		}

		if op == OpOr {
			// Already consumed
		} else {
			// Token already consumed above
		}

		loc := p.currentLoc()
		right := p.parseMultiplicative()
		expr = &BinaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       op,
			Left:     expr,
			Right:    right,
		}
	}
}

// parseMultiplicative parses * / % & << >> (left-associative)
func (p *Parser) parseMultiplicative() Expr {
	expr := p.parseUnary()

	for {
		tok := p.tokens.Peek()
		var op BinaryOp
		switch {
		case tok.IsPunct("*"):
			op = OpMul
		case tok.IsPunct("/"):
			op = OpDiv
		case tok.IsPunct("%"):
			op = OpMod
		case tok.IsPunct("&") && !isDoubleAmp(p):
			op = OpAnd
		case tok.IsPunct("<<"):
			op = OpShl
		case tok.IsPunct(">>"):
			op = OpShr
		default:
			return expr
		}

		loc := p.currentLoc()
		p.tokens.Next()
		right := p.parseUnary()
		expr = &BinaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       op,
			Left:     expr,
			Right:    right,
		}
	}
}

// isDoubleAmp checks if current & is part of &&
func isDoubleAmp(p *Parser) bool {
	// The lexer should emit && as a single token, so if we see &, it's just &
	return p.tokens.Peek().IsPunct("&&")
}

// parseUnary parses unary operators
func (p *Parser) parseUnary() Expr {
	tok := p.tokens.Peek()
	loc := p.currentLoc()

	// Unary operators: @ & - ~ !
	if tok.IsPunct("@") {
		p.tokens.Next()
		operand := p.parseUnary()
		return &UnaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       UnaryDeref,
			Operand:  operand,
		}
	}
	if tok.IsPunct("&") {
		p.tokens.Next()
		operand := p.parseUnary()
		return &UnaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       UnaryAddr,
			Operand:  operand,
		}
	}
	if tok.IsPunct("-") {
		p.tokens.Next()
		operand := p.parseUnary()
		return &UnaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       UnaryNeg,
			Operand:  operand,
		}
	}
	if tok.IsPunct("~") {
		p.tokens.Next()
		operand := p.parseUnary()
		return &UnaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       UnaryNot,
			Operand:  operand,
		}
	}
	if tok.IsPunct("!") {
		p.tokens.Next()
		operand := p.parseUnary()
		return &UnaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       UnaryLNot,
			Operand:  operand,
		}
	}

	// sizeof
	if tok.IsKeyword("sizeof") {
		p.tokens.Next()
		if p.tokens.Peek().IsPunct("(") {
			// Could be sizeof(type) or sizeof(expr)
			p.tokens.Next() // consume '('

			// Check if it's a type
			if p.isTypeName() {
				targetType := p.parseType()
				if _, err := p.tokens.ExpectPunct(")"); err != nil {
					p.error("expected ')' after sizeof type")
				}
				return &SizeofTypeExpr{
					baseExpr:   baseExpr{Loc: loc},
					TargetType: targetType,
				}
			}

			// It's sizeof(expr)
			operand := p.parseExpression()
			if _, err := p.tokens.ExpectPunct(")"); err != nil {
				p.error("expected ')' after sizeof expression")
			}
			return &UnaryExpr{
				baseExpr: baseExpr{Loc: loc},
				Op:       UnarySizeof,
				Operand:  operand,
			}
		}
		// sizeof expr (without parens)
		operand := p.parseUnary()
		return &UnaryExpr{
			baseExpr: baseExpr{Loc: loc},
			Op:       UnarySizeof,
			Operand:  operand,
		}
	}

	// Type cast: type(expr)
	if p.isBaseTypeName(tok) {
		p.tokens.Next()
		if p.tokens.Peek().IsPunct("(") {
			p.tokens.Next()
			operand := p.parseExpression()
			if _, err := p.tokens.ExpectPunct(")"); err != nil {
				p.error("expected ')' after type cast")
			}
			return &CastExpr{
				baseExpr:   baseExpr{Loc: loc},
				TargetType: p.tokenToBaseType(tok),
				Operand:    operand,
			}
		}
		// Not a cast, error
		p.error("expected '(' for type cast")
		return nil
	}

	return p.parsePostfix()
}

// isBaseTypeName checks if token is a base type keyword
func (p *Parser) isBaseTypeName(tok Token) bool {
	if tok.Category != TokKEY {
		return false
	}
	switch tok.Value {
	case "byte", "uint8", "int16", "uint16":
		return true
	}
	return false
}

// tokenToBaseType converts a type keyword token to a Type
func (p *Parser) tokenToBaseType(tok Token) *Type {
	switch tok.Value {
	case "byte", "uint8":
		return TypeUint8Type
	case "int16":
		return TypeInt16Type
	case "uint16":
		return TypeUint16Type
	}
	return nil
}

// parsePostfix parses postfix expressions
func (p *Parser) parsePostfix() Expr {
	expr := p.parsePrimary()
	return p.parsePostfixRest(expr)
}

// parsePostfixRest parses the rest of postfix operations
func (p *Parser) parsePostfixRest(expr Expr) Expr {
	for {
		tok := p.tokens.Peek()
		loc := p.currentLoc()

		// Function call: ( args )
		if tok.IsPunct("(") {
			p.tokens.Next()
			args := make([]Expr, 0)
			if !p.tokens.Peek().IsPunct(")") {
				for {
					arg := p.parseExpression()
					args = append(args, arg)
					if !p.tokens.Peek().IsPunct(",") {
						break
					}
					p.tokens.Next()
				}
			}
			if _, err := p.tokens.ExpectPunct(")"); err != nil {
				p.error("expected ')' after arguments")
			}
			expr = &CallExpr{
				baseExpr: baseExpr{Loc: loc},
				Func:     expr,
				Args:     args,
			}
			continue
		}

		// Array subscript: [ index ]
		if tok.IsPunct("[") {
			p.tokens.Next()
			index := p.parseExpression()
			if _, err := p.tokens.ExpectPunct("]"); err != nil {
				p.error("expected ']' after index")
			}
			expr = &IndexExpr{
				baseExpr: baseExpr{Loc: loc},
				Array:    expr,
				Index:    index,
			}
			continue
		}

		// Field access: . field
		if tok.IsPunct(".") {
			p.tokens.Next()
			fieldTok, err := p.tokens.ExpectID()
			if err != nil {
				p.error("expected field name after '.'")
				return expr
			}
			expr = &FieldExpr{
				baseExpr: baseExpr{Loc: loc},
				Object:   expr,
				Field:    fieldTok.Value,
				IsArrow:  false,
			}
			continue
		}

		// Pointer field access: -> field
		if tok.IsPunct("->") {
			p.tokens.Next()
			fieldTok, err := p.tokens.ExpectID()
			if err != nil {
				p.error("expected field name after '->'")
				return expr
			}
			expr = &FieldExpr{
				baseExpr: baseExpr{Loc: loc},
				Object:   expr,
				Field:    fieldTok.Value,
				IsArrow:  true,
			}
			continue
		}

		break
	}

	return expr
}

// parsePrimary parses primary expressions
func (p *Parser) parsePrimary() Expr {
	tok := p.tokens.Peek()
	loc := p.currentLoc()

	// Identifier
	if tok.Category == TokID {
		p.tokens.Next()
		return &IdentExpr{
			baseExpr: baseExpr{Loc: loc},
			Name:     tok.Value,
		}
	}

	// Literal
	if tok.Category == TokLIT {
		p.tokens.Next()
		if strings.HasPrefix(tok.Value, "\"") {
			return &LiteralExpr{
				baseExpr: baseExpr{Loc: loc},
				Kind:     LitString,
				StrVal:   tok.Value,
			}
		}
		return &LiteralExpr{
			baseExpr: baseExpr{Loc: loc},
			Kind:     LitInt,
			IntVal:   p.parseLiteralValue(tok),
		}
	}

	// Parenthesized expression
	if tok.IsPunct("(") {
		p.tokens.Next()
		expr := p.parseExpression()
		if _, err := p.tokens.ExpectPunct(")"); err != nil {
			p.error("expected ')'")
		}
		return expr
	}

	p.error("expected expression, got %s %q", tok.Category, tok.Value)
	return nil
}

// parseLiteralValue parses a literal token value to int64
func (p *Parser) parseLiteralValue(tok Token) int64 {
	val := tok.Value

	// Handle hex: 0xNNNN
	if strings.HasPrefix(val, "0x") || strings.HasPrefix(val, "0X") {
		n, _ := strconv.ParseInt(val[2:], 16, 64)
		return n
	}

	// Handle decimal
	n, _ := strconv.ParseInt(val, 10, 64)
	return n
}

// parseBinaryRest continues parsing binary operators (used for recovery)
func (p *Parser) parseBinaryRest(expr Expr, minPrec int) Expr {
	// Simplified: just check for assignment
	if p.tokens.Peek().IsPunct("=") {
		loc := p.currentLoc()
		p.tokens.Next()
		rhs := p.parseExpression()
		return &AssignExpr{
			baseExpr: baseExpr{Loc: loc},
			LHS:      expr,
			RHS:      rhs,
		}
	}
	return expr
}
