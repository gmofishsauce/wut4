// YAPL Semantic Analyzer - Core Analysis and Type Checking
// Performs semantic analysis and prepares for IR generation

package main

import (
	"fmt"
)

// Analyzer performs semantic analysis
type Analyzer struct {
	prog       *Program
	errors     []string
	structs    map[string]*StructDef
	globals    map[string]*VarDef
	constants  map[string]*ConstDef
	functions  map[string]*FuncDef
	currentFn  *FuncDef
	locals     map[string]*VarDef // current function's locals + params
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(prog *Program) *Analyzer {
	return &Analyzer{
		prog:      prog,
		errors:    make([]string, 0),
		structs:   make(map[string]*StructDef),
		globals:   make(map[string]*VarDef),
		constants: make(map[string]*ConstDef),
		functions: make(map[string]*FuncDef),
	}
}

func (a *Analyzer) error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	a.errors = append(a.errors, msg)
}

func (a *Analyzer) errorAt(file string, line int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	a.errors = append(a.errors, fmt.Sprintf("%s:%d: error: %s", file, line, msg))
}

// Analyze performs semantic analysis and generates IR
func (a *Analyzer) Analyze() (*IR, []string) {
	// Phase 1: Build symbol tables
	a.buildSymbolTables()

	// Phase 2: Type check all functions
	a.typeCheck()

	// If there were errors, don't generate IR
	if len(a.errors) > 0 {
		return nil, a.errors
	}

	// Phase 3: Generate IR
	ir := a.generateIR()

	return ir, a.errors
}

// Phase 1: Build symbol tables
func (a *Analyzer) buildSymbolTables() {
	// Register structs
	for _, s := range a.prog.Structs {
		if _, exists := a.structs[s.Name]; exists {
			a.error("duplicate struct definition: %s", s.Name)
		}
		a.structs[s.Name] = s
	}

	// Register constants
	for _, c := range a.prog.Constants {
		if _, exists := a.constants[c.Name]; exists {
			a.error("duplicate constant definition: %s", c.Name)
		}
		a.constants[c.Name] = c
	}

	// Register global variables
	for _, v := range a.prog.Globals {
		if _, exists := a.globals[v.Name]; exists {
			a.error("duplicate global variable: %s", v.Name)
		}
		a.globals[v.Name] = v
	}

	// Register functions
	for _, f := range a.prog.Functions {
		if _, exists := a.functions[f.Name]; exists {
			a.error("duplicate function definition: %s", f.Name)
		}
		a.functions[f.Name] = f
	}
}

// Phase 2: Type checking
func (a *Analyzer) typeCheck() {
	for _, f := range a.prog.Functions {
		a.typeCheckFunc(f)
	}
}

func (a *Analyzer) typeCheckFunc(f *FuncDef) {
	a.currentFn = f

	// Build local symbol table
	a.locals = make(map[string]*VarDef)

	// Add parameters
	for _, p := range f.Params {
		if _, exists := a.locals[p.Name]; exists {
			a.errorAt(a.prog.SourceFile, f.Line, "duplicate parameter: %s", p.Name)
		}
		a.locals[p.Name] = p
	}

	// Add locals
	for _, l := range f.Locals {
		if _, exists := a.locals[l.Name]; exists {
			a.errorAt(a.prog.SourceFile, f.Line, "duplicate local variable: %s", l.Name)
		}
		a.locals[l.Name] = l
	}

	// Type check statements
	for _, stmt := range f.Body {
		a.typeCheckStmt(stmt)
	}

	a.currentFn = nil
	a.locals = nil
}

