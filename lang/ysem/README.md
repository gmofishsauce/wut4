# YAPL Semantic Analyzer (Pass 3)

## Overview

The semantic analyzer is Pass 3 of the YAPL compiler pipeline. It reads the AST output from Pass 2 (parser), performs semantic analysis including type checking, and generates intermediate representation (IR) code.

## Usage

```bash
# As part of the pipeline
cat source.yapl | ../ylex/ylex source.yapl | ../yparse/yparse | ./ysem > output.ir

# Or with files
./ysem < ast_input.txt > output.ir
```

## Input Format

The semantic analyzer reads the parser's AST output from stdin. See `../yparse/README.md` for the parser output format.

## Output Format

The semantic analyzer writes IR to stdout. See `../IR_FORMAT.md` for the complete IR specification.

### Quick Reference

```
#ir 1
#source filename.yapl

DATA name visibility type size [init]
CONST name visibility type value

STRUCT name size align
  FIELD name offset type
ENDSTRUCT

FUNC name
  VISIBILITY PUBLIC|STATIC
  RETURN type
  PARAMS n
    PARAM name type index
  LOCALS nbytes
    LOCAL name type offset
  FRAMESIZE n

  ; IR instructions
  t0 = CONST.W 0x000A
  t1 = LOAD.W [SP+0]
  t2 = ADD.W t0, t1
  STORE.W [SP+2], t2
  RETURN t2

ENDFUNC
```

## Semantic Analysis

### Type Checking

- Verifies type compatibility in assignments
- Checks operand types for all operators
- Validates function call arguments match parameters
- Ensures return statements match function return type

### Symbol Resolution

- Resolves all identifier references to declarations
- Detects undefined and duplicate identifiers
- Handles scoping (global vs. local)

### Implementation Limits

| Limit | Value |
|-------|-------|
| Max parameters per function | 16 |
| Max local variables per function | 32 |
| Max stack frame size | 256 bytes |
| Max struct fields | 32 |
| Max nesting depth | 16 |

## Error Handling

Errors are written to stderr:
```
filename:line: error: message
```

On error, no IR is written to stdout and exit code is 1.

## Building

```bash
GO111MODULE=off go build -o ysem .
```

## Testing

```bash
# Test with a simple program
echo 'func int16 Main() { return 42; }' | \
  ../ylex/ylex test.yapl | ../yparse/yparse | ./ysem
```
