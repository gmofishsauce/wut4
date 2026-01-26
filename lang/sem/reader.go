// YAPL Semantic Analyzer - AST Reader
// Parses the parser's output format

package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ASTReader reads the parser's AST output
type ASTReader struct {
	scanner *bufio.Scanner
	line    string
	lineNum int
	atEOF   bool
}

// NewASTReader creates a new AST reader
func NewASTReader(r io.Reader) *ASTReader {
	return &ASTReader{
		scanner: bufio.NewScanner(r),
		lineNum: 0,
		atEOF:   false,
	}
}

func (r *ASTReader) nextLine() bool {
	if r.scanner.Scan() {
		r.line = r.scanner.Text()
		r.lineNum++
		return true
	}
	r.atEOF = true
	return false
}

func (r *ASTReader) peekLine() string {
	return r.line
}

func (r *ASTReader) error(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("line %d: %s", r.lineNum, msg)
}

// Read parses the entire AST
func (r *ASTReader) Read() (*Program, error) {
	prog := &Program{
		Structs:   make([]*StructDef, 0),
		Constants: make([]*ConstDef, 0),
		Globals:   make([]*VarDef, 0),
		Functions: make([]*FuncDef, 0),
		AsmDecls:  make([]string, 0),
	}

	for r.nextLine() {
		line := strings.TrimSpace(r.line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse file directive
		if strings.HasPrefix(line, "#file ") {
			prog.SourceFile = strings.TrimPrefix(line, "#file ")
			continue
		}

		// Parse struct
		if strings.HasPrefix(line, "STRUCT ") {
			s, err := r.readStruct()
			if err != nil {
				return nil, err
			}
			prog.Structs = append(prog.Structs, s)
			continue
		}

		// Parse constant
		if strings.HasPrefix(line, "CONST ") {
			c, err := r.readConst(line)
			if err != nil {
				return nil, err
			}
			prog.Constants = append(prog.Constants, c)
			continue
		}

		// Parse constant array
		if strings.HasPrefix(line, "CONSTARRAY ") {
			c, err := r.readConstArray(line)
			if err != nil {
				return nil, err
			}
			prog.Constants = append(prog.Constants, c)
			continue
		}

		// Parse global variable
		if strings.HasPrefix(line, "VAR ") {
			v, err := r.readGlobalVar(line)
			if err != nil {
				return nil, err
			}
			prog.Globals = append(prog.Globals, v)
			continue
		}

		// Parse function
		if strings.HasPrefix(line, "FUNC ") {
			f, err := r.readFunc(line)
			if err != nil {
				return nil, err
			}
			prog.Functions = append(prog.Functions, f)
			continue
		}

		// Parse file-level inline assembly
		if strings.HasPrefix(line, "ASM ") {
			asmText := r.extractAsmText(line)
			prog.AsmDecls = append(prog.AsmDecls, asmText)
			continue
		}
	}

	return prog, nil
}

// extractAsmText extracts the assembly text from an ASM line
// The format is: ASM "assembly text"
func (r *ASTReader) extractAsmText(line string) string {
	// Skip "ASM " prefix
	rest := strings.TrimPrefix(line, "ASM ")
	// Remove surrounding quotes
	if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
		return rest[1 : len(rest)-1]
	}
	return rest
}

func (r *ASTReader) readStruct() (*StructDef, error) {
	// Current line: "STRUCT name"
	parts := strings.Fields(r.line)
	if len(parts) < 2 {
		return nil, r.error("invalid STRUCT line")
	}

	s := &StructDef{
		Name:   parts[1],
		Fields: make([]*FieldDef, 0),
	}

	for r.nextLine() {
		line := strings.TrimSpace(r.line)

		if strings.HasPrefix(line, "FIELD ") {
			f, err := r.readField(line)
			if err != nil {
				return nil, err
			}
			s.Fields = append(s.Fields, f)
			continue
		}

		if strings.HasPrefix(line, "SIZE ") {
			// "SIZE n ALIGN m"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				s.Size, _ = strconv.Atoi(parts[1])
				s.Align, _ = strconv.Atoi(parts[3])
			}
			break
		}

		if line == "" {
			break
		}
	}

	return s, nil
}

func (r *ASTReader) readField(line string) (*FieldDef, error) {
	// "FIELD type name offset"
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil, r.error("invalid FIELD line: %s", line)
	}

	offset, _ := strconv.Atoi(parts[3])

	return &FieldDef{
		Name:   parts[2],
		Type:   parseType(parts[1]),
		Offset: offset,
	}, nil
}