func (a *Analyzer) typeCheckStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *ExprStmt:
		if s.X != nil {
			a.typeCheckExpr(s.X)
		}

	case *ReturnStmt:
		if s.Value != nil {
			a.typeCheckExpr(s.Value)
			// Check return type matches
			retType := s.Value.GetType()
			if retType != nil && a.currentFn != nil {
				if !a.typesCompatible(a.currentFn.ReturnType, retType) {
					a.errorAt(a.prog.SourceFile, s.Line, "return type mismatch")
				}
			}
		} else if a.currentFn != nil && a.currentFn.ReturnType.Kind != TypeVoid {
			a.errorAt(a.prog.SourceFile, s.Line, "non-void function must return a value")
		}

	case *IfStmt:
		a.typeCheckExpr(s.Cond)
		for _, st := range s.Then {
			a.typeCheckStmt(st)
		}
		for _, st := range s.Else {
			a.typeCheckStmt(st)
		}

	case *WhileStmt:
		a.typeCheckExpr(s.Cond)
		for _, st := range s.Body {
			a.typeCheckStmt(st)
		}

	case *ForStmt:
		if s.Init != nil {
			a.typeCheckExpr(s.Init)
		}
		if s.Cond != nil {
			a.typeCheckExpr(s.Cond)
		}
		if s.Post != nil {
			a.typeCheckExpr(s.Post)
		}
		for _, st := range s.Body {
			a.typeCheckStmt(st)
		}

	case *GotoStmt:
		// Label checking would go here

	case *LabelStmt:
		// Label registration would go here

	case *BreakStmt, *ContinueStmt:
		// Loop context checking would go here
	}
}

func (a *Analyzer) typeCheckExpr(expr Expr) *Type {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *LiteralExpr:
		if e.IsStr {
			// String literal is pointer to byte
			t := &Type{Kind: TypePointer, Pointee: Uint8Type}
			e.SetType(t)
			return t
		}
		// Integer literal - default to int16 for now
		e.SetType(Int16Type)
		return Int16Type

	case *IdentExpr:
		// Look up identifier
		t := a.lookupType(e.Name)
		if t == nil {
			a.error("undefined identifier: %s", e.Name)
			return nil
		}
		e.SetType(t)
		return t

	case *BinaryExpr:
		leftType := a.typeCheckExpr(e.Left)
		rightType := a.typeCheckExpr(e.Right)

		if leftType == nil || rightType == nil {
			return nil
		}

		// Check operand compatibility
		if !a.typesCompatible(leftType, rightType) {
			a.error("type mismatch in binary expression")
		}

		// Comparison operators return int16 (0 or 1)
		switch e.Op {
		case OpEq, OpNe, OpLt, OpLe, OpGt, OpGe, OpLAnd, OpLOr:
			e.SetType(Int16Type)
			return Int16Type
		}

		e.SetType(leftType)
		return leftType

	case *UnaryExpr:
		operandType := a.typeCheckExpr(e.Operand)
		if operandType == nil {
			return nil
		}

		switch e.Op {
		case OpAddr:
			// Address-of: result is pointer to operand type
			t := &Type{Kind: TypePointer, Pointee: operandType}
			e.SetType(t)
			return t
		case OpDeref:
			// Dereference: operand must be pointer
			if operandType.Kind != TypePointer {
				a.error("cannot dereference non-pointer type")
				return nil
			}
			t := operandType.Pointee
			e.SetType(t)
			return t
		case OpNeg, OpNot, OpLNot:
			e.SetType(operandType)
			return operandType
		}
		return operandType

	case *AssignExpr:
		lhsType := a.typeCheckExpr(e.LHS)
		rhsType := a.typeCheckExpr(e.RHS)

		if lhsType == nil || rhsType == nil {
			return nil
		}

		if !a.typesCompatible(lhsType, rhsType) {
			a.error("type mismatch in assignment")
		}

		e.SetType(lhsType)
		return lhsType

	case *CallExpr:
		// Look up function
		fn, exists := a.functions[e.Func]
		if !exists {
			a.error("undefined function: %s", e.Func)
			return nil
		}

		// Check argument count
		if len(e.Args) != len(fn.Params) {
			a.error("wrong number of arguments to %s: expected %d, got %d",
				e.Func, len(fn.Params), len(e.Args))
		}

		// Type check arguments
		for i, arg := range e.Args {
			argType := a.typeCheckExpr(arg)
			if i < len(fn.Params) && argType != nil {
				if !a.typesCompatible(fn.Params[i].Type, argType) {
					a.error("argument %d type mismatch in call to %s", i+1, e.Func)
				}
			}
		}

		e.SetType(fn.ReturnType)
		return fn.ReturnType

	case *IndexExpr:
		arrayType := a.typeCheckExpr(e.Array)
		indexType := a.typeCheckExpr(e.Index)

		if arrayType == nil {
			return nil
		}

		// Index must be integral
		if indexType != nil && !indexType.IsIntegral() {
			a.error("array index must be integral type")
		}

		// Array must be array or pointer type
		var elemType *Type
		if arrayType.Kind == TypeArray {
			elemType = arrayType.ElemType
		} else if arrayType.Kind == TypePointer {
			elemType = arrayType.Pointee
		} else {
			a.error("cannot index non-array/non-pointer type")
			return nil
		}

		e.SetType(elemType)
		return elemType

	case *FieldExpr:
		objType := a.typeCheckExpr(e.Object)
		if objType == nil {
			return nil
		}

		// For ->, object must be pointer to struct
		if e.IsArrow {
			if objType.Kind != TypePointer {
				a.error("-> requires pointer type")
				return nil
			}
			objType = objType.Pointee
		}

		// Object must be struct type
		if objType.Kind != TypeStruct {
			a.error("field access requires struct type")
			return nil
		}

		// Look up struct
		structDef, exists := a.structs[objType.Name]
		if !exists {
			a.error("undefined struct: %s", objType.Name)
			return nil
		}

		// Look up field
		for _, f := range structDef.Fields {
			if f.Name == e.Field {
				e.SetType(f.Type)
				return f.Type
			}
		}

		a.error("struct %s has no field %s", objType.Name, e.Field)
		return nil

	case *CastExpr:
		a.typeCheckExpr(e.Operand)
		e.SetType(e.Target)
		return e.Target

	case *SizeofExpr:
		e.SetType(Uint16Type)
		return Uint16Type
	}

	return nil
}

