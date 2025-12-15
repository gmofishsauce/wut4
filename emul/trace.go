// Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.

package main

import (
	"fmt"
	"io"
)

// Tracer generates detailed execution traces
type Tracer struct {
	out                io.Writer
	prevRegs           [8]uint16
	prevFlags          uint16
	currentInstruction *Instruction
}

// NewTracer creates a new tracer writing to the given output
func NewTracer(out io.Writer) *Tracer {
	return &Tracer{
		out: out,
	}
}

// TracePreInstruction traces state before instruction execution
func (t *Tracer) TracePreInstruction(cpu *CPU) {
	// Save register state before execution
	copy(t.prevRegs[:], cpu.gen[cpu.mode][:])
	t.prevFlags = cpu.getFlags()

	// Print cycle and PC
	fmt.Fprintf(t.out, "\n")
	fmt.Fprintf(t.out, "========================================\n")
	fmt.Fprintf(t.out, "CYCLE: %016d\n", cpu.cycles)

	mode := "kernel"
	if cpu.mode == ModeUser {
		mode = "user"
	}
	fmt.Fprintf(t.out, "PC: 0x%04X [%s] [context=%d]\n", cpu.pc, mode, cpu.context)

	// Fetch and disassemble instruction
	physAddr, err := cpu.translateCode(cpu.pc)
	if err == nil {
		inst := cpu.physMem[physAddr]
		decoded := decode(inst)
		t.currentInstruction = decoded

		fmt.Fprintf(t.out, "INST: 0x%04X  %s\n", inst, disassemble(decoded))
		fmt.Fprintf(t.out, "DECODE: %s\n", t.formatDecode(decoded))
	} else {
		fmt.Fprintf(t.out, "INST: <page fault fetching instruction>\n")
	}

	// Print registers before
	fmt.Fprintf(t.out, "REGS BEFORE: ")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(t.out, "r%d=%04X ", i, cpu.gen[cpu.mode][i])
	}
	fmt.Fprintf(t.out, "\n")

	// Print flags before
	c := (t.prevFlags & FLAG_C) != 0
	z := (t.prevFlags & FLAG_Z) != 0
	n := (t.prevFlags & FLAG_N) != 0
	v := (t.prevFlags & FLAG_V) != 0
	fmt.Fprintf(t.out, "FLAGS BEFORE: C=%d Z=%d N=%d V=%d\n",
		boolToInt(c), boolToInt(z), boolToInt(n), boolToInt(v))
}

// TracePostInstruction traces state after instruction execution
func (t *Tracer) TracePostInstruction(cpu *CPU, inst *Instruction) {
	// Print what happened during execution (if instruction modifies registers)
	changed := false
	for i := 0; i < 8; i++ {
		if cpu.gen[cpu.mode][i] != t.prevRegs[i] {
			changed = true
			break
		}
	}

	if changed || cpu.getFlags() != t.prevFlags {
		fmt.Fprintf(t.out, "EXECUTE: ")
		for i := 0; i < 8; i++ {
			if cpu.gen[cpu.mode][i] != t.prevRegs[i] {
				fmt.Fprintf(t.out, "r%d ← 0x%04X ", i, cpu.gen[cpu.mode][i])
			}
		}
		fmt.Fprintf(t.out, "\n")
	}

	// Print flags after
	flags := cpu.getFlags()
	if flags != t.prevFlags {
		c := (flags & FLAG_C) != 0
		z := (flags & FLAG_Z) != 0
		n := (flags & FLAG_N) != 0
		v := (flags & FLAG_V) != 0
		fmt.Fprintf(t.out, "FLAGS AFTER: C=%d Z=%d N=%d V=%d\n",
			boolToInt(c), boolToInt(z), boolToInt(n), boolToInt(v))
	}

	// Print registers after
	fmt.Fprintf(t.out, "REGS AFTER: ")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(t.out, "r%d=%04X ", i, cpu.gen[cpu.mode][i])
	}
	fmt.Fprintf(t.out, "\n")
}

