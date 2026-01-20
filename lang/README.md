# YAPL Compiler for WUT-4

## Project Overview

This directory contains the YAPL (Yet Another Programming Language) compiler, a self-hosting compiler designed to run natively on the WUT-4 architecture.

**Primary Goal:** Self-hosting - the compiler must be small enough to compile itself while running on WUT-4's 64KB code space.

**Language Specification:** https://docs.google.com/document/d/1hgsayGjZJc6WUVjSEsPRWVxPeXkVFLKpRCx5jc5hrx8/edit?usp=sharing

## Why YAPL?

YAPL is specifically designed to be compiled by a small, simple compiler that fits within WUT-4's memory constraints:

- **Machine types only** (byte, int16, uint16) - no complex type system
- **No preprocessor** - constant expressions evaluated in lexer pass
- **Single-dimensional arrays** - simpler than multidimensional
- **Minimal operators** - compact parser and code generator
- **Simple visibility rules** - uppercase = public, lowercase = private
- **No string type** - byte arrays only (BCPL-style)

## Compiler Architecture

The compiler uses a **multi-pass pipeline** with externalized state between passes. Each pass is a separate program that reads input files and writes output files, keeping each pass small enough to fit in 64KB.

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
│ - Output: IR (three-address code or SSA) │
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

- **Types**: byte, int16, uint16, @byte, @int16, @uint16, block32, block64, structs, arrays
- **Control flow**: if/else, while
- **Operators**: Arithmetic, bitwise, logical, comparison
- **Constants**: Compile-time expression evaluation
- **Visibility**: Uppercase = public, lowercase = private
- **Directives**: #if, #else, #endif, #line, #file

### Not Yet Specified

- Function definitions and calls
- Return statements
- Calling convention (stack frames, argument passing)
- Module/import system

## Inter-Pass File Formats

**To Be Defined:**
- Token stream format (binary? text? s-expressions?)
- AST representation
- Symbol table format
- IR format

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

**Status**: Design phase

**Completed:**
- Language specification (v0.1)
- Compiler architecture design
- Project structure

**Next Steps:**
1. Extend spec to cover functions and calling conventions
2. Define inter-pass file formats
3. Implement Pass 1 (Lexer + Constant Evaluator) in Go
4. Build test framework for Pass 1

## Related Documentation

- **WUT-4 Architecture**: `../emul/README.md`
- **WUT-4 Assembler**: `../asm/README.md`
- **YAPL Language Spec**: https://docs.google.com/document/d/1hgsayGjZJc6WUVjSEsPRWVxPeXkVFLKpRCx5jc5hrx8/edit?usp=sharing

## Implementation Notes

This is a long-term project requiring patience and methodical development. The multi-pass architecture with externalized state is specifically chosen to make the project tractable despite the ambitious self-hosting goal.

Each pass should be developed, tested, and validated independently before moving to the next pass. The intermediate file formats are as important as the code itself.