func (r *ASTReader) readConst(line string) (*ConstDef, error) {
	// "CONST name value"
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, r.error("invalid CONST line: %s", line)
	}

	value, _ := strconv.ParseInt(parts[2], 0, 64)

	return &ConstDef{
		Name:  parts[1],
		Type:  Uint16Type, // constants default to uint16 for now
		Value: value,
	}, nil
}

func (r *ASTReader) readConstArray(line string) (*ConstDef, error) {
	// "CONSTARRAY name [n]type [INIT "string"]"
	// Example: "CONSTARRAY GREETING [0]byte INIT "Hello""
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, r.error("invalid CONSTARRAY line: %s", line)
	}

	name := parts[1]

	// Parse array type [n]type
	typePart := parts[2]
	var arrayLen int
	var baseTypeName string

	if strings.HasPrefix(typePart, "[") {
		idx := strings.Index(typePart, "]")
		if idx > 0 {
			arrayLen, _ = strconv.Atoi(typePart[1:idx])
			if arrayLen == 0 {
				arrayLen = -1 // Sentinel for inferred size
			}
			baseTypeName = typePart[idx+1:]
		}
	}

	// Check for INIT
	var initBytes []byte
	initIdx := strings.Index(line, " INIT ")
	if initIdx >= 0 {
		initPart := line[initIdx+6:]
		// Extract the string literal
		if strings.HasPrefix(initPart, "\"") {
			initBytes = processStringLiteral(initPart)
			// If array length is inferred (-1), set it from string length
			if arrayLen == -1 {
				arrayLen = len(initBytes)
			}
		}
	}

	return &ConstDef{
		Name:      name,
		Type:      parseType(baseTypeName),
		ArrayLen:  arrayLen,
		InitBytes: initBytes,
	}, nil
}

// processStringLiteral extracts bytes from a quoted string literal,
// handles escape sequences, and adds a null terminator.
func processStringLiteral(s string) []byte {
	// Remove surrounding quotes
	if len(s) < 2 || s[0] != '"' {
		return nil
	}

	// Find the closing quote
	endIdx := strings.LastIndex(s, "\"")
	if endIdx <= 0 {
		return nil
	}
	s = s[1:endIdx]

	result := make([]byte, 0, len(s)+1)
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '0':
				result = append(result, 0)
				i += 2
			case 'n':
				result = append(result, '\n')
				i += 2
			case 't':
				result = append(result, '\t')
				i += 2
			case 'r':
				result = append(result, '\r')
				i += 2
			case '\\':
				result = append(result, '\\')
				i += 2
			case '"':
				result = append(result, '"')
				i += 2
			case 'x':
				// \xHH hex escape
				if i+3 < len(s) {
					b, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
					if err == nil {
						result = append(result, byte(b))
						i += 4
						continue
					}
				}
				// Invalid escape, just copy
				result = append(result, s[i+1])
				i += 2
			default:
				result = append(result, s[i+1])
				i += 2
			}
		} else {
			result = append(result, s[i])
			i++
		}
	}
	// Auto-add null terminator
	result = append(result, 0)
	return result
}

func (r *ASTReader) readGlobalVar(line string) (*VarDef, error) {
	// "VAR visibility type name OFFSET n [INIT "string"]"
	// Example: "VAR STATIC int16 x OFFSET 0"
	// Example: "VAR STATIC [0]byte msg OFFSET 4 INIT "hello""
	parts := strings.Fields(line)
	if len(parts) < 6 {
		return nil, r.error("invalid VAR line: %s", line)
	}

	offset, _ := strconv.Atoi(parts[5])

	// Check for array syntax like [128]int16 or [0]byte (inferred)
	typePart := parts[2]
	var arrayLen int
	var baseTypeName string

	if strings.HasPrefix(typePart, "[") {
		// Array type: [n]type
		idx := strings.Index(typePart, "]")
		if idx > 0 {
			arrayLen, _ = strconv.Atoi(typePart[1:idx])
			if arrayLen == 0 {
				arrayLen = -1 // Sentinel for inferred size
			}
			baseTypeName = typePart[idx+1:]
		}
	} else {
		baseTypeName = typePart
	}

	// Check for INIT
	var initBytes []byte
	initIdx := strings.Index(line, " INIT ")
	if initIdx >= 0 {
		initPart := line[initIdx+6:]
		// Extract the string literal
		if strings.HasPrefix(initPart, "\"") {
			initBytes = processStringLiteral(initPart)
			// If array length is inferred (-1), set it from string length
			if arrayLen == -1 {
				arrayLen = len(initBytes)
			}
		}
	}

	return &VarDef{
		Name:      parts[3],
		Type:      parseType(baseTypeName),
		Offset:    offset,
		ArrayLen:  arrayLen,
		InitBytes: initBytes,
	}, nil
}