// TraceMemoryRead traces a memory read operation
func (t *Tracer) TraceMemoryRead(vaddr uint16, paddr uint32, value uint16, isByte bool) {
	if isByte {
		fmt.Fprintf(t.out, "MEM READ BYTE: vaddr=0x%04X → paddr=0x%06X value=0x%02X\n",
			vaddr, paddr, value&0xFF)
	} else {
		fmt.Fprintf(t.out, "MEM READ WORD: vaddr=0x%04X → paddr=0x%06X value=0x%04X\n",
			vaddr, paddr, value)
	}
}

// TraceMemoryWrite traces a memory write operation
func (t *Tracer) TraceMemoryWrite(vaddr uint16, paddr uint32, value uint16, isByte bool) {
	if isByte {
		fmt.Fprintf(t.out, "MEM WRITE BYTE: vaddr=0x%04X → paddr=0x%06X value=0x%02X\n",
			vaddr, paddr, value&0xFF)
	} else {
		fmt.Fprintf(t.out, "MEM WRITE WORD: vaddr=0x%04X → paddr=0x%06X value=0x%04X\n",
			vaddr, paddr, value)
	}
}

// TraceException traces an exception
func (t *Tracer) TraceException(cpu *CPU, vector uint16, data uint16) {
	fmt.Fprintf(t.out, "\n*** EXCEPTION: vector=0x%04X data=0x%04X\n", vector, data)
	fmt.Fprintf(t.out, "    Saved PC (IRR) ← 0x%04X\n", cpu.spr[ModeKernel][SPR_IRR])
	fmt.Fprintf(t.out, "    Cause (ICR) ← 0x%04X\n", cpu.spr[ModeKernel][SPR_ICR])
	fmt.Fprintf(t.out, "    Data (IDR) ← 0x%04X\n", cpu.spr[ModeKernel][SPR_IDR])
	fmt.Fprintf(t.out, "    Previous mode (ISR) ← %d\n", cpu.spr[ModeKernel][SPR_ISR])
}

// TraceModeSwitch traces a mode change
func (t *Tracer) TraceModeSwitch(cpu *CPU, fromMode uint8, toMode uint8) {
	fromName := "kernel"
	if fromMode == ModeUser {
		fromName = "user"
	}
	toName := "kernel"
	if toMode == ModeUser {
		toName = "user"
	}
	fmt.Fprintf(t.out, "MODE SWITCH: %s → %s\n", fromName, toName)
}

// TraceSPRRead traces a special register read
func (t *Tracer) TraceSPRRead(spr uint16, value uint16) {
	fmt.Fprintf(t.out, "SPR READ: spr=%d (%s) value=0x%04X\n", spr, t.sprName(spr), value)
}

// TraceSPRWrite traces a special register write
func (t *Tracer) TraceSPRWrite(spr uint16, value uint16) {
	fmt.Fprintf(t.out, "SPR WRITE: spr=%d (%s) value=0x%04X\n", spr, t.sprName(spr), value)
}

// TraceConsoleInput traces console input
func (t *Tracer) TraceConsoleInput(value uint16) {
	if value >= 32 && value < 127 {
		fmt.Fprintf(t.out, "CONSOLE INPUT: 0x%02X '%c'\n", value, byte(value))
	} else {
		fmt.Fprintf(t.out, "CONSOLE INPUT: 0x%02X\n", value)
	}
}

// TraceConsoleOutput traces console output
func (t *Tracer) TraceConsoleOutput(value uint16) {
	if value >= 32 && value < 127 {
		fmt.Fprintf(t.out, "CONSOLE OUTPUT: 0x%02X '%c'\n", value, byte(value))
	} else {
		fmt.Fprintf(t.out, "CONSOLE OUTPUT: 0x%02X\n", value)
	}
}

