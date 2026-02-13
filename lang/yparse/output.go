// YAPL Parser - Output Serialization
// Serializes AST and symbol table to Pass 2 output format

package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// OutputWriter writes the AST and symbol table to the output format
type OutputWriter struct {
	w      *bufio.Writer
	indent int
}

// NewOutputWriter creates a new output writer
func NewOutputWriter(w io.Writer) *OutputWriter {
	return &OutputWriter{
		w:      bufio.NewWriter(w),
		indent: 0,
	}
}

// Flush flushes the output buffer
func (ow *OutputWriter) Flush() {
	ow.w.Flush()
}

// write writes a line with current indentation
func (ow *OutputWriter) write(format string, args ...interface{}) {
	indent := strings.Repeat("  ", ow.indent)
	fmt.Fprintf(ow.w, "%s%s\n", indent, fmt.Sprintf(format, args...))
}

// WriteProgram writes the complete program to output
func (ow *OutputWriter) WriteProgram(prog *Program, symtab *SymbolTable, filename string) {
	ow.write("#file %s", filename)
	ow.write("")

	if prog.IsBootstrap {
		ow.write("BOOTSTRAP")
		ow.write("")
	}

	// Write file-level inline assembly first
	ow.writeAsmDecls(prog)

	// Write symbol table sections
	ow.writeStructs(prog, symtab)
	ow.writeConstants(prog)
	ow.writeGlobalVars(prog, symtab)

	// Write functions
	for _, decl := range prog.Decls {
		if fd, ok := decl.(*FuncDecl); ok {
			ow.writeFunc(fd, symtab)
		}
	}
}

// writeStructs writes struct definitions
func (ow *OutputWriter) writeStructs(prog *Program, symtab *SymbolTable) {
	for _, decl := range prog.Decls {
		if sd, ok := decl.(*StructDecl); ok {
			ow.write("STRUCT %s", sd.Name)
			ow.indent++
			for _, field := range sd.Fields {
				if field.ArrayLen > 0 {
					ow.write("FIELD [%d]%s %s %d", field.ArrayLen, field.FieldType.String(), field.Name, field.Offset)
				} else {
					ow.write("FIELD %s %s %d", field.FieldType.String(), field.Name, field.Offset)
				}
			}
			ow.write("SIZE %d ALIGN %d", sd.Size, sd.Align)
			ow.indent--
			ow.write("")
		}
	}
}

// writeConstants writes constant definitions
func (ow *OutputWriter) writeConstants(prog *Program) {
	for _, decl := range prog.Decls {
		if cd, ok := decl.(*ConstDecl); ok {
			if cd.ArrayLen != 0 {
				// Const array with initializer
				initStr := ""
				if cd.Init != nil {
					if lit, ok := cd.Init.(*LiteralExpr); ok && lit.Kind == LitString {
						initStr = lit.StrVal
					}
				}
				arrayLen := cd.ArrayLen
				if arrayLen == -1 {
					arrayLen = 0 // Will be inferred by semantic analyzer
				}
				if initStr != "" {
					ow.write("CONSTARRAY %s [%d]%s INIT %s", cd.Name, arrayLen, cd.ConstType.String(), initStr)
				} else {
					ow.write("CONSTARRAY %s [%d]%s", cd.Name, arrayLen, cd.ConstType.String())
				}
			} else {
				// Scalar constant
				ow.write("CONST %s %d", cd.Name, cd.Value)
			}
		}
	}
}

// writeAsmDecls writes file-level inline assembly declarations
func (ow *OutputWriter) writeAsmDecls(prog *Program) {
	for _, decl := range prog.Decls {
		if ad, ok := decl.(*AsmDecl); ok {
			ow.write("ASM \"%s\"", ad.AsmText)
		}
	}
}