func (a *Analyzer) lookupType(name string) *Type {
	// Check locals first
	if a.locals != nil {
		if v, exists := a.locals[name]; exists {
			if v.ArrayLen > 0 {
				return &Type{Kind: TypeArray, ElemType: v.Type, ArrayLen: v.ArrayLen}
			}
			return v.Type
		}
	}

	// Check constants (including const arrays)
	if c, exists := a.constants[name]; exists {
		if c.ArrayLen > 0 {
			return &Type{Kind: TypeArray, ElemType: c.Type, ArrayLen: c.ArrayLen}
		}
		return c.Type
	}

	// Check globals
	if v, exists := a.globals[name]; exists {
		if v.ArrayLen > 0 {
			return &Type{Kind: TypeArray, ElemType: v.Type, ArrayLen: v.ArrayLen}
		}
		return v.Type
	}

	// Check functions (for function pointers, though YAPL may not support these)
	if _, exists := a.functions[name]; exists {
		// Functions are not first-class values in YAPL
		return nil
	}

	return nil
}

func (a *Analyzer) typesCompatible(t1, t2 *Type) bool {
	if t1 == nil || t2 == nil {
		return false
	}

	if t1.Kind != t2.Kind {
		// Allow integral types to mix (with warning ideally)
		if t1.IsIntegral() && t2.IsIntegral() {
			return true
		}
		return false
	}

	switch t1.Kind {
	case TypeVoid:
		return true
	case TypeBase:
		return t1.BaseType == t2.BaseType
	case TypePointer:
		// Allow pointer to void to match any pointer
		if t1.Pointee.Kind == TypeVoid || t2.Pointee.Kind == TypeVoid {
			return true
		}
		return a.typesCompatible(t1.Pointee, t2.Pointee)
	case TypeArray:
		return t1.ArrayLen == t2.ArrayLen && a.typesCompatible(t1.ElemType, t2.ElemType)
	case TypeStruct:
		return t1.Name == t2.Name
	}

	return false
}
