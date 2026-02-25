# YAPL Compiler for WUT-4 - Context Reference

This document is designed to efficiently rebuild Claude's context for working on the YAPL compiler. Point Claude at this file at the start of a session.

## Quick Orientation

**YAPL** (Yet Another Programming Language) is a C-like systems language targeting the **WUT-4**, a 16-bit RISC processor that exists only in emulation. The compiler is written in Go, organized as 4 separate pass programs driven by the `ya` driver. The long-term goal is self-hosting: YAPL compiling itself on WUT-4's 64KB address space.

**Repository root:** `github.com/gmofishsauce/wut4`

**Key locations:**

| Path | Contents |
|------|----------|
| `lang/` | Compiler source (this directory) |
| `lang/ya/` | Driver program |
| `lang/ylex/` | Pass 1: Lexer + constant evaluator |
| `lang/yparse/` | Pass 2: Parser + AST builder |
| `lang/ysem/` | Pass 3: Semantic analyzer + IR generator |
| `lang/ygen/` | Pass 4: Code generator (IR -> WUT-4 asm) |
| `lang/test/` | Test programs (.yapl files) |
| `lang/yapl_grammar.ebnf` | Authoritative YAPL grammar |
| `lang/IR_FORMAT.md` | IR specification (Pass 3 output format) |
| `lang/yasm/` | WUT-4 assembler |
| `emul/` | WUT-4 emulator |
| `specs/wut4arch.pdf` | WUT-4 architecture specification |
| `specs/wut4asm.pdf` | WUT-4 assembly language specification |
| `specs/YAPL.pdf` | Informal YAPL language description |

## Important Rules for Claude

1. **Do NOT rename binaries** without asking the user first, even if names seem generic or conflicting.
2. **Use the `ya` driver** to compile, not ad-hoc pipelines. Use `ya -k` to preserve intermediate files.
3. **`YAPL` environment variable** can point at the compiler tree root to use local builds instead of installed ones.
4. **Binaries are installed to `~/go/bin/`**: `ya`, `ylex`, `yparse`, `ysem`, `ygen`, `yasm`.
5. **Build with:** `cd lang && ./build` (runs `go install` for each pass).

## Compiler Pipeline

```
source.yapl --> ylex --> yparse --> ysem --> ygen --> ypeep [optional] --> yasm --> wut4.out
              tokens   AST+syms   IR(3addr)  .asm     asm to asm           .asm --> binary 
```

All inter-pass formats are ASCII text. Use `ya -k -v source.yapl` to see all stages and keep intermediate files:
- `<name>.lexout` - Token stream
- `<name>.parseout` - AST + symbol table
- `<name>.ir` - Three-address IR with virtual registers
- `<name>.asm` - WUT-4 assembly

Other useful flags: `-S` or `-c` stop after assembly generation, `-o file` sets output name.

## YAPL Language Summary

YAPL resembles C with deliberate simplifications for a small compiler.

### Types
- **Integer:** `byte` (alias `uint8`), `int16`, `uint16`
- **Pointer:** `@byte`, `@int16`, `@uint16`, `@void` (note: `@` not `*`)
- **Block:** `block32`, `block64`, `block128` (opaque; only `&` allowed)
- **Struct:** `struct name { fields };`
- **Arrays:** 1D only: `var byte buf[256];`
- **void:** function return type only (but `@void` is a valid pointer type)

### Key Syntax Differences from C
- **`@` is dereference** (not `*`): `@ptr` dereferences, `&x` takes address
- **`var` and `const` keywords** for declarations (not bare types)
- **`func` keyword** for functions: `func int16 add(int16 a, int16 b) { ... }`
- **No type promotion** - must use explicit casts: `int16(x)`
- **Visibility by case** - uppercase initial = public/global, lowercase = static/private
- **Single global namespace** - all top-level symbols (functions, variables, constants, struct tags) share one namespace, so struct tags can be used directly as type names
- **No preprocessor** - `#if`/`#else`/`#endif`, `#include` handled by lexer; constant expressions folded at lex time
- **`#asm("...")`** for inline assembly (raw string, no escapes)
- **`#pragma`** directives handled by lexer: `#pragma bootstrap` for standalone programs (replaces `#asm(".bootstrap")` which no longer triggers bootstrap mode), `#pragma message <text>` prints to stderr at compile time
- **Declarations before statements** in function bodies
- **Labels only at function level**, not inside blocks
- **No `for`-init declarations** - declare loop vars before the `for`
- **String literals** initialize byte arrays with auto null terminator
- **Max identifier length:** 15 chars