// writeGlobalVars writes global variable definitions
func (ow *OutputWriter) writeGlobalVars(prog *Program, symtab *SymbolTable) {
	for _, decl := range prog.Decls {
		if vd, ok := decl.(*VarDecl); ok {
			sym := symtab.LookupGlobal(vd.Name)
			if sym == nil {
				continue
			}
			storage := sym.Storage.String()
			// Get initializer string if present
			initStr := ""
			if vd.Init != nil {
				if lit, ok := vd.Init.(*LiteralExpr); ok && lit.Kind == LitString {
					initStr = lit.StrVal
				}
			}
			if vd.ArrayLen > 0 || vd.ArrayLen == -1 {
				// ArrayLen > 0 is explicit size, -1 is inferred
				arrayLen := vd.ArrayLen
				if arrayLen == -1 {
					arrayLen = 0 // Will be inferred by semantic analyzer
				}
				if initStr != "" {
					ow.write("VAR %s [%d]%s %s OFFSET %d INIT %s", storage, arrayLen, vd.VarType.String(), vd.Name, sym.Offset, initStr)
				} else {
					ow.write("VAR %s [%d]%s %s OFFSET %d", storage, arrayLen, vd.VarType.String(), vd.Name, sym.Offset)
				}
			} else {
				ow.write("VAR %s %s %s OFFSET %d", storage, vd.VarType.String(), vd.Name, sym.Offset)
			}
		}
	}
	ow.write("")
}

// writeFunc writes a function definition
func (ow *OutputWriter) writeFunc(fd *FuncDecl, symtab *SymbolTable) {
	sym := symtab.LookupGlobal(fd.Name)

	ow.write("FUNC %s %s", fd.ReturnType.String(), fd.Name)
	ow.indent++

	// Parameters
	if sym != nil {
		for _, param := range sym.Params {
			location := ow.paramLocation(param)
			ow.write("PARAM %s %s %s", param.Type.String(), param.Name, location)
		}
	}

	// Locals
	if sym != nil {
		for _, local := range sym.Locals {
			if local.IsConst {
				ow.write("CONST %s %d", local.Name, local.ConstVal)
			} else if local.ArrayLen > 0 || local.ArrayLen == -1 {
				// ArrayLen > 0 is explicit size, -1 is inferred (output as 0)
				arrayLen := local.ArrayLen
				if arrayLen == -1 {
					arrayLen = 0
				}
				ow.write("LOCAL [%d]%s %s OFFSET %d", arrayLen, local.Type.String(), local.Name, local.Offset)
			} else {
				ow.write("LOCAL %s %s OFFSET %d", local.Type.String(), local.Name, local.Offset)
			}
		}
	}

	// Frame size
	if sym != nil {
		ow.write("FRAMESIZE %d", sym.FrameSize)
	}

	// Body
	ow.write("BODY")
	ow.indent++
	for _, stmt := range fd.Body {
		ow.writeFuncStmt(stmt)
	}
	ow.indent--
	ow.write("END")

	ow.indent--
	ow.write("")
}

// paramLocation returns the location string for a parameter
func (ow *OutputWriter) paramLocation(param *ParamSymbol) string {
	if param.Index < 3 {
		return fmt.Sprintf("R%d", param.Index+1)
	}
	return fmt.Sprintf("STACK %d", (param.Index-3)*2)
}

// writeFuncStmt writes a function-level statement or label
func (ow *OutputWriter) writeFuncStmt(stmt FuncStmt) {
	switch s := stmt.(type) {
	case *LabelStmt:
		ow.write("LABEL %s", s.Label)
	case *AsmStmt:
		ow.write("ASM \"%s\"", s.AsmText)
	default:
		ow.writeStmt(stmt.(Stmt))
	}
}

