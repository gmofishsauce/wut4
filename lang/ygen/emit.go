// YAPL Code Generator - Assembly Emission
// Functions to emit WUT-4 assembly output

package main

import (
	"bufio"
	"fmt"
)

// Emitter handles assembly output
type Emitter struct {
	out        *bufio.Writer
	labelCount int
}

// NewEmitter creates a new emitter
func NewEmitter(w *bufio.Writer) *Emitter {
	return &Emitter{out: w}
}

// NewLabel generates a unique label
func (e *Emitter) NewLabel(prefix string) string {
	// Labels must begin with a letter per assembler spec (not a dot)
	label := fmt.Sprintf("L_%s%d", prefix, e.labelCount)
	e.labelCount++
	return label
}

// Comment emits a comment
func (e *Emitter) Comment(format string, args ...interface{}) {
	fmt.Fprintf(e.out, "; %s\n", fmt.Sprintf(format, args...))
}

// BlankLine emits a blank line
func (e *Emitter) BlankLine() {
	fmt.Fprintln(e.out)
}

// Directive emits an assembler directive
func (e *Emitter) Directive(dir string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Fprintf(e.out, "    %s %v\n", dir, fmt.Sprint(args...))
	} else {
		fmt.Fprintf(e.out, "    %s\n", dir)
	}
}

// Label emits a label
func (e *Emitter) Label(name string) {
	fmt.Fprintf(e.out, "%s:\n", name)
}

// Raw emits raw text (for inline assembly)
func (e *Emitter) Raw(text string) {
	fmt.Fprintf(e.out, "    %s\n", text)
}

// Instr0 emits a zero-operand instruction
func (e *Emitter) Instr0(op string) {
	fmt.Fprintf(e.out, "    %s\n", op)
}

// Instr1 emits a one-operand instruction
func (e *Emitter) Instr1(op string, arg1 interface{}) {
	fmt.Fprintf(e.out, "    %s %v\n", op, arg1)
}

// Instr2 emits a two-operand instruction
func (e *Emitter) Instr2(op string, arg1, arg2 interface{}) {
	fmt.Fprintf(e.out, "    %s %v, %v\n", op, arg1, arg2)
}

// Instr3 emits a three-operand instruction
func (e *Emitter) Instr3(op string, arg1, arg2, arg3 interface{}) {
	fmt.Fprintf(e.out, "    %s %v, %v, %v\n", op, arg1, arg2, arg3)
}

// --- Specific instruction helpers ---

// Ldi emits load immediate (assembler pseudo-instruction)
func (e *Emitter) Ldi(dest int, value int) {
	e.Instr2("ldi", RegName(dest), value)
}

// LdiLabel emits load immediate with a label
func (e *Emitter) LdiLabel(dest int, label string) {
	e.Instr2("ldi", RegName(dest), label)
}

// Ldw emits load word
func (e *Emitter) Ldw(dest, base int, offset int) {
	e.Instr3("ldw", RegName(dest), RegName(base), offset)
}

// Ldb emits load byte (sign-extended)
func (e *Emitter) Ldb(dest, base int, offset int) {
	e.Instr3("ldb", RegName(dest), RegName(base), offset)
}

// Stw emits store word
func (e *Emitter) Stw(src, base int, offset int) {
	e.Instr3("stw", RegName(src), RegName(base), offset)
}

// Stb emits store byte
func (e *Emitter) Stb(src, base int, offset int) {
	e.Instr3("stb", RegName(src), RegName(base), offset)
}

// LdwLarge emits load word with potentially large offset
// Uses scratch register (r3) if offset exceeds 7-bit signed range (-64 to +63)
func (e *Emitter) LdwLarge(dest, base int, offset int, scratch int) {
	if offset >= -64 && offset <= 63 {
		e.Ldw(dest, base, offset)
	} else {
		// Compute address in scratch, then load
		e.Ldi(scratch, offset)
		e.Add(scratch, base, scratch)
		e.Ldw(dest, scratch, 0)
	}
}

// StwLarge emits store word with potentially large offset
// Uses scratch register if offset exceeds 7-bit signed range (-64 to +63)
func (e *Emitter) StwLarge(src, base int, offset int, scratch int) {
	if offset >= -64 && offset <= 63 {
		e.Stw(src, base, offset)
	} else {
		// Compute address in scratch, then store
		e.Ldi(scratch, offset)
		e.Add(scratch, base, scratch)
		e.Stw(src, scratch, 0)
	}
}

// Adi emits add immediate
func (e *Emitter) Adi(dest, src int, imm int) {
	e.Instr3("adi", RegName(dest), RegName(src), imm)
}

// Add emits add (XOP)
func (e *Emitter) Add(dest, src1, src2 int) {
	e.Instr3("add", RegName(dest), RegName(src1), RegName(src2))
}

