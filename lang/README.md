# YAPL Compiler for WUT-4

## Project Overview

This directory contains the YAPL (Yet Another Programming Language) compiler, a self-hosted (eventually) compiler designed to run natively on the WUT-4 architecture.

WUT-4: ../specs/wut4arch.pdf

**Primary Goal:** Self-hosting - the compiler must be small enough to compile itself while running on WUT-4's 64KB code space.

**Language Description:** https://docs.google.com/document/d/1hgsayGjZJc6WUVjSEsPRWVxPeXkVFLKpRCx5jc5hrx8/edit?usp=sharing

## Why YAPL?

YAPL is specifically designed to be compiled by a small, simple compiler that fits within WUT-4's memory constraints:

- **Machine types only** (byte, int16, uint16) - no complex type system
- **No preprocessor** - constant expressions evaluated in lexer pass
- **Single-dimensional arrays** - simpler than multidimensional
- **Minimal operators** - compact parser and code generator
- **Simple visibility rules** - uppercase = public, lowercase = private
- **No string type** - byte arrays only (BCPL-style)

## Compiler Usage

### Building

```bash
cd lang
./build
```

This installs the compiler components (`ylex`, `yparse`, `ysem`, `ygen`, `ya`) to your Go bin directory (typically `~/go/bin`). Ensure this directory is in your PATH.

### Compiling Programs

```bash
# Compile to binary
ya hello.yapl

# Compile with custom output name
ya -o myprogram hello.yapl

# Stop after generating assembly (don't run assembler)
ya -S hello.yapl

# Keep all intermediate files for debugging
ya -k hello.yapl

# Verbose output showing each compilation stage
ya -v hello.yapl

# Combine flags
ya -k -v -S hello.yapl
```

### Intermediate Files

With the `-k` flag, intermediate files are written to the source directory:

| File | Contents |
|------|----------|
| `<name>.lexout` | Pass 1 token stream |
| `<name>.parseout` | Pass 2 AST output |
| `<name>.ir` | Pass 3 intermediate representation |
| `<name>.asm` | Pass 4 assembly output |

### Development Mode

For development, you can build binaries locally and use the `YAPL` environment variable:

```bash
# Build locally instead of installing
cd ylex && go build -o ylex . && cd ..
cd yparse && go build -o yparse . && cd ..
cd ysem && go build -o ysem . && cd ..
cd ygen && go build -o ygen . && cd ..
cd ya && go build -o ya . && cd ..

# Point to local builds
export YAPL=/path/to/lang
ya -v hello.yapl
```

When `YAPL` is set, the driver looks for binaries at `$YAPL/ylex/ylex`, `$YAPL/yparse/yparse`, etc. Otherwise, it searches PATH.

## Compiler Architecture

The compiler uses a **multi-pass pipeline** with externalized state between passes. Each pass is a separate program that reads input files and writes output files, keeping each pass small enough to fit in 64KB. All the intermediate files are ASCII, making debugging  and testing simpler.

### Four-Pass Design

```
Source Code (.yapl)
    ↓
┌─────────────────────────────────────────┐
│ Pass 1: Lexer + Constant Evaluator      │
│ - Tokenize source                        │
│ - Handle #if/#line/#file directives      │
│ - Evaluate constant expressions          │
│ - Output: Token stream                   │
└─────────────────────────────────────────┘
    ↓ tokens file
┌─────────────────────────────────────────┐
│ Pass 2: Parser                           │
│ - Parse tokens into AST                  │
│ - Build symbol table                     │
│ - Check visibility rules                 │
│ - Output: AST + Symbol Table             │
└─────────────────────────────────────────┘
    ↓ AST file + symbol table
┌─────────────────────────────────────────┐
│ Pass 3: IR Generator                     │
│ - Convert AST to intermediate repr       │
│ - Type checking                          │
│ - Use symbol table from Pass 2           │
│ - Output: IR (DAGs, etc.)                │
└─────────────────────────────────────────┘
    ↓ IR file
┌─────────────────────────────────────────┐
│ Pass 4: Code Generator                   │
│ - Register allocation                    │
│ - Instruction selection                  │
│ - Generate WUT-4 assembly                │
│ - Output: .w4a assembly file             │
└─────────────────────────────────────────┘
    ↓ assembly file
[WUT-4 Assembler] → Object/Executable
```

## Design Benefits

1. **Each pass stays small** - focused single responsibility fits in 64KB
2. **Independently testable** - feed known input files, verify output files
3. **Debuggable** - inspect intermediate files between passes
4. **Parallelizable development** - define formats, implement passes independently
5. **Classic approach** - proven strategy from memory-constrained systems

## WUT-4 Architecture Constraints

- **Harvard architecture**: 64KB code space + 64KB data space (separate)
- **16-bit word size**
- **8 general-purpose registers** (r0-r7, r0 hardwired to zero)
- **Memory-mapped I/O**
- See `../emul/README.md` for full architecture details

## Target Language Features

### Current Specification (v0.1)

