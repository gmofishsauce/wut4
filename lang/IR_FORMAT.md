# YAPL Intermediate Representation (IR) Format

## Overview

This document specifies the IR format output by Pass 3 (semantic analyzer) of the YAPL compiler. The IR is a linear three-address code with virtual registers, designed to be simple enough for a self-hosting compiler while capturing all information needed for code generation.

## Design Goals

- **Human-readable**: ASCII text format for debugging
- **Simple**: Linear structure, easy to parse and generate
- **Complete**: Contains all information Pass 4 needs for code generation
- **Bounded**: Implementation limits keep compiler complexity manageable

## Implementation Limits

| Limit | Value | Rationale |
|-------|-------|-----------|
| Max parameters per function | 16 | Keeps arg offsets small |
| Max local variables per function | 32 | Keeps local offsets small |
| Max stack frame size | 256 bytes | Single-byte offset addressing |
| Max identifier length | 15 chars | Already in language spec |
| Max struct fields | 32 | Reasonable for small language |
| Max array dimension | 65535 | 16-bit index |
| Max nesting depth (blocks) | 16 | Keeps compiler stack bounded |

## File Structure

```
#ir 1                              ; IR format version
#source filename.yapl              ; original source file

; Global data declarations
DATA ...

; Constant declarations
CONST ...

; Struct definitions
STRUCT ...
ENDSTRUCT

; Function definitions
FUNC ...
ENDFUNC
```

## Sections

### Header

```
#ir 1                    ; IR format version (required, first line)
#source example.yapl     ; source file name
```

### Global Data

Format: `DATA name visibility type size [initializer]`

```
DATA g_count PUBLIC WORD 2 0x0000        ; public uint16, init to 0
DATA s_buffer STATIC BYTES 128 0x00      ; static byte[128], zero-filled
DATA G_Message PUBLIC STRING 7 "Hello\n" ; public string literal
DATA s_point STATIC STRUCT:point_t 4     ; static struct, 4 bytes
```

Visibility:
- `PUBLIC` - uppercase identifier, visible across compilation units
- `STATIC` - lowercase identifier, file-private

Types:
- `WORD` - 16-bit value (int16 or uint16)
- `BYTE` - 8-bit value
- `BYTES n` - byte array of n elements
- `WORDS n` - word array of n elements
- `STRING n` - string literal of n bytes (including escapes, no null terminator)
- `STRUCT:name` - struct type
- `BLOCK32`, `BLOCK64`, `BLOCK128` - block types (4, 8, 16 bytes)

### Constants

Format: `CONST name visibility type value`

```
CONST SIZE PUBLIC UINT16 0x0040
CONST s_mask STATIC UINT16 0x00FF
CONST OFFSET PUBLIC INT16 0xFFFB        ; -5 in two's complement
```

Types: `UINT8`, `INT16`, `UINT16`

### Struct Definitions

```
STRUCT point_t 4 2                ; name, total size, alignment
  FIELD x 0 INT16                 ; name, offset, type
  FIELD y 2 INT16
ENDSTRUCT

STRUCT node_t 8 2
  FIELD value 0 INT16
  FIELD next 2 PTR:node_t         ; pointer to struct
  FIELD data 4 BYTES:4            ; inline array
ENDSTRUCT
```

### Functions

```
FUNC name
  VISIBILITY PUBLIC|STATIC
  RETURN type                     ; VOID, INT16, UINT16, UINT8, PTR:type
  PARAMS n
    PARAM name type index         ; index 0-2 in R1-R3, 3+ on stack
    ...
  LOCALS nbytes
    LOCAL name type offset        ; offset from SP
    ...
  FRAMESIZE n                     ; total stack frame size

  ; IR instructions
  ...

ENDFUNC
```

## IR Instructions

### Virtual Registers

Virtual registers are named `t0`, `t1`, `t2`, ... and represent unlimited temporary values. Pass 4 performs register allocation to map these to physical registers R1-R6 (R0 is zero, R7 is SP).

### Address Forms

| Form | Meaning |
|------|---------|
| `[SP+n]` | Stack-relative (locals, spilled temps) |
| `[label]` | Global/static variable |
| `[t]` | Indirect through virtual register |
| `[t+n]` | Base + displacement (struct field, array element) |

### Constants and Data Movement

| Instruction | Description |
|-------------|-------------|
| `t = CONST.W value` | Load 16-bit constant into t |
| `t = CONST.B value` | Load 8-bit constant (zero-extended to 16) |
| `t = LOAD.W [addr]` | Load 16-bit word from memory |
| `t = LOAD.B [addr]` | Load byte, sign-extend to 16 bits |
| `t = LOAD.BU [addr]` | Load byte, zero-extend to 16 bits |
| `STORE.W [addr], t` | Store 16-bit word to memory |
| `STORE.B [addr], t` | Store low byte to memory |
| `t = ADDR label` | Load address of global/static |
| `t = PARAM n` | Load parameter n (0-indexed) |
| `t = COPY s` | Copy virtual register s to t |

### Stack Operations

| Instruction | Description |
|-------------|-------------|
| `PUSH t` | Push 16-bit value (SP -= 2, then store) |
| `t = POP` | Pop 16-bit value (load, then SP += 2) |

