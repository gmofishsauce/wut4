# WUT-4 Emulator

A functional emulator for the WUT-4 16-bit RISC architecture.

## Overview

The WUT-4 emulator executes binaries produced by the WUT-4 assembler located in `../asm/`. It supports:

- Full WUT-4 instruction set (base, XOP, YOP, ZOP, VOP)
- User and kernel execution modes
- Memory Management Unit (MMU) with paging
- 256 user contexts
- Console I/O via stdin/stdout
- Detailed execution tracing
- Precise exception handling

## Building

```bash
go build
```

This produces an `emul` executable.

## Usage

```bash
./emul [options] <binary-file>
```

### Options

- `-trace <file>` - Write detailed execution trace to file
- `-max-cycles N` - Stop execution after N cycles (useful for debugging infinite loops)
- `-version` - Show version and exit

### Examples

Run a binary:
```bash
./emul program.bin
```

Run with trace output:
```bash
./emul -trace trace.txt program.bin
```

Run with cycle limit:
```bash
./emul -max-cycles 10000 program.bin
```

## Architecture

### WUT-4 Specification (../specs/wut4arch.pdf)

- 16-bit little-endian architecture
- 8 general-purpose registers (r0-r7), r0 hardwired to zero
- Split code and data address spaces (64KB each)
- Privileged (kernel) and user modes
- 128 special registers (SPRs)
- MMU with 16 code + 16 data pages per context (4KB pages)
- 16MB physical memory
- Sequentially consistent memory model

### Memory Map

- **Physical Memory**: 16MB (8M words)
- **Virtual Address Space**: 64KB code + 64KB data per context
- **Page Size**: 4096 bytes (2048 words)
- **MMU**: Direct-mapped, 32 slots per context (16 code, 16 data)

### Special Registers

| SPR | Name | Description | Access |
|-----|------|-------------|--------|
| 0 | LINK | Link register | User/Kernel |
| 1 | FLAGS | CPU flags (C, Z, N, V) | User/Kernel |
| 6 | CYCLO | Cycle counter low | User/Kernel (RO) |
| 7 | CYCHI | Cycle counter high | User/Kernel (RO) |
| 8 | IRR | Interrupt return register | Kernel |
| 9 | ICR | Interrupt cause register | Kernel |
| 10 | IDR | Interrupt data register | Kernel |
| 11 | ISR | Interrupt state register | Kernel |
| 15 | CONTEXT | User context register | Kernel |
| 16-23 | USERGEN | User general registers | Kernel |
| 32-47 | USER_CODE_MMU | User code MMU entries | Kernel |
| 48-63 | USER_DATA_MMU | User data MMU entries | Kernel |
| 64-79 | KERN_CODE_MMU | Kernel code MMU entries | Kernel |
| 80-95 | KERN_DATA_MMU | Kernel data MMU entries | Kernel |
| 96 | CONSOLE_OUT | Console output (write byte) | Kernel |
| 97 | CONSOLE_IN | Console input (read byte) | Kernel |

### Instruction Set

**Base Instructions:**
- `LDW` - Load word
- `LDB` - Load byte (sign extended)
- `STW` - Store word
- `STB` - Store byte
- `ADI` - Add immediate (sets flags)
- `LUI` - Load upper immediate
- `BRx` - Conditional branch (8 conditions)
- `JAL` - Jump and link

**XOP (3-operand ALU):**
- `ADD`, `ADC`, `SUB`, `SBB`, `AND`, `OR`, `XOR`

**YOP (2-operand):**
- `LSP`, `SSP` - Load/store special register
- `LSI`, `SSI` - Load/store special register indirect
- `LCW` - Load code word
- `SYS` - System call (vectors 0-7)

**ZOP (1-operand):**
- `NOT`, `NEG`, `ZXT`, `SXT`, `SRA`, `SRL`, `DUB`

**VOP (0-operand):**
- `CCF`, `SCF` - Clear/set carry flag
- `DI`, `EI` - Disable/enable interrupts
- `HLT` - Halt (kernel only)
- `BRK` - Breakpoint
- `RTI` - Return from interrupt
- `DIE` - Illegal instruction (0xFFFF)

## Console I/O

The emulator connects console I/O to stdin and stdout:
- Programs write bytes to SPR 96 to output to stdout
- Programs read bytes from SPR 97 to input from stdin
- I/O operations are only available in kernel mode

## Trace Format

When using `-trace`, the emulator generates a detailed trace showing:

- Cycle number and PC
- Current mode and context
- Instruction (hex and disassembly)
- Decoded fields
- Register values before and after
- CPU flags before and after
- Memory operations (reads/writes with addresses)
- Special register operations
- Exceptions and mode switches
- Console I/O

### Example Trace

```
========================================
CYCLE: 0000000000000000
PC: 0x0000 [kernel] [context=0]
INST: 0xA000  LUI r0, 0x000
DECODE: op=5 rA=0 rB=0 imm=0
REGS BEFORE: r0=0000 r1=0000 r2=0000 r3=0000 r4=0000 r5=0000 r6=0000 r7=0000
FLAGS BEFORE: C=0 Z=0 N=0 V=0
EXECUTE: r0 ← 0x0000
REGS AFTER: r0=0000 r1=0000 r2=0000 r3=0000 r4=0000 r5=0000 r6=0000 r7=0000
```

## Binary Format

The emulator expects raw 16-bit little-endian words:
- No header or metadata
- Loaded starting at physical address 0
- Binary size must be even (multiple of 2 bytes)

## Bootstrap Process

1. CPU starts in kernel mode, context 0
2. PC = 0, interrupts disabled
3. Kernel MMU slot 0 (code and data) → physical page 0, RWX permissions
4. Binary loaded at physical address 0
5. Execution begins at address 0

## Error Handling

The emulator handles:
- **Illegal instructions** (0x0000, 0xFFFF, invalid opcodes)
- **Page faults** (invalid permissions, unmapped pages)
- **Alignment faults** (word access to odd address)
- **Double faults** (exception in kernel with interrupts disabled) → HALT

## Performance

The emulator is not cycle-accurate and runs as fast as possible. Typical speeds on modern hardware:
- Without trace: 10-50 MHz
- With trace: 1-5 MHz (I/O bound)

## Development

### File Structure

```
emul/
├── main.go       - Entry point, CLI
├── cpu.go        - CPU state, execution loop
├── decode.go     - Instruction decoder
├── execute.go    - Instruction implementations
├── memory.go     - Memory and MMU
├── spr.go        - Special register handling
├── io.go         - Console I/O
├── trace.go      - Trace generation
├── disasm.go     - Disassembler
├── constants.go  - Constants (SPRs, flags, etc.)
└── go.mod        - Go module
```

### Building from Source

Requirements:
- Go 1.21 or later

Build:
```bash
cd emul
go build
```

Test:
```bash
go test ./...
```

## License

Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

## See Also

- `../asm/` - WUT-4 assembler
- `wut4.pdf` - Complete architecture specification