// Sub emits subtract (XOP)
func (e *Emitter) Sub(dest, src1, src2 int) {
	e.Instr3("sub", RegName(dest), RegName(src1), RegName(src2))
}

// And emits bitwise AND (XOP)
func (e *Emitter) And(dest, src1, src2 int) {
	e.Instr3("and", RegName(dest), RegName(src1), RegName(src2))
}

// Or emits bitwise OR (XOP)
func (e *Emitter) Or(dest, src1, src2 int) {
	e.Instr3("or", RegName(dest), RegName(src1), RegName(src2))
}

// Xor emits bitwise XOR (XOP)
func (e *Emitter) Xor(dest, src1, src2 int) {
	e.Instr3("xor", RegName(dest), RegName(src1), RegName(src2))
}

// Not emits bitwise NOT (ZOP)
func (e *Emitter) Not(reg int) {
	e.Instr1("not", RegName(reg))
}

// Neg emits negate (ZOP)
func (e *Emitter) Neg(reg int) {
	e.Instr1("neg", RegName(reg))
}

// Sxt emits sign extend (ZOP)
func (e *Emitter) Sxt(reg int) {
	e.Instr1("sxt", RegName(reg))
}

// Sra emits shift right arithmetic (ZOP)
func (e *Emitter) Sra(reg int) {
	e.Instr1("sra", RegName(reg))
}

// Srl emits shift right logical (ZOP)
func (e *Emitter) Srl(reg int) {
	e.Instr1("srl", RegName(reg))
}

// Sll emits shift left logical (implemented as add r,r)
func (e *Emitter) Sll(reg int) {
	e.Add(reg, reg, reg)
}

// Mv emits move (assembler pseudo-instruction)
func (e *Emitter) Mv(dest, src int) {
	e.Instr2("mv", RegName(dest), RegName(src))
}

// Tst emits test (compare, sets flags)
func (e *Emitter) Tst(r1, r2 int) {
	e.Instr2("tst", RegName(r1), RegName(r2))
}

// Br emits unconditional branch
func (e *Emitter) Br(label string) {
	e.Instr1("br", label)
}

// Jmp emits an unconditional long jump (uses jal, full 16-bit range).
// Clobbers LINK, but LINK is saved/restored in every function prologue/epilogue.
func (e *Emitter) Jmp(label string) {
	e.Instr1("jal", label)
}

// Brz emits branch if zero
func (e *Emitter) Brz(label string) {
	e.Instr1("brz", label)
}

// Brnz emits branch if not zero
func (e *Emitter) Brnz(label string) {
	e.Instr1("brnz", label)
}

// Brslt emits branch if signed less than
func (e *Emitter) Brslt(label string) {
	e.Instr1("brslt", label)
}

// Brsge emits branch if signed greater or equal
func (e *Emitter) Brsge(label string) {
	e.Instr1("brsge", label)
}

// Brult emits branch if unsigned less than
func (e *Emitter) Brult(label string) {
	e.Instr1("brult", label)
}

// Bruge emits branch if unsigned greater or equal
func (e *Emitter) Bruge(label string) {
	e.Instr1("bruge", label)
}

// Jal emits jump and link
func (e *Emitter) Jal(label string) {
	e.Instr1("jal", label)
}

// Ret emits return (jump indirect to LINK)
func (e *Emitter) Ret() {
	e.Instr0("ret")
}

// Ji emits jump indirect
func (e *Emitter) Ji(reg int) {
	e.Instr1("ji", RegName(reg))
}

// Lsp emits load special purpose register (LINK is SPR 0)
func (e *Emitter) Lsp(dest, sprReg int) {
	e.Instr2("lsp", RegName(dest), RegName(sprReg))
}

// Ssp emits store special purpose register (LINK is SPR 0)
func (e *Emitter) Ssp(src, sprReg int) {
	e.Instr2("ssp", RegName(src), RegName(sprReg))
}

// --- Data emission helpers ---

// DataCode switches to code section
func (e *Emitter) DataCode() {
	fmt.Fprintln(e.out, ".code")
}

// DataSection switches to data section
func (e *Emitter) DataSection() {
	fmt.Fprintln(e.out, ".data")
}

// Align emits alignment directive
func (e *Emitter) Align(n int) {
	fmt.Fprintf(e.out, "    .align %d\n", n)
}

// Words emits word data
func (e *Emitter) Words(values ...int) {
	fmt.Fprint(e.out, "    .words")
	for _, v := range values {
		fmt.Fprintf(e.out, " 0x%04X", uint16(v))
	}
	fmt.Fprintln(e.out)
}

// Bytes emits byte data
func (e *Emitter) Bytes(data string) {
	fmt.Fprintf(e.out, "    .bytes %s\n", data)
}

// Space emits space reservation
func (e *Emitter) Space(n int) {
	fmt.Fprintf(e.out, "    .space %d\n", n)
}

// Flush flushes the output buffer
func (e *Emitter) Flush() {
	e.out.Flush()
}
