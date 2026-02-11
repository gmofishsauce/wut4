// YAPL Semantic Analyzer - IR Generation
// Generates intermediate representation for Pass 4

package main

import (
	"bufio"
	"fmt"
	"strings"
)

// IR represents the intermediate representation
type IR struct {
	SourceFile string
	AsmDecls   []string // File-level inline assembly
	Structs    []*IRStruct
	Constants  []*IRConst
	Globals    []*IRData
	Functions  []*IRFunc
}

// IRStruct represents a struct in the IR
type IRStruct struct {
	Name   string
	Size   int
	Align  int
	Fields []*IRField
}

// IRField represents a struct field
type IRField struct {
	Name   string
	Offset int
	Type   string // IR type string
}

// IRConst represents a constant
type IRConst struct {
	Name       string
	Visibility string // PUBLIC or STATIC
	Type       string // UINT8, INT16, UINT16
	Value      int64
}

// IRData represents global data
type IRData struct {
	Name       string
	Visibility string // PUBLIC or STATIC
	Type       string // WORD, BYTE, BYTES n, etc.
	Size       int
	Init       string // optional initializer
}

// IRFunc represents a function
type IRFunc struct {
	Name       string
	Visibility string
	ReturnType string
	Params     []*IRParam
	Locals     []*IRLocal
	FrameSize  int
	Instrs     []*IRInstr
}

// IRParam represents a function parameter
type IRParam struct {
	Name  string
	Type  string
	Index int
}

// IRLocal represents a local variable
type IRLocal struct {
	Name   string
	Type   string
	Offset int
}

// IRInstr represents an IR instruction
type IRInstr struct {
	Op     string   // CONST.W, LOAD.W, ADD.W, etc.
	Dest   string   // destination (virtual register or empty)
	Args   []string // operands
	Label  string   // for LABEL instruction
	Target string   // for JUMP instructions
}

// IRGen generates IR from the analyzed AST
type IRGen struct {
	analyzer   *Analyzer
	prog       *Program
	ir         *IR
	currentFn  *IRFunc
	currentAst *FuncDef  // current function's AST
	locals     map[string]*VarDef // current function's locals and params
	tempCount  int
	labelCount int
	loopLabels []string // stack of loop end labels for break
	loopCont   []string // stack of loop continue labels
}

func newIRGen(a *Analyzer) *IRGen {
	return &IRGen{
		analyzer:   a,
		prog:       a.prog,
		ir:         &IR{SourceFile: a.prog.SourceFile},
		loopLabels: make([]string, 0),
		loopCont:   make([]string, 0),
	}
}

func (g *IRGen) newTemp() string {
	t := fmt.Sprintf("t%d", g.tempCount)
	g.tempCount++
	return t
}

func (g *IRGen) newLabel(prefix string) string {
	l := fmt.Sprintf(".%s%d", prefix, g.labelCount)
	g.labelCount++
	return l
}

func (g *IRGen) emit(op, dest string, args ...string) {
	instr := &IRInstr{Op: op, Dest: dest, Args: args}
	g.currentFn.Instrs = append(g.currentFn.Instrs, instr)
}

func (g *IRGen) emitLabel(label string) {
	instr := &IRInstr{Op: "LABEL", Label: label}
	g.currentFn.Instrs = append(g.currentFn.Instrs, instr)
}

func (g *IRGen) emitJump(target string) {
	instr := &IRInstr{Op: "JUMP", Target: target}
	g.currentFn.Instrs = append(g.currentFn.Instrs, instr)
}

func (g *IRGen) emitJumpZ(cond, target string) {
	instr := &IRInstr{Op: "JUMPZ", Args: []string{cond}, Target: target}
	g.currentFn.Instrs = append(g.currentFn.Instrs, instr)
}

func (g *IRGen) emitJumpNZ(cond, target string) {
	instr := &IRInstr{Op: "JUMPNZ", Args: []string{cond}, Target: target}
	g.currentFn.Instrs = append(g.currentFn.Instrs, instr)
}

// Generate produces the IR
func (g *IRGen) Generate() *IR {
	// Copy file-level inline assembly
	g.ir.AsmDecls = append(g.ir.AsmDecls, g.prog.AsmDecls...)

	// Generate struct definitions
	for _, s := range g.prog.Structs {
		g.genStruct(s)
	}

	// Generate constants
	for _, c := range g.prog.Constants {
		g.genConst(c)
	}

	// Generate globals
	for _, v := range g.prog.Globals {
		g.genGlobal(v)
	}

	// Generate functions
	for _, f := range g.prog.Functions {
		g.genFunc(f)
	}

	return g.ir
}