func (r *ASTReader) readFunc(line string) (*FuncDef, error) {
	// "FUNC returnType name"
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, r.error("invalid FUNC line: %s", line)
	}

	f := &FuncDef{
		Name:       parts[2],
		ReturnType: parseType(parts[1]),
		Params:     make([]*VarDef, 0),
		Locals:     make([]*VarDef, 0),
		Body:       make([]Stmt, 0),
	}

	// Read function header
	for r.nextLine() {
		line := strings.TrimSpace(r.line)

		if strings.HasPrefix(line, "PARAM ") {
			p, err := r.readParam(line)
			if err != nil {
				return nil, err
			}
			f.Params = append(f.Params, p)
			continue
		}

		if strings.HasPrefix(line, "LOCAL ") {
			l, err := r.readLocal(line)
			if err != nil {
				return nil, err
			}
			f.Locals = append(f.Locals, l)
			continue
		}

		if strings.HasPrefix(line, "FRAMESIZE ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				f.FrameSize, _ = strconv.Atoi(parts[1])
			}
			continue
		}

		if line == "BODY" {
			// Read function body
			body, err := r.readBody()
			if err != nil {
				return nil, err
			}
			f.Body = body
			break
		}

		if line == "" {
			continue
		}
	}

	return f, nil
}

func (r *ASTReader) readParam(line string) (*VarDef, error) {
	// "PARAM type name reg"
	// Example: "PARAM int16 a R1"
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil, r.error("invalid PARAM line: %s", line)
	}

	return &VarDef{
		Name:    parts[2],
		Type:    parseType(parts[1]),
		RegHint: parts[3],
		IsParam: true,
	}, nil
}

func (r *ASTReader) readLocal(line string) (*VarDef, error) {
	// "LOCAL type name OFFSET n"
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return nil, r.error("invalid LOCAL line: %s", line)
	}

	offset, _ := strconv.Atoi(parts[4])

	return &VarDef{
		Name:   parts[2],
		Type:   parseType(parts[1]),
		Offset: offset,
	}, nil
}

func (r *ASTReader) readBody() ([]Stmt, error) {
	stmts := make([]Stmt, 0)

	for r.nextLine() {
		line := strings.TrimSpace(r.line)

		if line == "END" {
			break
		}

		if line == "" {
			continue
		}

		stmt, err := r.readStmt(line, 0)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	return stmts, nil
}

func (r *ASTReader) readStmt(line string, depth int) (Stmt, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, nil
	}

	switch parts[0] {
	case "EXPR":
		// Expression statement: "EXPR linenum"
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		expr, err := r.readExpr(depth + 1)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{baseStmt: baseStmt{Line: lineNum}, X: expr}, nil

	case "RETURN":
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		// Check if there's a return value
		expr, _ := r.readExpr(depth + 1)
		return &ReturnStmt{baseStmt: baseStmt{Line: lineNum}, Value: expr}, nil

	case "IF":
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		return r.readIf(lineNum, depth)

	case "WHILE":
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		return r.readWhile(lineNum, depth)

	case "FOR":
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		return r.readFor(lineNum, depth)

	case "GOTO":
		lineNum := 0
		label := ""
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			label = parts[2]
		}
		return &GotoStmt{baseStmt: baseStmt{Line: lineNum}, Label: label}, nil

	case "LABEL":
		label := ""
		if len(parts) > 1 {
			label = parts[1]
		}
		return &LabelStmt{baseStmt: baseStmt{Line: 0}, Label: label}, nil

	case "BREAK":
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		return &BreakStmt{baseStmt: baseStmt{Line: lineNum}}, nil

	case "CONTINUE":
		lineNum := 0
		if len(parts) > 1 {
			lineNum, _ = strconv.Atoi(parts[1])
		}
		return &ContinueStmt{baseStmt: baseStmt{Line: lineNum}}, nil

	case "ASM":
		// Inline assembly statement
		asmText := r.extractAsmText(line)
		return &AsmStmt{baseStmt: baseStmt{Line: 0}, AsmText: asmText}, nil
	}

	return nil, nil
}