// writeStmt writes a statement
func (ow *OutputWriter) writeStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *ExprStmt:
		if s.X != nil {
			ow.write("EXPR %d", s.Loc.Line)
			ow.indent++
			ow.writeExpr(s.X)
			ow.indent--
		}

	case *Block:
		for _, inner := range s.Stmts {
			ow.writeStmt(inner)
		}

	case *IfStmt:
		ow.write("IF %d", s.Loc.Line)
		ow.indent++
		ow.writeExpr(s.Cond)
		ow.indent--
		ow.write("THEN")
		ow.indent++
		ow.writeStmt(s.Then)
		ow.indent--
		if s.Else != nil {
			ow.write("ELSE")
			ow.indent++
			ow.writeStmt(s.Else)
			ow.indent--
		}
		ow.write("ENDIF")

	case *WhileStmt:
		ow.write("WHILE %d", s.Loc.Line)
		ow.indent++
		ow.writeExpr(s.Cond)
		ow.indent--
		ow.write("DO")
		ow.indent++
		ow.writeStmt(s.Body)
		ow.indent--
		ow.write("ENDWHILE")

	case *ForStmt:
		ow.write("FOR %d", s.Loc.Line)
		ow.indent++
		ow.write("INIT")
		ow.indent++
		if s.Init != nil {
			ow.writeExpr(s.Init)
		}
		ow.indent--
		ow.write("COND")
		ow.indent++
		if s.Cond != nil {
			ow.writeExpr(s.Cond)
		}
		ow.indent--
		ow.write("POST")
		ow.indent++
		if s.Post != nil {
			ow.writeExpr(s.Post)
		}
		ow.indent--
		ow.write("DO")
		ow.indent++
		ow.writeStmt(s.Body)
		ow.indent--
		ow.indent--
		ow.write("ENDFOR")

	case *ReturnStmt:
		if s.Value != nil {
			ow.write("RETURN %d", s.Loc.Line)
			ow.indent++
			ow.writeExpr(s.Value)
			ow.indent--
		} else {
			ow.write("RETURN %d", s.Loc.Line)
		}

	case *BreakStmt:
		ow.write("BREAK %d", s.Loc.Line)

	case *ContinueStmt:
		ow.write("CONTINUE %d", s.Loc.Line)

	case *GotoStmt:
		ow.write("GOTO %s %d", s.Label, s.Loc.Line)

	case *AsmStmt:
		ow.write("ASM \"%s\"", s.AsmText)
	}
}

// writeExpr writes an expression
func (ow *OutputWriter) writeExpr(expr Expr) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		ow.write("BINARY %s", e.Op.String())
		ow.indent++
		ow.writeExpr(e.Left)
		ow.writeExpr(e.Right)
		ow.indent--

	case *AssignExpr:
		ow.write("ASSIGN")
		ow.indent++
		ow.writeExpr(e.LHS)
		ow.writeExpr(e.RHS)
		ow.indent--

	case *UnaryExpr:
		ow.write("UNARY %s", e.Op.String())
		ow.indent++
		ow.writeExpr(e.Operand)
		ow.indent--

	case *CastExpr:
		ow.write("CAST %s", e.TargetType.String())
		ow.indent++
		ow.writeExpr(e.Operand)
		ow.indent--

	case *CallExpr:
		ow.write("CALL")
		ow.indent++
		ow.writeExpr(e.Func)
		if len(e.Args) > 0 {
			ow.write("ARGS %d", len(e.Args))
			ow.indent++
			for _, arg := range e.Args {
				ow.writeExpr(arg)
			}
			ow.indent--
		}
		ow.indent--

	case *IndexExpr:
		ow.write("INDEX")
		ow.indent++
		ow.writeExpr(e.Array)
		ow.writeExpr(e.Index)
		ow.indent--

	case *FieldExpr:
		if e.IsArrow {
			ow.write("ARROW %s", e.Field)
		} else {
			ow.write("FIELD %s", e.Field)
		}
		ow.indent++
		ow.writeExpr(e.Object)
		ow.indent--

	case *IdentExpr:
		ow.write("ID %s", e.Name)

	case *LiteralExpr:
		if e.Kind == LitString {
			ow.write("STR %s", e.StrVal)
		} else {
			ow.write("LIT %d", e.IntVal)
		}

	case *SizeofTypeExpr:
		ow.write("SIZEOF %s", e.TargetType.String())
	}
}