func (g *IRGen) genStruct(s *StructDef) {
	irs := &IRStruct{
		Name:   s.Name,
		Size:   s.Size,
		Align:  s.Align,
		Fields: make([]*IRField, 0, len(s.Fields)),
	}
	for _, f := range s.Fields {
		irs.Fields = append(irs.Fields, &IRField{
			Name:   f.Name,
			Offset: f.Offset,
			Type:   typeToIR(f.Type),
		})
	}
	g.ir.Structs = append(g.ir.Structs, irs)
}

func (g *IRGen) genConst(c *ConstDef) {
	// Check if this is a const array with initializer
	if c.ArrayLen != 0 || len(c.InitBytes) > 0 {
		// Treat const arrays as global data (read-only semantically)
		arrayLen := c.ArrayLen
		if arrayLen == -1 && len(c.InitBytes) > 0 {
			arrayLen = len(c.InitBytes)
		}

		irType := "WORD"
		size := 2
		if c.Type != nil {
			size = c.Type.Size()
			if arrayLen > 0 {
				if c.Type.Kind == TypeBase && c.Type.BaseType == BaseUint8 {
					irType = fmt.Sprintf("BYTES %d", arrayLen)
				} else {
					irType = fmt.Sprintf("WORDS %d", arrayLen)
				}
				size = c.Type.Size() * arrayLen
			}
		}

		var initStr string
		if len(c.InitBytes) > 0 {
			initStr = formatInitBytes(c.InitBytes)
		}

		ird := &IRData{
			Name:       c.Name,
			Visibility: visibility(c.Name),
			Type:       irType,
			Size:       size,
			Init:       initStr,
		}
		g.ir.Globals = append(g.ir.Globals, ird)
		return
	}

	// Scalar constant
	irc := &IRConst{
		Name:       c.Name,
		Visibility: visibility(c.Name),
		Type:       typeToIR(c.Type),
		Value:      c.Value,
	}
	g.ir.Constants = append(g.ir.Constants, irc)
}

func (g *IRGen) genGlobal(v *VarDef) {
	irType := "WORD"
	size := 2
	arrayLen := v.ArrayLen
	if arrayLen == -1 && len(v.InitBytes) > 0 {
		// Inferred size from initializer
		arrayLen = len(v.InitBytes)
	}
	if v.Type != nil {
		size = v.Type.Size()
		if arrayLen > 0 {
			if v.Type.Kind == TypeBase && v.Type.BaseType == BaseUint8 {
				irType = fmt.Sprintf("BYTES %d", arrayLen)
			} else {
				irType = fmt.Sprintf("WORDS %d", arrayLen)
			}
			size = v.Type.Size() * arrayLen
		} else {
			irType = typeToIRData(v.Type)
		}
	}

	// Format initializer string for IR
	var initStr string
	if len(v.InitBytes) > 0 {
		initStr = formatInitBytes(v.InitBytes)
	}

	ird := &IRData{
		Name:       v.Name,
		Visibility: visibility(v.Name),
		Type:       irType,
		Size:       size,
		Init:       initStr,
	}
	g.ir.Globals = append(g.ir.Globals, ird)
}

// formatInitBytes formats byte array as a hex string for IR output
func formatInitBytes(b []byte) string {
	var sb strings.Builder
	sb.WriteString("\"")
	for _, c := range b {
		if c >= 32 && c < 127 && c != '"' && c != '\\' {
			sb.WriteByte(c)
		} else {
			sb.WriteString(fmt.Sprintf("\\x%02X", c))
		}
	}
	sb.WriteString("\"")
	return sb.String()
}