func (r *ASTReader) readIf(lineNum int, depth int) (*IfStmt, error) {
	stmt := &IfStmt{baseStmt: baseStmt{Line: lineNum}}

	// Read condition
	cond, err := r.readExpr(depth + 1)
	if err != nil {
		return nil, err
	}
	stmt.Cond = cond

	// Read then block
	stmt.Then = make([]Stmt, 0)
	for r.nextLine() {
		line := strings.TrimSpace(r.line)
		if line == "ELSE" || line == "ENDIF" {
			if line == "ELSE" {
				stmt.Else = make([]Stmt, 0)
				for r.nextLine() {
					line = strings.TrimSpace(r.line)
					if line == "ENDIF" {
						break
					}
					s, _ := r.readStmt(line, depth+1)
					if s != nil {
						stmt.Else = append(stmt.Else, s)
					}
				}
			}
			break
		}
		s, _ := r.readStmt(line, depth+1)
		if s != nil {
			stmt.Then = append(stmt.Then, s)
		}
	}

	return stmt, nil
}

func (r *ASTReader) readWhile(lineNum int, depth int) (*WhileStmt, error) {
	stmt := &WhileStmt{baseStmt: baseStmt{Line: lineNum}}

	// Read condition
	cond, err := r.readExpr(depth + 1)
	if err != nil {
		return nil, err
	}
	stmt.Cond = cond

	// Read body
	stmt.Body = make([]Stmt, 0)
	for r.nextLine() {
		line := strings.TrimSpace(r.line)
		if line == "ENDWHILE" {
			break
		}
		s, _ := r.readStmt(line, depth+1)
		if s != nil {
			stmt.Body = append(stmt.Body, s)
		}
	}

	return stmt, nil
}

func (r *ASTReader) readFor(lineNum int, depth int) (*ForStmt, error) {
	stmt := &ForStmt{baseStmt: baseStmt{Line: lineNum}}

	// Read init, cond, post (each on separate lines, may be EMPTY)
	for i := 0; i < 3; i++ {
		r.nextLine()
		line := strings.TrimSpace(r.line)
		if line != "EMPTY" {
			expr, _ := r.readExprFromLine(line, depth+1)
			switch i {
			case 0:
				stmt.Init = expr
			case 1:
				stmt.Cond = expr
			case 2:
				stmt.Post = expr
			}
		}
	}

	// Read body
	stmt.Body = make([]Stmt, 0)
	for r.nextLine() {
		line := strings.TrimSpace(r.line)
		if line == "ENDFOR" {
			break
		}
		s, _ := r.readStmt(line, depth+1)
		if s != nil {
			stmt.Body = append(stmt.Body, s)
		}
	}

	return stmt, nil
}

func (r *ASTReader) readExpr(depth int) (Expr, error) {
	if !r.nextLine() {
		return nil, nil
	}
	return r.readExprFromLine(strings.TrimSpace(r.line), depth)
}