- **Types**: byte (uint8), int16, uint16, @byte, @int16, @uint16, block32, block64, block128, void, structs, arrays
- **Control flow**: if/else, while, for, goto, labels
- **Operators**: Arithmetic, bitwise, logical, comparison
- **Constants**: Compile-time expression evaluation
- **Visibility**: Uppercase = public, lowercase = private
- **Directives**: #if, #else, #endif, #line, #file, #asm

### Inline Assembly (#asm)

The `#asm` directive allows embedding raw assembly code directly in YAPL source. The assembly text passes through the entire compiler unchanged and appears in the generated assembly output exactly where it is positioned in the source.

**Syntax:**
```
#asm("assembly text");
```

**Usage contexts:**

1. **File level** - Outside any function, for assembler directives:
   ```
   #asm(".section .data");
   #asm(".align 2");
   ```

2. **Within functions** - For instructions the compiler cannot generate:
   ```
   func void EnableInterrupts() {
       #asm("sei");
   }
   ```

**Constraints:**
- The string cannot contain escape sequences (no `\n`, `\t`, etc.)
- Each `#asm` statement produces exactly one line of assembly
- For multiple lines of assembly, use multiple `#asm` statements
- Must be terminated with a semicolon

**Example:**
```
// File-level: assembler directive
#asm(".global _start");

func void Halt() {
    // Function-level: emit a halt instruction
    #asm("hlt");
}
```

## Runtime Model and Calling Convention

The runtime model follows early UNIX conventions, adapted for WUT-4's Harvard architecture.

### Memory Layout

**Code Space (I-space):**
- Code begins at address 0 and fills increasing addresses as it is generated

**Data Space (D-space):**
- Static data begins at address 0 and is allocated at increasing addresses
- The end of allocated static data is called "the break"
- Stack starts at 0xFFFE and grows downward toward the break

### Registers

| Register | Purpose |
|----------|---------|
| R0 | Hardwired to zero |
| R1 | Argument 1 / Return value |
| R2 | Argument 2 |
| R3 | Argument 3 |
| R4 | Callee-saved |
| R5 | Callee-saved |
| R6 | Callee-saved |
| R7 | Stack Pointer (SP) |

### Calling Convention

**Argument Passing:**
- First three arguments passed in R1, R2, R3
- Additional arguments (if any) passed on the stack
- Only 16-bit arguments are allowed

**Return Values:**
- Return value in R1
- Only 16-bit return values are allowed

**Register Preservation:**
- **Caller-saved:** R1, R2, R3 (caller must save if values needed after call)
- **Callee-saved:** R4, R5, R6 (callee must save before use and restore before return)

**Stack Discipline:**
- R7 is the stack pointer
- Stack grows downward (toward lower addresses)
- No frame pointer
- SP does not change during function execution (no dynamic allocation like `salloc()`)
- Local variables and function arguments accessed at fixed offsets from SP
- This fixed-SP model simplifies code generation and debugging (??? But not stack backtracing)

## Inter-Pass File Formats

### Pass 1 Output: Token Stream (.tok)

The token stream is a human-readable text file with one token per line.

**Format:** `token#, CATEGORY, value`

Fields are comma-separated (commas do not appear in the YAPL language). A space after each comma is recommended for readability.

The token number is a sequential identifier (1, 2, 3, ...) for debugging purposes, allowing specific tokens to be referenced by number. It is *not* the source line number—source lines are tracked separately via `#line` directives.

**Special Directives:**

Two directives appear in the token stream as metadata (not as regular tokens):

- `#file <filename>` - appears before the first token; identifies source file for error messages
- `#line <number>` - appears when source line number changes; applies to subsequent tokens

These do not follow the `token#, CATEGORY, value` format.

**Token Fields:**

| Field | Description |
|-------|-------------|
| `token#` | Sequential token identifier (1, 2, 3, ...) for debugging |
| `CATEGORY` | One of: `KEY`, `ID`, `PUNCT`, `LIT` |
| `value` | Token value (see below) |

**Categories:**

| Category | Description | Value |
|----------|-------------|-------|
| `KEY` | Keywords (including type names) | The keyword itself |
| `ID` | Identifiers | The identifier string |
| `PUNCT` | Punctuation and operators | The operator or punctuation mark |
| `LIT` | Literals | Numeric (hex) or string (in double quotes) |

**Keywords:**

Control flow: `if`, `else`, `while`, `for`, `return`, `break`, `continue`, `goto`

Declarations: `var`, `const`, `func`, `struct`

Other: `sizeof`

Reserved (not yet implemented): `case`, `default`, `select`, `switch`

Types: `byte`, `uint8`, `int16`, `uint16`, `void`, `block32`, `block64`, `block128`

**Example:**

Source (file `example.yapl`):
```
const uint16 SIZE = 64;
var int16 buf[SIZE * 2];
```

Token stream:
```
#file example.yapl
#line 1
1, KEY, const
2, KEY, uint16
3, ID, SIZE
4, PUNCT, =
5, LIT, 0x0040
6, PUNCT, ;
#line 2
7, KEY, var
8, KEY, int16
9, ID, buf
10, PUNCT, [
11, LIT, 0x0080
12, PUNCT, ]
13, PUNCT, ;
```

Note: `SIZE * 2` is folded to `0x0080` (128) by the lexer.