### Operator Precedence (low to high)
1. Assignment `=` (right-assoc)
2. `||`
3. `&&`
4. Comparison `== != < > <= >=` (non-associative)
5. Additive `+ - | ^`
6. Multiplicative `* / % & << >>`
7. Unary `@ & - ~ ! sizeof` and type casts
8. Postfix `() [] . ->`

### Example Program
```
const byte hello[] = "hello, world.\n";

func void main() {
    Putstr(&hello);
}

func void Putc(byte b) {
    #asm("ldi r2 96");
    #asm("ssp r1 r2");
}

func void Putstr(@byte bp) {
    while (@bp != 0) {
        Putc(@bp);
        bp = bp + 1;
    }
}
```

## WUT-4 Architecture Essentials

16-bit RISC, Harvard architecture (separate 64KB I-space and D-space).

### Registers
| Reg | Purpose |
|-----|---------|
| R0 | Hardwired zero for most instructions; LINK for a few |
| R1 | Arg 1 / return value (caller-saved) |
| R2 | Arg 2 (caller-saved) |
| R3 | Arg 3 (caller-saved) |
| R4 | Callee-saved |
| R5 | Callee-saved |
| R6 | Callee-saved |
| R7 | Stack pointer (grows downward) |

**LINK register** (SPR 0) holds return address from `jal`. LINK is callee-saved:
every function unconditionally saves and restores it in its prologue/epilogue.
This eliminates a class of bugs where an unguarded LINK is clobbered by a nested
call. A future optimization could skip the save/restore for proven leaf functions.

### Calling Convention
- Args 0-2 in R1-R3; args 3+ pushed on stack right-to-left
- Return value in R1
- SP is fixed during function execution (no dynamic alloc)
- Callee saves/restores R4-R6 and LINK

### Stack Frame Layout
```
[higher addresses]
  arg N (if N > 3)     [SP + FRAMESIZE + ...]
  saved LINK           [SP + FRAMESIZE - 2]  (always present)
  saved R6
  saved R5
  saved R4
  saved register params (R1-R3, as needed)
  virtual register spill slots
  local variables
[SP points here - lower addresses]
```

### Key Instructions
- `ldi Rd, imm` - load immediate (lots of details in ISA - can use labels)
- `ldw Rd, Rs, imm` - load word: Rd = mem[Rs + imm]
- `ldb Rd, Rs, imm` - load byte
- `stw Rs, Rd, imm` - store word: mem[Rd + imm] = Rs
- `stb Rs, Rd, imm` - store byte
- `adi Rd, Rs, imm` - add immediate
- `lui Rd, imm` - load upper immediate
- `jal target` - jump and link (return addr in LINK SPR)
- `brz/brnz/brn/brnn/brc/brnc Rs, offset` - conditional branches
- XOP (3-reg ALU): `add Rd, Rs1, Rs2`, `sub`, `and`, `or`, `xor`, etc.
- YOP: `lsp` / `ssp` (load/store special reg), `tst` (set flags)
- ZOP: `not`, `neg`, `sxt`, `sra`, `srl`, `ji` (jump indirect via LINK)
- VOP: `hlt`, `ccf`, `scf`, `di`, `ei`, `rti`, `brk`, `die`

### I/O (for test programs)
- SPR 96: Console output (write byte via `ssp`)
- SPR 97: Console input (read byte via `lsp`)
- SPR 98: Console status

## IR Format Quick Reference

Three-address code with virtual registers `t0, t1, ...`. Key instructions:

```
t = CONST.W 0x000A          # load 16-bit constant
t = CONST.B 0xFF            # load 8-bit constant
t = LOAD.W [SP+n]           # load word from stack
t = LOAD.B [addr]           # load byte (sign-extend)
t = LOAD.BU [addr]          # load byte (zero-extend)
STORE.W [addr], t            # store word
STORE.B [addr], t            # store byte
t = ADDR label               # load address of global/static
t = PARAM n                  # load parameter n
t = ADD.W a, b               # arithmetic (also SUB, MUL, DIV.S, DIV.U, MOD.S, MOD.U, NEG)
t = AND.W a, b               # bitwise (also OR, XOR, NOT, SHL, SHR, SAR)
t = EQ.W a, b                # comparison -> 0/1 (also NE, LT.S, LE.S, GT.S, GE.S, LT.U, LE.U, GT.U, GE.U)
LABEL name                   # define label
JUMP label                   # unconditional jump
JUMPZ t, label               # jump if zero
JUMPNZ t, label              # jump if nonzero
ARG n, t                     # set call argument
t = CALL func, n             # call with n args, result in t
CALL func, n                 # void call
RETURN t                     # return value
RETURN                       # void return
```

