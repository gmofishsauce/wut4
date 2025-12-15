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

// CPU mode constants
const (
	ModeKernel = 0
	ModeUser   = 1
)

// CPU represents the WUT-4 processor state
type CPU struct {
	// Architectural state
	pc      uint16         // Program counter
	gen     [2][8]uint16   // General registers [mode][reg]
	spr     [2][128]uint16 // Special registers [mode][spr]
	mode    uint8          // 0=kernel, 1=user
	context uint8          // Current user context (1-255, 0=kernel)

	// Memory
	physMem []uint16        // 16MB physical memory (8M words)
	mmu     [256][32]uint16 // MMU pages [context][slot]

	// I/O
	consoleIn  io.Reader // stdin
	consoleOut io.Writer // stdout

	// Execution state
	cycles     uint64 // Cycle counter
	running    bool   // Run/halt flag
	intEnabled bool   // Interrupt enable

	// Exception state
	pendingException bool   // True if exception occurred
	exceptionVector  uint16 // Exception vector address
	exceptionData    uint16 // Additional exception data (e.g., fault address)

	// Trace
	tracer *Tracer
}

// NewCPU creates and initializes a new CPU
func NewCPU() *CPU {
	cpu := &CPU{
		physMem: make([]uint16, 8*1024*1024), // 8M words = 16MB
		running: true,
		mode:    ModeKernel,
	}

	// Initialize kernel MMU: slot 0 points to physical page 0 with RWX permissions
	// Permission bits: 00 = RWX (see spec)
	cpu.mmu[0][0] = 0x0000  // Code page 0 → physical page 0
	cpu.mmu[0][16] = 0x0000 // Data page 0 → physical page 0

	return cpu
}

// Reset resets the CPU to initial state
func (cpu *CPU) Reset() {
	cpu.pc = 0
	cpu.mode = ModeKernel
	cpu.context = 0
	cpu.cycles = 0
	cpu.running = true
	cpu.intEnabled = false
	cpu.pendingException = false

	// Clear all registers
	for m := 0; m < 2; m++ {
		for i := 0; i < 8; i++ {
			cpu.gen[m][i] = 0
		}
		for i := 0; i < 128; i++ {
			cpu.spr[m][i] = 0
		}
	}
}

// Run executes the fetch-decode-execute loop
func (cpu *CPU) Run() error {
	for cpu.running {
		// Trace instruction before execution
		if cpu.tracer != nil {
			cpu.tracer.TracePreInstruction(cpu)
		}

		// Fetch
		inst, err := cpu.fetch()
		if err != nil {
			return err
		}

		// Handle pending exception from previous instruction
		if cpu.pendingException {
			cpu.handleException()
			cpu.cycles++
			continue
		}

		// Decode
		decoded := decode(inst)

		// Execute
		err = cpu.execute(decoded)
		if err != nil {
			return err
		}

		// Handle exception if one occurred during execute
		if cpu.pendingException {
			cpu.handleException()
		}

		cpu.cycles++

		// Trace after execution
		if cpu.tracer != nil {
			cpu.tracer.TracePostInstruction(cpu, decoded)
		}
	}

	return nil
}

// fetch fetches the next instruction from memory
func (cpu *CPU) fetch() (uint16, error) {
	// Translate PC through code MMU
	physAddr, err := cpu.translateCode(cpu.pc)
	if err != nil {
		cpu.raiseException(0x0012, cpu.pc) // Page fault on code fetch
		return 0, nil                       // Will be handled in next cycle
	}

	inst := cpu.physMem[physAddr]
	return inst, nil
}

// handleException processes a pending exception
func (cpu *CPU) handleException() {
	// Save state to kernel special registers
	cpu.spr[ModeKernel][SPR_IRR] = cpu.pc
	cpu.spr[ModeKernel][SPR_ICR] = cpu.exceptionVector
	cpu.spr[ModeKernel][SPR_IDR] = cpu.exceptionData
	cpu.spr[ModeKernel][SPR_ISR] = uint16(cpu.mode)

	// Trace exception
	if cpu.tracer != nil {
		cpu.tracer.TraceException(cpu, cpu.exceptionVector, cpu.exceptionData)
	}

	// Switch to kernel mode and disable interrupts
	cpu.mode = ModeKernel
	cpu.intEnabled = false

	// Jump to exception vector
	cpu.pc = cpu.exceptionVector

	// Clear exception state
	cpu.pendingException = false
	cpu.exceptionVector = 0
	cpu.exceptionData = 0
}

// raiseException sets up an exception to be handled
func (cpu *CPU) raiseException(vector uint16, data uint16) {
	// Check for double fault: exception while in kernel mode with interrupts disabled
	if cpu.mode == ModeKernel && !cpu.intEnabled {
		// Log double fault to trace file before halting
		if cpu.tracer != nil {
			cpu.tracer.TraceDoubleFault(cpu, vector, data)
		}
		fmt.Fprintf(cpu.consoleOut, "\n\n*** DOUBLE FAULT ***\n")
		fmt.Fprintf(cpu.consoleOut, "Exception 0x%04X (data=0x%04X) in kernel mode with interrupts disabled\n", vector, data)
		fmt.Fprintf(cpu.consoleOut, "PC=0x%04X, Cycle=%d\n\n", cpu.pc, cpu.cycles)
		cpu.running = false
		return
	}

	cpu.pendingException = true
	cpu.exceptionVector = vector
	cpu.exceptionData = data
}

// getFlags returns the current CPU flags (C, Z, N, V)
func (cpu *CPU) getFlags() uint16 {
	return cpu.spr[cpu.mode][SPR_FLAGS] & 0x000F
}

// setFlags sets the CPU flags
func (cpu *CPU) setFlags(flags uint16) {
	cpu.spr[cpu.mode][SPR_FLAGS] = (cpu.spr[cpu.mode][SPR_FLAGS] & 0xFFF0) | (flags & 0x000F)
}

// updateFlags updates flags based on a 16-bit result
func (cpu *CPU) updateFlags(result uint32, carry bool, overflow bool) {
	var flags uint16

	// Carry flag (bit 0)
	if carry {
		flags |= FLAG_C
	}

	// Zero flag (bit 1)
	if (result & 0xFFFF) == 0 {
		flags |= FLAG_Z
	}

	// Negative flag (bit 2) - MSB of result
	if (result & 0x8000) != 0 {
		flags |= FLAG_N
	}

	// Overflow flag (bit 3)
	if overflow {
		flags |= FLAG_V
	}

	cpu.setFlags(flags)
}