**Literal Representation:**
- Numeric literals: hexadecimal (e.g., `0x0040`, `0xFFFB` for -5)
- String literals: double-quoted, as in source (e.g., `"hello\n"`)

### Lexer Responsibilities

1. **Tokenization** - convert source text to token stream
2. **Constant symbol table** - track `const` declarations for folding
3. **Constant expression folding** - evaluate and emit single `LIT` token
4. **Conditional compilation** - process `#if`/`#else`/`#endif` (exclude/include code)
5. **File tracking** - emit `#file <filename>` before first token
6. **Line tracking** - emit `#line <number>` when source line changes
7. **File as r-value** - resolve `#file` in expressions to string literal of filename

**Constant Expression Contexts:**
- After `const <identifier> =`
- Inside array dimension brackets `[...]`
- After `#if`

**Constant Expression Operators:**
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Bitwise: `&`, `|`, `^`, `~`, `<<`, `>>`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=` (for `#if`)
- Unary: `-`, `~`, `!`

(No ternary operator and no comma operator, even in declarations)

### Pass 2 Output: AST + Symbol Table

The parser outputs a text-based AST representation. Structure:

```
#file <filename>

STRUCT <name>
  FIELD <type> <name> <offset>
  ...
  SIZE <n> ALIGN <m>

CONST <name> <value>

VAR <visibility> <type> <name> OFFSET <n>

FUNC <returnType> <name>
  PARAM <type> <name> <regHint>
  ...
  LOCAL <type> <name> OFFSET <n>
  ...
  FRAMESIZE <n>
  BODY
    <statements>
  END
```

Statements include EXPR, RETURN, IF/ELSE/ENDIF, WHILE/ENDWHILE, FOR/ENDFOR, GOTO, LABEL, BREAK, CONTINUE.

Expressions are represented as prefix notation: BINARY <op>, UNARY <op>, CALL, INDEX, FIELD, ASSIGN, LIT <value>, ID <name>, etc.

### Pass 3 Output: IR

The semantic analyzer outputs three-address code IR. See `IR_FORMAT.md` for complete specification.

**Quick Reference:**

```
#ir 1
#source filename.yapl

STRUCT <name> <size> <align>
  FIELD <name> <offset> <type>
ENDSTRUCT

CONST <name> <visibility> <type> <value>
DATA <name> <visibility> <type> <size>

FUNC <name>
  VISIBILITY PUBLIC|STATIC
  RETURN <type>
  PARAMS <n>
    PARAM <name> <type> <index>
  LOCALS <nbytes>
    LOCAL <name> <type> <offset>
  FRAMESIZE <n>

  ; Three-address code with virtual registers (t0, t1, ...)
  t0 = CONST.W 0x000A
  t1 = LOAD.W [SP+0]
  t2 = ADD.W t0, t1
  STORE.W [SP+2], t2
  JUMPZ t2, .label
  RETURN t2

ENDFUNC
```

**Implementation Limits:** Max 16 parameters, 32 locals, 256-byte frame size.

## Bootstrap Strategy

1. **Phase 1**: Write compiler in Go (leverage existing Go toolchain)
2. **Phase 2**: Compiler generates WUT-4 assembly, runs via cross-compilation
3. **Phase 3**: Compiler compiles itself (written in YAPL), runs on WUT-4 emulator
4. **Phase 4**: Self-hosted compiler runs natively on WUT-4 hardware

## Testing Strategy

Each pass will have comprehensive tests:

- **Pass 1 tests**: Known source → expected tokens
- **Pass 2 tests**: Known tokens → expected AST/symbols
- **Pass 3 tests**: Known AST → expected IR
- **Pass 4 tests**: Known IR → expected assembly

Since passes externalize state to files, testing is straightforward.

## Current Status

**Status**: Go implementation complete

**Completed:**
- Language specification (v0.1)
- Compiler architecture design
- Project structure
- Runtime model and calling convention
- Pass 1 lexical analyzer (`ylex`) with tests
- Pass 2 parser (`yparse`) with tests
- Pass 3 IR format specification (`IR_FORMAT.md`)
- Pass 3 semantic analyzer (`ysem`) with type checking and IR generation
- Pass 4 code generator (`ygen`)
- Compiler driver (`ya`)

**Next Steps:**
1. Expand test coverage
2. Write YAPL version of the compiler for self-hosting

## Related Documentation

- **WUT-4 Architecture**: `../specs/wut4arch.pdf``
- **WUT-4 Assembler**: `../asm/README.md`
- **YAPL Language Spec**: https://docs.google.com/document/d/1hgsayGjZJc6WUVjSEsPRWVxPeXkVFLKpRCx5jc5hrx8/edit?usp=sharing
- **Pass 3 IR Format**: `IR_FORMAT.md`
- **Semantic Analyzer**: `ysem/README.md`

## Implementation Notes

This is a long-term project requiring patience and methodical development. The multi-pass architecture with externalized state is specifically chosen to make the project tractable despite the ambitious self-hosting goal.

Each pass should be developed, tested, and validated independently before moving to the next pass. The intermediate file formats are as important as the code itself.