func (g *IRGen) genFunc(f *FuncDef) {
	g.tempCount = 0
	g.currentAst = f
	g.currentFn = &IRFunc{
		Name:       f.Name,
		Visibility: visibility(f.Name),
		ReturnType: typeToIR(f.ReturnType),
		Params:     make([]*IRParam, 0, len(f.Params)),
		Locals:     make([]*IRLocal, 0, len(f.Locals)),
		FrameSize:  f.FrameSize,
		Instrs:     make([]*IRInstr, 0),
	}

	// Build locals map for identifier lookup
	g.locals = make(map[string]*VarDef)
	for _, p := range f.Params {
		g.locals[p.Name] = p
	}
	for _, l := range f.Locals {
		g.locals[l.Name] = l
	}

	// Add parameters to IR
	for i, p := range f.Params {
		g.currentFn.Params = append(g.currentFn.Params, &IRParam{
			Name:  p.Name,
			Type:  typeToIR(p.Type),
			Index: i,
		})
	}

	// Add locals to IR (convert negative offsets to positive)
	// Negative offsets are relative to original SP (before frame allocation)
	// After allocating frameSize bytes, new SP = original SP - frameSize
	// To access a local at negative offset from new SP: frameSize + offset
	for _, l := range f.Locals {
		offset := l.Offset
		if offset < 0 {
			offset = f.FrameSize + offset // Convert -6 with frameSize 12 -> 6
		}
		g.currentFn.Locals = append(g.currentFn.Locals, &IRLocal{
			Name:   l.Name,
			Type:   typeToIR(l.Type),
			Offset: offset,
		})
	}

	// Generate function body
	for _, stmt := range f.Body {
		g.genStmt(stmt)
	}

	// If function is void and last instruction isn't RETURN, add one
	if f.ReturnType.Kind == TypeVoid {
		needsReturn := true
		if len(g.currentFn.Instrs) > 0 {
			last := g.currentFn.Instrs[len(g.currentFn.Instrs)-1]
			if last.Op == "RETURN" {
				needsReturn = false
			}
		}
		if needsReturn {
			g.emit("RETURN", "")
		}
	}

	g.ir.Functions = append(g.ir.Functions, g.currentFn)
	g.currentFn = nil
}

func (g *IRGen) genStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *ExprStmt:
		if s.X != nil {
			g.genExpr(s.X)
		}

	case *ReturnStmt:
		if s.Value != nil {
			val := g.genExpr(s.Value)
			g.emit("RETURN", "", val)
		} else {
			g.emit("RETURN", "")
		}

	case *IfStmt:
		g.genIf(s)

	case *WhileStmt:
		g.genWhile(s)

	case *ForStmt:
		g.genFor(s)

	case *GotoStmt:
		g.emitJump(s.Label)

	case *LabelStmt:
		g.emitLabel(s.Label)

	case *BreakStmt:
		if len(g.loopLabels) > 0 {
			g.emitJump(g.loopLabels[len(g.loopLabels)-1])
		}

	case *ContinueStmt:
		if len(g.loopCont) > 0 {
			g.emitJump(g.loopCont[len(g.loopCont)-1])
		}

	case *AsmStmt:
		// Inline assembly - emit as ASM instruction
		instr := &IRInstr{Op: "ASM", Args: []string{s.AsmText}}
		g.currentFn.Instrs = append(g.currentFn.Instrs, instr)
	}
}

func (g *IRGen) genIf(s *IfStmt) {
	cond := g.genExpr(s.Cond)

	if len(s.Else) > 0 {
		elseLabel := g.newLabel("else")
		endLabel := g.newLabel("endif")

		g.emitJumpZ(cond, elseLabel)

		for _, st := range s.Then {
			g.genStmt(st)
		}
		g.emitJump(endLabel)

		g.emitLabel(elseLabel)
		for _, st := range s.Else {
			g.genStmt(st)
		}

		g.emitLabel(endLabel)
	} else {
		endLabel := g.newLabel("endif")
		g.emitJumpZ(cond, endLabel)

		for _, st := range s.Then {
			g.genStmt(st)
		}

		g.emitLabel(endLabel)
	}
}

func (g *IRGen) genWhile(s *WhileStmt) {
	loopLabel := g.newLabel("while")
	endLabel := g.newLabel("endwhile")

	g.loopLabels = append(g.loopLabels, endLabel)
	g.loopCont = append(g.loopCont, loopLabel)

	g.emitLabel(loopLabel)
	cond := g.genExpr(s.Cond)
	g.emitJumpZ(cond, endLabel)

	for _, st := range s.Body {
		g.genStmt(st)
	}
	g.emitJump(loopLabel)

	g.emitLabel(endLabel)

	g.loopLabels = g.loopLabels[:len(g.loopLabels)-1]
	g.loopCont = g.loopCont[:len(g.loopCont)-1]
}