func (r *ASTReader) readExprFromLine(line string, depth int) (Expr, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, nil
	}

	switch parts[0] {
	case "LIT":
		val := int64(0)
		if len(parts) > 1 {
			val, _ = strconv.ParseInt(parts[1], 0, 64)
		}
		return &LiteralExpr{IntVal: val}, nil

	case "STRLIT":
		// String literal - rest of line is the string
		str := ""
		if len(parts) > 1 {
			str = strings.Join(parts[1:], " ")
		}
		return &LiteralExpr{StrVal: str, IsStr: true}, nil

	case "ID":
		name := ""
		if len(parts) > 1 {
			name = parts[1]
		}
		return &IdentExpr{Name: name}, nil

	case "BINARY":
		op := parseBinaryOp(parts[1])
		left, _ := r.readExpr(depth + 1)
		right, _ := r.readExpr(depth + 1)
		return &BinaryExpr{Op: op, Left: left, Right: right}, nil

	case "UNARY":
		op := parseUnaryOp(parts[1])
		operand, _ := r.readExpr(depth + 1)
		return &UnaryExpr{Op: op, Operand: operand}, nil

	case "ASSIGN":
		lhs, _ := r.readExpr(depth + 1)
		rhs, _ := r.readExpr(depth + 1)
		return &AssignExpr{LHS: lhs, RHS: rhs}, nil

	case "CALL":
		funcExpr, _ := r.readExpr(depth + 1)
		funcName := ""
		if id, ok := funcExpr.(*IdentExpr); ok {
			funcName = id.Name
		}
		args := make([]Expr, 0)
		// Read ARGS
		r.nextLine()
		argsLine := strings.TrimSpace(r.line)
		if argsLine == "ARGS" {
			// Read arguments until we hit something that's not an expression
			for {
				r.nextLine()
				argLine := strings.TrimSpace(r.line)
				// Check if this looks like an expression
				argParts := strings.Fields(argLine)
				if len(argParts) == 0 {
					break
				}
				if argParts[0] == "LIT" || argParts[0] == "ID" || argParts[0] == "BINARY" ||
					argParts[0] == "UNARY" || argParts[0] == "CALL" || argParts[0] == "INDEX" ||
					argParts[0] == "FIELD" || argParts[0] == "STRLIT" {
					arg, _ := r.readExprFromLine(argLine, depth+1)
					if arg != nil {
						args = append(args, arg)
					}
				} else {
					break
				}
			}
		}
		return &CallExpr{Func: funcName, Args: args}, nil

	case "INDEX":
		array, _ := r.readExpr(depth + 1)
		index, _ := r.readExpr(depth + 1)
		return &IndexExpr{Array: array, Index: index}, nil

	case "FIELD":
		isArrow := len(parts) > 1 && parts[1] == "ARROW"
		obj, _ := r.readExpr(depth + 1)
		r.nextLine()
		fieldLine := strings.TrimSpace(r.line)
		fieldParts := strings.Fields(fieldLine)
		fieldName := ""
		if len(fieldParts) > 1 {
			fieldName = fieldParts[1]
		}
		return &FieldExpr{Object: obj, Field: fieldName, IsArrow: isArrow}, nil

	case "CAST":
		targetType := parseType(parts[1])
		operand, _ := r.readExpr(depth + 1)
		return &CastExpr{Target: targetType, Operand: operand}, nil

	case "SIZEOF":
		targetType := parseType(parts[1])
		return &SizeofExpr{Target: targetType}, nil
	}

	return nil, nil
}

func parseType(s string) *Type {
	// Handle pointer prefix
	if strings.HasPrefix(s, "@") {
		pointee := parseType(s[1:])
		return &Type{Kind: TypePointer, Pointee: pointee}
	}

	// Handle array syntax [n]type
	if strings.HasPrefix(s, "[") {
		idx := strings.Index(s, "]")
		if idx > 0 {
			n, _ := strconv.Atoi(s[1:idx])
			elem := parseType(s[idx+1:])
			return &Type{Kind: TypeArray, ElemType: elem, ArrayLen: n}
		}
	}

	// Base types
	switch strings.ToLower(s) {
	case "void":
		return VoidType
	case "byte", "uint8":
		return Uint8Type
	case "int16":
		return Int16Type
	case "uint16":
		return Uint16Type
	case "block32":
		return Block32Type
	case "block64":
		return Block64Type
	case "block128":
		return Block128Type
	}

	// Struct type
	return &Type{Kind: TypeStruct, Name: s}
}

func parseBinaryOp(s string) BinaryOp {
	switch s {
	case "ADD":
		return OpAdd
	case "SUB":
		return OpSub
	case "MUL":
		return OpMul
	case "DIV":
		return OpDiv
	case "MOD":
		return OpMod
	case "AND":
		return OpAnd
	case "OR":
		return OpOr
	case "XOR":
		return OpXor
	case "SHL":
		return OpShl
	case "SHR":
		return OpShr
	case "EQ":
		return OpEq
	case "NE":
		return OpNe
	case "LT":
		return OpLt
	case "LE":
		return OpLe
	case "GT":
		return OpGt
	case "GE":
		return OpGe
	case "LAND":
		return OpLAnd
	case "LOR":
		return OpLOr
	default:
		return OpAdd
	}
}

func parseUnaryOp(s string) UnaryOp {
	switch s {
	case "NEG":
		return OpNeg
	case "NOT":
		return OpNot
	case "LNOT":
		return OpLNot
	case "ADDR":
		return OpAddr
	case "DEREF":
		return OpDeref
	default:
		return OpNeg
	}
}
