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

// Special Register addresses
const (
	SPR_LINK    = 0  // Link register (per mode)
	SPR_FLAGS   = 1  // CPU flags (C, Z, N, V)
	SPR_CYCLO   = 6  // Cycle counter low
	SPR_CYCHI   = 7  // Cycle counter high
	SPR_IRR     = 8  // Interrupt return register (kernel only)
	SPR_ICR     = 9  // Interrupt cause register (kernel only)
	SPR_IDR     = 10 // Interrupt data register (kernel only)
	SPR_ISR     = 11 // Interrupt state register (kernel only)
	SPR_CONTEXT = 15 // User context register (kernel only)

	// SPR 16-23: User general registers (kernel access to user regs)
	SPR_USERGEN_BASE = 16

	// SPR 32-47: User code MMU
	SPR_USER_CODE_MMU_BASE = 32

	// SPR 48-63: User data MMU
	SPR_USER_DATA_MMU_BASE = 48

	// SPR 64-79: Kernel code MMU
	SPR_KERN_CODE_MMU_BASE = 64

	// SPR 80-95: Kernel data MMU
	SPR_KERN_DATA_MMU_BASE = 80

	// SPR 96-127: I/O registers
	SPR_IO_BASE = 96

	// UART registers (emulated console UART with 64-byte FIFOs)
	// u0 = 96: Write transmit data (low byte)
	// u1 = 97: Read receive data (low byte)
	// u2 = 98: Transmit status (bit 0=overflow, bit 15=empty)
	// u3 = 99: Receive status (bit 0=underflow, bit 15=data available)

	// SPI registers for SD card
	// s0 = 100: SPI data register (read/write)
	// s1 = 101: SPI select register (active low: bit 0 = SD card)
	SPR_SPI_DATA   = 100
	SPR_SPI_SELECT = 101
)

// CPU Flags
const (
	FLAG_C = 0x0001 // Carry
	FLAG_Z = 0x0002 // Zero
	FLAG_N = 0x0004 // Negative
	FLAG_V = 0x0008 // Overflow
	FLAG_T = 0x0100 // Trap (kernel only, bit 8)
	FLAG_IE = 0x0200 // Interrupt enable (kernel only, bit 9, read-only)
)

// Exception vectors
const (
	EX_RESET           = 0x0000 // Reset vector
	EX_ILLEGAL_INST    = 0x0010 // Illegal instruction
	EX_PAGE_FAULT      = 0x0012 // Page fault (memory access violation)
	EX_ALIGNMENT_FAULT = 0x0014 // Alignment fault
	EX_MACHINE_CHECK   = 0x001E // Machine check

	// System call vectors (SYS 0-7 map to vectors 0x0010-0x001E, step 2)
	// Actually per spec, SYS instruction with rB=0 and rA=0-7 use vectors
	// The spec says "SYS 0 through SYS 7" but need to check exact mapping
	EX_SYSCALL_BASE = 0x0010
)

// Branch conditions
const (
	BR_ALWAYS = 0 // Unconditional branch
	BR_LINK   = 1 // Branch and link (unconditional, saves PC)
	BR_EQ     = 2 // Equal / Zero
	BR_NE     = 3 // Not equal / Not zero
	BR_CS     = 4 // Carry set / Unsigned >=
	BR_CC     = 5 // Carry clear / Unsigned <
	BR_GE     = 6 // Signed >= (N == V)
	BR_LT     = 7 // Signed < (N != V)
)