func (g *IRGen) genFor(s *ForStmt) {
	loopLabel := g.newLabel("for")
	contLabel := g.newLabel("forcont")
	endLabel := g.newLabel("endfor")

	g.loopLabels = append(g.loopLabels, endLabel)
	g.loopCont = append(g.loopCont, contLabel)

	// Init
	if s.Init != nil {
		g.genExpr(s.Init)
	}

	g.emitLabel(loopLabel)

	// Condition
	if s.Cond != nil {
		cond := g.genExpr(s.Cond)
		g.emitJumpZ(cond, endLabel)
	}

	// Body
	for _, st := range s.Body {
		g.genStmt(st)
	}

	g.emitLabel(contLabel)

	// Post
	if s.Post != nil {
		g.genExpr(s.Post)
	}

	g.emitJump(loopLabel)
	g.emitLabel(endLabel)

	g.loopLabels = g.loopLabels[:len(g.loopLabels)-1]
	g.loopCont = g.loopCont[:len(g.loopCont)-1]
}

// genExpr generates IR for an expression, returns the virtual register containing the result
func (g *IRGen) genExpr(expr Expr) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *LiteralExpr:
		t := g.newTemp()
		if e.IsStr {
			// String literals would need special handling
			// For now, emit address of string data
			g.emit("ADDR", t, fmt.Sprintf("_str%d", g.tempCount))
		} else {
			g.emit("CONST.W", t, fmt.Sprintf("0x%04X", uint16(e.IntVal)))
		}
		return t

	case *IdentExpr:
		return g.genIdentLoad(e.Name)

	case *BinaryExpr:
		return g.genBinary(e)

	case *UnaryExpr:
		return g.genUnary(e)

	case *AssignExpr:
		return g.genAssign(e)

	case *CallExpr:
		return g.genCall(e)

	case *IndexExpr:
		return g.genIndex(e)

	case *FieldExpr:
		return g.genField(e)

	case *CastExpr:
		// For now, just evaluate the operand (casts are mostly no-ops at IR level)
		return g.genExpr(e.Operand)

	case *SizeofExpr:
		t := g.newTemp()
		size := 0
		if e.Target != nil {
			size = e.Target.Size()
		}
		g.emit("CONST.W", t, fmt.Sprintf("0x%04X", size))
		return t
	}

	return ""
}

func (g *IRGen) genIdentLoad(name string) string {
	t := g.newTemp()

	// Check if it's a local/param
	if g.locals != nil {
		if v, exists := g.locals[name]; exists {
			if v.IsParam {
				idx := g.paramIndex(name)
				g.emit("PARAM", t, fmt.Sprintf("%d", idx))
			} else {
				// Locals have negative offsets from SP in parser output
				// Convert to positive offset for IR: frameSize + negativeOffset
				offset := v.Offset
				if offset < 0 {
					offset = g.currentAst.FrameSize + offset
				}
				// Use LOAD.BU for byte variables
				if g.isByteType(v.Type) {
					g.emit("LOAD.BU", t, fmt.Sprintf("[SP+%d]", offset))
				} else {
					g.emit("LOAD.W", t, fmt.Sprintf("[SP+%d]", offset))
				}
			}
			return t
		}
	}

	// Check constants
	if c, exists := g.analyzer.constants[name]; exists {
		g.emit("CONST.W", t, fmt.Sprintf("0x%04X", uint16(c.Value)))
		return t
	}

	// Check globals
	if _, exists := g.analyzer.globals[name]; exists {
		g.emit("LOAD.W", t, fmt.Sprintf("[%s]", name))
		return t
	}

	return t
}

