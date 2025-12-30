# WUT-4 Assembler and Disassembler

A complete assembler and disassembler for the WUT-4 16-bit RISC architecture.

## Building

```bash
go build
```

This creates the `asm` binary in the current directory.

## Usage

### Assembler

```bash
./asm <input.asm> [output.bin]
```

Assembles a WUT-4 assembly source file into a binary executable. If no output file is specified, the default is `out.bin`.

Example:
```bash
./asm test.asm test.bin
```

### Disassembler

```bash
./asm -d <binary-file>
```

Disassembles a WUT-4 binary file and prints the assembly code to stdout.

Example:
```bash
./asm -d test.bin
```

## Assembly Language Syntax

### Basic Structure

- Lines can contain an optional label, an opcode/directive, arguments, and comments
- Labels must be followed by a colon (`:`)
- Comments start with semicolon (`;`)
- Arguments can be separated by spaces or commas (or both)

Example:
```assembly
label:  opcode arg1, arg2, arg3  ; comment
```

### Registers

- 8 general purpose registers: `r0` through `r7`
- `r0` typically reads as 0 and writes to a "black hole"
- Special register `link` maps to `r0` in certain contexts

### Instructions

#### Base Instructions
- `ldw rA, rB, imm7` - Load word: rA = mem[rB + imm7]
- `ldb rA, rB, imm7` - Load byte: rA = mem[rB + imm7] (sign extended)
- `stw rA, rB, imm7` - Store word: mem[rB + imm7] = rA
- `stb rA, rB, imm7` - Store byte: mem[rB + imm7] = rA (low byte)
- `adi rA, rB, imm7` - Add immediate: rA = rB + imm7
- `lui rA, imm10` - Load upper immediate: rA = imm10 << 6

#### Branch Instructions
- `br offset` - Unconditional branch
- `brl offset` - Branch and link (subroutine call)
- `brz offset` / `breq offset` - Branch if zero/equal
- `brnz offset` / `brneq offset` - Branch if not zero/not equal
- `brc offset` / `bruge offset` - Branch if carry/unsigned >=
- `brnc offset` / `brult offset` - Branch if no carry/unsigned <
- `brsge offset` - Branch if signed >=
- `brslt offset` - Branch if signed <

#### Jump Instructions
- `jal rA, rB, imm6` - Jump and link: PC = rB[15:6] + imm6, rA = old PC + 2

#### Extended Instructions (XOPs - 3 operands)
- `add rA, rB, rC` - Add: rA = rB + rC
- `adc rA, rB, rC` - Add with carry
- `sub rA, rB, rC` - Subtract: rA = rB - rC
- `sbb rA, rB, rC` - Subtract with borrow
- `and rA, rB, rC` - Bitwise AND
- `or rA, rB, rC` - Bitwise OR
- `xor rA, rB, rC` - Bitwise XOR

#### Extended Instructions (YOPs - 2 operands)
- `lsp rA, rB` - Load special register
- `ssp rA, rB` - Store special register
- `lsi rA, rB` - Load special indirect
- `ssi rA, rB` - Store special indirect
- `lcw rA, rB` - Load code word
- `sys N` - System call (N = 0..7)
- `tst rA, rB` - Test (subtract and set flags without storing result)

#### Extended Instructions (ZOPs - 1 operand)
- `not rA` - Bitwise NOT
- `neg rA` - Negate (two's complement)
- `sxt rA` - Sign extend byte to word
- `sra rA` - Shift right arithmetic
- `srl rA` - Shift right logical
- `ji rA` - Jump indirect: PC = rA

#### Extended Instructions (VOPs - 0 operands)
- `ccf` - Clear carry flag
- `scf` - Set carry flag
- `di` - Disable interrupts
- `ei` - Enable interrupts
- `hlt` - Halt
- `brk` - Breakpoint
- `rti` - Return from interrupt
- `die` - Die (illegal instruction, generates 0xFFFF)

### Instruction Aliases

- `ldi rT, imm16` - Load immediate (expands to LUI/ADI as needed)
- `mv rT, rS` - Move register (expands to `adi rT, rS, 0`)
- `ret [rN]` - Return (expands to `ji rN`, default `ji link`)
- `sla rN` - Shift left arithmetic (expands to `adc rN, rN, rN`)
- `sll rN` - Shift left logical (expands to `add rN, rN, rN`)

### Directives

- `.code` - Switch to code segment
- `.data` - Switch to data segment
- `.align N` - Align to N-byte boundary
- `.bytes val1, val2, ...` or `.bytes "string"` - Emit byte values
- `.words val1, val2, ...` - Emit 16-bit word values
- `.space N` - Reserve N bytes of space
- `.set symbol, value` - Define a symbol with a constant value

### Expressions

Constant expressions are supported in immediate operands and directive arguments. Expressions can contain:
- Decimal numbers: `42`, `-10`
- Hexadecimal numbers: `0x100`, `0xFFFF`
- Operators: `+`, `-`, `*`, `/`
- Parentheses: `(expr)`
- Labels and symbols (defined with `.set`)

## Binary File Format

The output binary file has a 16-byte header followed by code and data segments:

### Header Structure
- Bytes 0-1: Magic number (0xDDD1, little endian)
- Bytes 2-3: Code size in bytes (little endian)
- Bytes 4-5: Data size in bytes (little endian)
- Bytes 6-15: Reserved (10 bytes of zeros)

### Segments
- Code segment starts at byte 16
- Data segment starts at byte 16 + code_size

All multi-byte values are stored in little endian format.

## Implementation Notes

- Written in Go using C-like constructs for eventual self-hosting
- Avoids closures and dynamic memory allocation where possible
- Two-pass assembly for proper forward reference resolution:
  - Pass 1 collects all labels and their addresses into the symbol table
  - Pass 2 generates final code with all forward references resolved
  - Forward references are fully supported for branch and jump instructions
- Special handling for register `r0`/`link` per architecture specification
- All instructions are 16-bit (2 bytes) and must be aligned on even addresses in code space

## Known Limitations

- Error messages could be more detailed
- No listing file generation
- No macro support