Address forms: `[SP+n]`, `[label]`, `[t]`, `[t+n]`

Implementation limits: 16 params, 32 locals, 256-byte frame, 32 struct fields, 16 nesting depth.

## Pass Details

### Pass 1 (ylex) - Lexer + Constant Evaluator
- Tokenizes source into `token#, CATEGORY, value` lines (CATEGORY: KEY, ID, PUNCT, LIT)
- Evaluates constant expressions in `const` initializers, array dimensions, and `#if` conditions
- Handles `#if`/`#else`/`#endif`, `#include`, `#file`, `#line`, `#pragma` directives
- `#include "path"` or `#include <path>` splices the named file's tokens in place; path is relative to the including file's directory, or absolute
- Numeric literals output as hex; strings as quoted
- Example: `const uint16 SIZE = 64;` -> token stream with `LIT, 0x0040`

### Pass 2 (yparse) - Parser
- Parses token stream into text-based AST with STRUCT, CONST, VAR, FUNC sections
- Builds symbol table; validates visibility rules
- Outputs hierarchical structure: FUNC contains PARAM, LOCAL, FRAMESIZE, BODY with statements
- Expressions in prefix notation: `BINARY +`, `UNARY @`, `CALL`, `INDEX`, `FIELD`, etc.

### Pass 3 (ysem) - Semantic Analyzer + IR Generator
- Reads AST from Pass 2
- Performs type checking on all expressions
- Resolves symbols, validates types in assignments/operations/calls
- Generates three-address IR with virtual registers (see IR format above)
- Output format specified in `IR_FORMAT.md`

### Pass 4 (ygen) - Code Generator
- Reads IR, parses instructions
- Maps virtual registers (t0-tN) to physical registers (R1-R6)
- Generates WUT-4 assembly with prologue/epilogue
- Emits bootstrap code (_start: sets SP, calls main, halts)
- Handles LINK register save/restore unconditionally (callee-save)

## Test Programs

| File | Tests |
|------|-------|
| `test/hello.yapl` | Basic I/O, string output via SPR, inline asm |
| `test/fib.yapl` | Loops, arithmetic, function calls, uint16-to-string |
| `test/bootstrap.yapl` | Stack init stub for multi-file compilation |
| `test/utoa.yapl` | Integer-to-ASCII conversion helper |
| `test/fib-combined.yapl` | Combined fib + helpers in single file |
| `test/test-simple.yapl` | Parameter passing, pointers, function returns |

Plus unit tests for ylex and yparse.

## Running Programs

```bash
# Compile and run on emulator
ya test/hello.yapl && emul wut4.out

# Compile with intermediate files preserved
ya -k -v test/hello.yapl

# Compile to assembly only
ya -S test/hello.yapl

# Run emulator with tracing
emul -trace trace.log wut4.out

# Run emulator with cycle limit
emul -max-cycles 10000 wut4.out

# Disassemble a binary
yasm -d wut4.out
```

## Known Architectural Decisions

- Multi-pass with ASCII inter-pass files: debuggability over speed; each pass must fit in 64KB for self-hosting
- No frame pointer: SP fixed during function execution, locals at fixed offsets
- `@` for dereference instead of `*`: avoids ambiguity with multiplication
- Visibility by identifier case: eliminates need for `static`/`extern` keywords
- Constant folding in lexer: simplifies parser and later passes
- `#asm()` passes through all passes unchanged to final assembly output

## Authoritative References

- **Grammar:** `yapl_grammar.ebnf` (this is the canonical grammar)
- **IR format:** `IR_FORMAT.md`
- **Architecture:** `../specs/wut4arch.pdf`
- **Assembly:** `../specs/wut4asm.pdf`
- **Language (informal):** `../specs/YAPL.pdf`