// TraceDoubleFault traces a double fault (fault in kernel mode)
func (t *Tracer) TraceDoubleFault(cpu *CPU, vector uint16, data uint16) {
	fmt.Fprintf(t.out, "\n")
	fmt.Fprintf(t.out, "========================================\n")
	fmt.Fprintf(t.out, "*** DOUBLE FAULT ***\n")
	fmt.Fprintf(t.out, "========================================\n")
	fmt.Fprintf(t.out, "Exception occurred in kernel mode\n")
	fmt.Fprintf(t.out, "Vector: 0x%04X\n", vector)
	fmt.Fprintf(t.out, "Data: 0x%04X\n", data)
	fmt.Fprintf(t.out, "PC: 0x%04X\n", cpu.pc)
	fmt.Fprintf(t.out, "Cycles: %d\n", cpu.cycles)
	fmt.Fprintf(t.out, "Interrupts enabled: %v\n", cpu.intEnabled)
	fmt.Fprintf(t.out, "\nKernel registers:\n")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(t.out, "  r%d = 0x%04X\n", i, cpu.gen[ModeKernel][i])
	}
	fmt.Fprintf(t.out, "\nEmulator halting.\n")
	fmt.Fprintf(t.out, "========================================\n")
}

// Helper methods

func (t *Tracer) formatDecode(inst *Instruction) string {
	if inst.isBase {
		return fmt.Sprintf("op=%d rA=%d rB=%d imm=%d", inst.opcode, inst.rA, inst.rB, inst.imm7)
	} else if inst.isXOP {
		return fmt.Sprintf("xop=%d rA=%d rB=%d rC=%d", inst.xop, inst.rA, inst.rB, inst.rC)
	} else if inst.isYOP {
		return fmt.Sprintf("yop=%d rA=%d rB=%d", inst.yop, inst.rA, inst.rB)
	} else if inst.isZOP {
		return fmt.Sprintf("zop=%d rA=%d", inst.zop, inst.rA)
	} else if inst.isVOP {
		return fmt.Sprintf("vop=%d", inst.vop)
	}
	return "unknown"
}

func (t *Tracer) sprName(spr uint16) string {
	switch spr {
	case SPR_LINK:
		return "LINK"
	case SPR_FLAGS:
		return "FLAGS"
	case SPR_CYCLO:
		return "CYCLO"
	case SPR_CYCHI:
		return "CYCHI"
	case SPR_IRR:
		return "IRR"
	case SPR_ICR:
		return "ICR"
	case SPR_IDR:
		return "IDR"
	case SPR_ISR:
		return "ISR"
	case SPR_CONTEXT:
		return "CONTEXT"
	case SPR_CONSOLE_OUT:
		return "CONSOLE_OUT"
	case SPR_CONSOLE_IN:
		return "CONSOLE_IN"
	default:
		if spr >= SPR_USERGEN_BASE && spr < SPR_USERGEN_BASE+8 {
			return fmt.Sprintf("USERGEN[%d]", spr-SPR_USERGEN_BASE)
		} else if spr >= SPR_USER_CODE_MMU_BASE && spr < SPR_USER_DATA_MMU_BASE {
			return fmt.Sprintf("USER_CODE_MMU[%d]", spr-SPR_USER_CODE_MMU_BASE)
		} else if spr >= SPR_USER_DATA_MMU_BASE && spr < SPR_KERN_CODE_MMU_BASE {
			return fmt.Sprintf("USER_DATA_MMU[%d]", spr-SPR_USER_DATA_MMU_BASE)
		} else if spr >= SPR_KERN_CODE_MMU_BASE && spr < SPR_KERN_DATA_MMU_BASE {
			return fmt.Sprintf("KERN_CODE_MMU[%d]", spr-SPR_KERN_CODE_MMU_BASE)
		} else if spr >= SPR_KERN_DATA_MMU_BASE && spr < SPR_IO_BASE {
			return fmt.Sprintf("KERN_DATA_MMU[%d]", spr-SPR_KERN_DATA_MMU_BASE)
		}
		return fmt.Sprintf("SPR%d", spr)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