func (g *IRGen) paramIndex(name string) int {
	if g.currentAst == nil {
		return -1
	}
	for i, p := range g.currentAst.Params {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (g *IRGen) genBinary(e *BinaryExpr) string {
	left := g.genExpr(e.Left)
	right := g.genExpr(e.Right)
	t := g.newTemp()

	// Determine signedness based on operand types
	signed := false
	if e.Left != nil && e.Left.GetType() != nil {
		signed = e.Left.GetType().IsSigned()
	}

	switch e.Op {
	case OpAdd:
		g.emit("ADD.W", t, left, right)
	case OpSub:
		g.emit("SUB.W", t, left, right)
	case OpMul:
		g.emit("MUL.W", t, left, right)
	case OpDiv:
		if signed {
			g.emit("DIV.S", t, left, right)
		} else {
			g.emit("DIV.U", t, left, right)
		}
	case OpMod:
		if signed {
			g.emit("MOD.S", t, left, right)
		} else {
			g.emit("MOD.U", t, left, right)
		}
	case OpAnd:
		g.emit("AND.W", t, left, right)
	case OpOr:
		g.emit("OR.W", t, left, right)
	case OpXor:
		g.emit("XOR.W", t, left, right)
	case OpShl:
		g.emit("SHL.W", t, left, right)
	case OpShr:
		if signed {
			g.emit("SAR.W", t, left, right)
		} else {
			g.emit("SHR.W", t, left, right)
		}
	case OpEq:
		g.emit("EQ.W", t, left, right)
	case OpNe:
		g.emit("NE.W", t, left, right)
	case OpLt:
		if signed {
			g.emit("LT.S", t, left, right)
		} else {
			g.emit("LT.U", t, left, right)
		}
	case OpLe:
		if signed {
			g.emit("LE.S", t, left, right)
		} else {
			g.emit("LE.U", t, left, right)
		}
	case OpGt:
		if signed {
			g.emit("GT.S", t, left, right)
		} else {
			g.emit("GT.U", t, left, right)
		}
	case OpGe:
		if signed {
			g.emit("GE.S", t, left, right)
		} else {
			g.emit("GE.U", t, left, right)
		}
	case OpLAnd:
		// Short-circuit AND
		endLabel := g.newLabel("land")
		g.emit("CONST.W", t, "0x0000")
		g.emitJumpZ(left, endLabel)
		g.emitJumpZ(right, endLabel)
		g.emit("CONST.W", t, "0x0001")
		g.emitLabel(endLabel)
	case OpLOr:
		// Short-circuit OR
		trueLabel := g.newLabel("lor_t")
		endLabel := g.newLabel("lor_e")
		g.emitJumpNZ(left, trueLabel)
		g.emitJumpNZ(right, trueLabel)
		g.emit("CONST.W", t, "0x0000")
		g.emitJump(endLabel)
		g.emitLabel(trueLabel)
		g.emit("CONST.W", t, "0x0001")
		g.emitLabel(endLabel)
	}

	return t
}

func (g *IRGen) genUnary(e *UnaryExpr) string {
	switch e.Op {
	case OpAddr:
		// Address-of: need to get address of lvalue
		return g.genAddrOf(e.Operand)

	case OpDeref:
		// Dereference: load through pointer
		ptr := g.genExpr(e.Operand)
		t := g.newTemp()
		// Check if we're dereferencing a byte pointer - use LOAD.B
		ptrType := e.Operand.GetType()
		if ptrType != nil && ptrType.Kind == TypePointer && ptrType.Pointee != nil &&
			ptrType.Pointee.Kind == TypeBase && ptrType.Pointee.BaseType == BaseUint8 {
			g.emit("LOAD.BU", t, fmt.Sprintf("[%s]", ptr))
		} else {
			g.emit("LOAD.W", t, fmt.Sprintf("[%s]", ptr))
		}
		return t

	case OpNeg:
		operand := g.genExpr(e.Operand)
		t := g.newTemp()
		g.emit("NEG.W", t, operand)
		return t

	case OpNot:
		operand := g.genExpr(e.Operand)
		t := g.newTemp()
		g.emit("NOT.W", t, operand)
		return t

	case OpLNot:
		operand := g.genExpr(e.Operand)
		t := g.newTemp()
		g.emit("EQ.W", t, operand, "0")
		return t
	}

	return g.genExpr(e.Operand)
}

func (g *IRGen) genAddrOf(expr Expr) string {
	t := g.newTemp()

	switch e := expr.(type) {
	case *IdentExpr:
		// Check if it's a local
		if g.locals != nil {
			if v, exists := g.locals[e.Name]; exists {
				if !v.IsParam {
					// Local variable: compute SP + offset
					offset := v.Offset
					if offset < 0 {
						offset = g.currentAst.FrameSize + offset
					}
					g.emit("ADD.W", t, "SP", fmt.Sprintf("%d", offset))
					return t
				}
			}
		}
		// Global: just get its address
		g.emit("ADDR", t, e.Name)

	case *IndexExpr:
		// Address of array element
		base := g.genAddrOf(e.Array)
		idx := g.genExpr(e.Index)
		// Multiply index by element size
		elemSize := 2 // default to word
		if e.Array.GetType() != nil {
			if e.Array.GetType().Kind == TypeArray {
				elemSize = e.Array.GetType().ElemType.Size()
			} else if e.Array.GetType().Kind == TypePointer {
				elemSize = e.Array.GetType().Pointee.Size()
			}
		}
		if elemSize > 1 {
			scale := g.newTemp()
			g.emit("CONST.W", scale, fmt.Sprintf("%d", elemSize))
			offset := g.newTemp()
			g.emit("MUL.W", offset, idx, scale)
			g.emit("ADD.W", t, base, offset)
		} else {
			g.emit("ADD.W", t, base, idx)
		}

	case *FieldExpr:
		// Address of struct field
		var objAddr string
		if e.IsArrow {
			objAddr = g.genExpr(e.Object)
		} else {
			objAddr = g.genAddrOf(e.Object)
		}
		// Get field offset
		offset := g.getFieldOffset(e.Object.GetType(), e.Field)
		if offset > 0 {
			g.emit("ADD.W", t, objAddr, fmt.Sprintf("%d", offset))
		} else {
			g.emit("COPY", t, objAddr)
		}
	}

	return t
}

func (g *IRGen) getFieldOffset(objType *Type, fieldName string) int {
	if objType == nil {
		return 0
	}

	// Get the struct type
	structType := objType
	if structType.Kind == TypePointer {
		structType = structType.Pointee
	}

	if structType.Kind != TypeStruct {
		return 0
	}

	// Look up struct definition
	if structDef, exists := g.analyzer.structs[structType.Name]; exists {
		for _, f := range structDef.Fields {
			if f.Name == fieldName {
				return f.Offset
			}
		}
	}

	return 0
}

func (g *IRGen) genAssign(e *AssignExpr) string {
	rhs := g.genExpr(e.RHS)
	g.genStore(e.LHS, rhs)
	return rhs
}

// isByteType returns true if the type is a byte (uint8) type
func (g *IRGen) isByteType(t *Type) bool {
	return t != nil && t.Kind == TypeBase && t.BaseType == BaseUint8
}

// isBytePointer returns true if the type is a pointer to a byte
func (g *IRGen) isBytePointer(t *Type) bool {
	return t != nil && t.Kind == TypePointer && t.Pointee != nil &&
		t.Pointee.Kind == TypeBase && t.Pointee.BaseType == BaseUint8
}

func (g *IRGen) genStore(lhs Expr, value string) {
	switch e := lhs.(type) {
	case *IdentExpr:
		// Check if it's a local/param
		if g.locals != nil {
			if v, exists := g.locals[e.Name]; exists {
				if v.IsParam {
					// Store to parameter - emit SETPARAM for Pass 4 to handle
					idx := g.paramIndex(e.Name)
					g.emit("SETPARAM", "", fmt.Sprintf("%d", idx), value)
					return
				}
				// Locals have negative offsets from SP in parser output
				// Convert to positive offset for IR: frameSize + negativeOffset
				offset := v.Offset
				if offset < 0 {
					offset = g.currentAst.FrameSize + offset
				}
				// Check if storing to a byte variable
				if g.isByteType(v.Type) {
					g.emit("STORE.B", "", fmt.Sprintf("[SP+%d]", offset), value)
				} else {
					g.emit("STORE.W", "", fmt.Sprintf("[SP+%d]", offset), value)
				}
				return
			}
		}
		// Global
		g.emit("STORE.W", "", fmt.Sprintf("[%s]", e.Name), value)

	case *UnaryExpr:
		if e.Op == OpDeref {
			ptr := g.genExpr(e.Operand)
			// Check if dereferencing a byte pointer
			ptrType := e.Operand.GetType()
			if g.isBytePointer(ptrType) {
				g.emit("STORE.B", "", fmt.Sprintf("[%s]", ptr), value)
			} else {
				g.emit("STORE.W", "", fmt.Sprintf("[%s]", ptr), value)
			}
		}

	case *IndexExpr:
		addr := g.genAddrOf(lhs)
		// Check if the element type is a byte
		lhsType := lhs.GetType()
		if g.isByteType(lhsType) {
			g.emit("STORE.B", "", fmt.Sprintf("[%s]", addr), value)
		} else {
			g.emit("STORE.W", "", fmt.Sprintf("[%s]", addr), value)
		}

	case *FieldExpr:
		addr := g.genAddrOf(lhs)
		// Check if the field type is a byte
		lhsType := lhs.GetType()
		if g.isByteType(lhsType) {
			g.emit("STORE.B", "", fmt.Sprintf("[%s]", addr), value)
		} else {
			g.emit("STORE.W", "", fmt.Sprintf("[%s]", addr), value)
		}
	}
}

func (g *IRGen) genCall(e *CallExpr) string {
	// Generate arguments
	argRegs := make([]string, len(e.Args))
	for i, arg := range e.Args {
		argRegs[i] = g.genExpr(arg)
	}

	// Emit ARG instructions
	for i, reg := range argRegs {
		g.emit("ARG", "", fmt.Sprintf("%d", i), reg)
	}

	// Check if function returns void
	fn, exists := g.analyzer.functions[e.Func]
	if !exists || fn.ReturnType.Kind == TypeVoid {
		g.emit("CALL", "", e.Func, fmt.Sprintf("%d", len(e.Args)))
		return ""
	}

	t := g.newTemp()
	g.emit("CALL", t, e.Func, fmt.Sprintf("%d", len(e.Args)))
	return t
}

func (g *IRGen) genIndex(e *IndexExpr) string {
	addr := g.genAddrOf(e)
	t := g.newTemp()
	g.emit("LOAD.W", t, fmt.Sprintf("[%s]", addr))
	return t
}

func (g *IRGen) genField(e *FieldExpr) string {
	addr := g.genAddrOf(e)
	t := g.newTemp()
	g.emit("LOAD.W", t, fmt.Sprintf("[%s]", addr))
	return t
}

// generateIR is called from analyzer.go
func (a *Analyzer) generateIR() *IR {
	gen := newIRGen(a)
	return gen.Generate()
}

// Write outputs the IR
func (ir *IR) Write(w *bufio.Writer) {
	fmt.Fprintf(w, "#ir 1\n")
	fmt.Fprintf(w, "#source %s\n", ir.SourceFile)
	fmt.Fprintln(w)

	// Write file-level inline assembly
	for _, asm := range ir.AsmDecls {
		fmt.Fprintf(w, "ASM \"%s\"\n", asm)
	}
	if len(ir.AsmDecls) > 0 {
		fmt.Fprintln(w)
	}

	// Write structs
	for _, s := range ir.Structs {
		fmt.Fprintf(w, "STRUCT %s %d %d\n", s.Name, s.Size, s.Align)
		for _, f := range s.Fields {
			fmt.Fprintf(w, "  FIELD %s %d %s\n", f.Name, f.Offset, f.Type)
		}
		fmt.Fprintln(w, "ENDSTRUCT")
		fmt.Fprintln(w)
	}

	// Write constants
	for _, c := range ir.Constants {
		fmt.Fprintf(w, "CONST %s %s %s 0x%04X\n", c.Name, c.Visibility, c.Type, uint16(c.Value))
	}
	if len(ir.Constants) > 0 {
		fmt.Fprintln(w)
	}

	// Write globals
	for _, d := range ir.Globals {
		if d.Init != "" {
			fmt.Fprintf(w, "DATA %s %s %s %d %s\n", d.Name, d.Visibility, d.Type, d.Size, d.Init)
		} else {
			fmt.Fprintf(w, "DATA %s %s %s %d\n", d.Name, d.Visibility, d.Type, d.Size)
		}
	}
	if len(ir.Globals) > 0 {
		fmt.Fprintln(w)
	}

	// Write functions
	for _, f := range ir.Functions {
		fmt.Fprintf(w, "FUNC %s\n", f.Name)
		fmt.Fprintf(w, "  VISIBILITY %s\n", f.Visibility)
		fmt.Fprintf(w, "  RETURN %s\n", f.ReturnType)
		fmt.Fprintf(w, "  PARAMS %d\n", len(f.Params))
		for _, p := range f.Params {
			fmt.Fprintf(w, "    PARAM %s %s %d\n", p.Name, p.Type, p.Index)
		}

		localBytes := 0
		for _, l := range f.Locals {
			if l.Offset + 2 > localBytes {
				localBytes = l.Offset + 2
			}
		}
		fmt.Fprintf(w, "  LOCALS %d\n", localBytes)
		for _, l := range f.Locals {
			fmt.Fprintf(w, "    LOCAL %s %s %d\n", l.Name, l.Type, l.Offset)
		}
		fmt.Fprintf(w, "  FRAMESIZE %d\n", f.FrameSize)
		fmt.Fprintln(w)

		// Write instructions
		for _, instr := range f.Instrs {
			writeInstr(w, instr)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "ENDFUNC")
		fmt.Fprintln(w)
	}
}

func writeInstr(w *bufio.Writer, instr *IRInstr) {
	switch instr.Op {
	case "ASM":
		// Inline assembly - output verbatim
		fmt.Fprintf(w, "  ASM \"%s\"\n", instr.Args[0])
	case "LABEL":
		fmt.Fprintf(w, "%s:\n", instr.Label)
	case "JUMP":
		fmt.Fprintf(w, "  JUMP %s\n", instr.Target)
	case "JUMPZ":
		fmt.Fprintf(w, "  JUMPZ %s, %s\n", instr.Args[0], instr.Target)
	case "JUMPNZ":
		fmt.Fprintf(w, "  JUMPNZ %s, %s\n", instr.Args[0], instr.Target)
	case "RETURN":
		if len(instr.Args) > 0 && instr.Args[0] != "" {
			fmt.Fprintf(w, "  RETURN %s\n", instr.Args[0])
		} else {
			fmt.Fprintln(w, "  RETURN")
		}
	case "CALL":
		if instr.Dest != "" {
			fmt.Fprintf(w, "  %s = CALL %s, %s\n", instr.Dest, instr.Args[0], instr.Args[1])
		} else {
			fmt.Fprintf(w, "  CALL %s, %s\n", instr.Args[0], instr.Args[1])
		}
	case "ARG":
		fmt.Fprintf(w, "  ARG %s, %s\n", instr.Args[0], instr.Args[1])
	case "STORE.W", "STORE.B":
		fmt.Fprintf(w, "  %s %s, %s\n", instr.Op, instr.Args[0], instr.Args[1])
	case "SETPARAM":
		fmt.Fprintf(w, "  SETPARAM %s, %s\n", instr.Args[0], instr.Args[1])
	default:
		// Regular assignment: dest = OP args...
		if instr.Dest != "" {
			if len(instr.Args) == 1 {
				fmt.Fprintf(w, "  %s = %s %s\n", instr.Dest, instr.Op, instr.Args[0])
			} else if len(instr.Args) == 2 {
				fmt.Fprintf(w, "  %s = %s %s, %s\n", instr.Dest, instr.Op, instr.Args[0], instr.Args[1])
			} else {
				fmt.Fprintf(w, "  %s = %s\n", instr.Dest, instr.Op)
			}
		}
	}
}

func visibility(name string) string {
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		return "PUBLIC"
	}
	return "STATIC"
}

func typeToIR(t *Type) string {
	if t == nil {
		return "VOID"
	}
	switch t.Kind {
	case TypeVoid:
		return "VOID"
	case TypeBase:
		switch t.BaseType {
		case BaseUint8:
			return "UINT8"
		case BaseInt16:
			return "INT16"
		case BaseUint16:
			return "UINT16"
		case BaseBlock32:
			return "BLOCK32"
		case BaseBlock64:
			return "BLOCK64"
		case BaseBlock128:
			return "BLOCK128"
		}
	case TypePointer:
		return "PTR:" + typeToIR(t.Pointee)
	case TypeArray:
		return fmt.Sprintf("[%d]%s", t.ArrayLen, typeToIR(t.ElemType))
	case TypeStruct:
		return "STRUCT:" + t.Name
	}
	return "VOID"
}

func typeToIRData(t *Type) string {
	if t == nil {
		return "WORD"
	}
	switch t.Kind {
	case TypeBase:
		switch t.BaseType {
		case BaseUint8:
			return "BYTE"
		case BaseInt16, BaseUint16:
			return "WORD"
		case BaseBlock32:
			return "BLOCK32"
		case BaseBlock64:
			return "BLOCK64"
		case BaseBlock128:
			return "BLOCK128"
		}
	case TypePointer:
		return "WORD" // pointers are 16-bit
	case TypeStruct:
		return "STRUCT:" + t.Name
	}
	return "WORD"
}