### Arithmetic

| Instruction | Description |
|-------------|-------------|
| `t = ADD.W a, b` | t = a + b |
| `t = SUB.W a, b` | t = a - b |
| `t = MUL.W a, b` | t = a * b (library call) |
| `t = DIV.S a, b` | t = a / b signed (library call) |
| `t = DIV.U a, b` | t = a / b unsigned (library call) |
| `t = MOD.S a, b` | t = a % b signed (library call) |
| `t = MOD.U a, b` | t = a % b unsigned (library call) |
| `t = NEG.W a` | t = -a |

### Bitwise

| Instruction | Description |
|-------------|-------------|
| `t = AND.W a, b` | t = a & b |
| `t = OR.W a, b` | t = a \| b |
| `t = XOR.W a, b` | t = a ^ b |
| `t = NOT.W a` | t = ~a |
| `t = SHL.W a, b` | t = a << b |
| `t = SHR.W a, b` | t = a >> b (logical, zero-fill) |
| `t = SAR.W a, b` | t = a >> b (arithmetic, sign-fill) |

### Comparison

All comparisons produce 1 (true) or 0 (false).

| Instruction | Description |
|-------------|-------------|
| `t = EQ.W a, b` | t = (a == b) |
| `t = NE.W a, b` | t = (a != b) |
| `t = LT.S a, b` | t = (a < b) signed |
| `t = LE.S a, b` | t = (a <= b) signed |
| `t = GT.S a, b` | t = (a > b) signed |
| `t = GE.S a, b` | t = (a >= b) signed |
| `t = LT.U a, b` | t = (a < b) unsigned |
| `t = LE.U a, b` | t = (a <= b) unsigned |
| `t = GT.U a, b` | t = (a > b) unsigned |
| `t = GE.U a, b` | t = (a >= b) unsigned |

### Control Flow

| Instruction | Description |
|-------------|-------------|
| `LABEL name` | Define a label |
| `JUMP label` | Unconditional jump |
| `JUMPZ t, label` | Jump if t == 0 |
| `JUMPNZ t, label` | Jump if t != 0 |

### Function Calls

| Instruction | Description |
|-------------|-------------|
| `ARG n, t` | Set argument n to value t |
| `CALL func, n` | Call function with n arguments (void return) |
| `t = CALL func, n` | Call function, result in t |
| `RETURN` | Return from void function |
| `RETURN t` | Return value t |

## Calling Convention

### Argument Passing

- Arguments 0, 1, 2 → R1, R2, R3
- Arguments 3+ → pushed to stack (right to left, so arg 3 is at lowest address)

### Return Value

- Returned in R1

### Register Preservation

- Caller-saved: R1, R2, R3 (may be destroyed by call)
- Callee-saved: R4, R5, R6 (must be preserved across calls)
- R7 is stack pointer, R0 is hardwired zero

### Stack Frame Layout

```
High addresses
+------------------+
| arg N (if N > 3) |  [SP + FRAMESIZE + 2*(N-3) + 2]
| ...              |
| arg 3            |  [SP + FRAMESIZE + 2]
| return address   |  [SP + FRAMESIZE]
+------------------+
| saved R6         |  (if used)
| saved R5         |  (if used)
| saved R4         |  (if used)
+------------------+
| local M          |
| ...              |
| local 1          |
| local 0          |  [SP + 0]
+------------------+
Low addresses (SP points here)
```

## Example

### Source

```
func int16 add(int16 a, int16 b) {
    var int16 sum;
    sum = a + b;
    return sum;
}

func int16 main() {
    var int16 x;
    var int16 y;
    x = 10;
    y = 20;
    return add(x, y);
}
```

### IR Output

```
#ir 1
#source example.yapl

FUNC add
  VISIBILITY STATIC
  RETURN INT16
  PARAMS 2
    PARAM a INT16 0
    PARAM b INT16 1
  LOCALS 2
    LOCAL sum INT16 0
  FRAMESIZE 2

  t0 = PARAM 0
  t1 = PARAM 1
  t2 = ADD.W t0, t1
  STORE.W [SP+0], t2
  t3 = LOAD.W [SP+0]
  RETURN t3

ENDFUNC

FUNC Main
  VISIBILITY PUBLIC
  RETURN INT16
  PARAMS 0
  LOCALS 4
    LOCAL x INT16 0
    LOCAL y INT16 2
  FRAMESIZE 4

  t0 = CONST.W 0x000A
  STORE.W [SP+0], t0
  t1 = CONST.W 0x0014
  STORE.W [SP+2], t1
  t2 = LOAD.W [SP+0]
  t3 = LOAD.W [SP+2]
  ARG 0, t2
  ARG 1, t3
  t4 = CALL add, 2
  RETURN t4

ENDFUNC
```

## Error Handling

Pass 3 performs semantic analysis and reports errors for:

- Type mismatches in assignments and operations
- Undefined identifiers
- Duplicate declarations
- Invalid operations on types (e.g., arithmetic on pointers)
- Implementation limit violations
- Missing return statements in non-void functions
- Unreachable code (optional warning)

Errors are written to stderr in the format:
```
filename:line: error: message
```

If errors occur, no IR is written to stdout and the exit code is non-zero.
