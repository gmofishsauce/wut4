# WUT-4 Emulator Testing Guide

This document describes the regression testing strategy for the WUT-4 emulator.

## Overview

The testing approach uses multiple layers to ensure comprehensive coverage:

1. **Integration Tests** - End-to-end tests using assembly programs
2. **Unit Tests** - Component-level tests for decoder, MMU, and memory
3. **Test Harness** - Infrastructure for running and validating test programs

## Quick Start

To run all tests:

```bash
./build_and_test.sh
```

This script will:
1. Build the assembler
2. Assemble all test programs in `testdata/`
3. Build the emulator
4. Run unit tests
5. Run integration tests

## Test Structure

### Integration Tests

Integration tests are located in `testdata/` organized by category:

```
testdata/
├── arithmetic/     # Arithmetic operations and flag updates
├── memory/         # Load/store operations
├── branch/         # Conditional and unconditional branches
└── exceptions/     # Exception handling
```

Each test is an assembly program (`.w4a`) that gets assembled to a binary (`.out`).

#### Test Convention

All integration test programs follow this convention:

- **Success**: Write `0x0000` to physical address `0xFFFE` (word address `0x7FFF`)
- **Failure**: Write a non-zero error code to the same address

The test harness reads this address after execution to determine pass/fail.

#### Example Test Program

```asm
; Test basic ADD instruction
.bootstrap

start:
    ldi r1, 5
    ldi r2, 7
    add r3, r1, r2      ; r3 = 5 + 7 = 12
    ldi r4, 12
    tst r3, r4
    bne fail            ; Branch if not equal

success:
    ; Write 0x0000 to test result address
    ldi r1, 0
    lui r2, 0x1FF
    adi r2, r2, 0x3F    ; r2 = 0x7FFF
    stw r1, r2, 0
    hlt

fail:
    ldi r1, 1           ; Error code 1
    lui r2, 0x1FF
    adi r2, r2, 0x3F
    stw r1, r2, 0
    hlt
```

### Unit Tests

Unit tests are written in Go and test individual components:

#### `decode_test.go`
Tests instruction decoding for all 5 instruction formats:
- Base instructions (LDW, STW, ADI, LUI, BRx, JAL)
- XOP (3-operand): ADD, SUB, AND, OR, XOR, etc.
- YOP (2-operand): LSP, SSP, TST, etc.
- ZOP (1-operand): NOT, NEG, SRA, SRL, etc.
- VOP (0-operand): HLT, EI, DI, RTI, etc.

Run decoder tests:
```bash
go test -v -run TestDecode
```

#### `memory_test.go`
Tests memory management and MMU:
- Virtual to physical address translation
- Page permissions (RWX, Execute-only, Invalid)
- Context switching between kernel and user modes
- Load/store operations (word and byte)
- Alignment checking
- Register 0 always reads as zero

Run memory tests:
```bash
go test -v -run "Test(MMU|Memory|LoadStore|Alignment|Context|Register0)"
```

### Integration Test Harness (`emul_test.go`)

The test harness provides infrastructure for running test programs:

- `runTestBinary(t, path)` - Loads and runs a binary, returns test result
- `TestIntegration()` - Discovers and runs all tests in `testdata/`
- Individual test functions for specific categories

The harness enforces a cycle limit (100,000 cycles) to prevent infinite loops.

## Writing New Tests

### Adding an Integration Test

1. Create an assembly file in the appropriate `testdata/` subdirectory:
   ```bash
   testdata/arithmetic/my_test.w4a
   ```

2. Write the test following the convention (write result to 0x7FFF)

3. Run the build script to assemble and test:
   ```bash
   ./build_and_test.sh
   ```

The test will be automatically discovered by `TestIntegration()`.

### Adding a Unit Test

1. Add a test function to the appropriate `*_test.go` file:
   ```go
   func TestMyFeature(t *testing.T) {
       cpu := NewCPU()
       // ... test setup and assertions
   }
   ```

2. Run with `go test`:
   ```bash
   go test -v -run TestMyFeature
   ```

## Current Test Coverage

### Integration Tests
- ✅ Basic arithmetic (ADD with flag updates)
- ✅ Load/store operations
- ✅ Conditional branches (BEQ, BNE, BCS, BCC)
- ✅ Alignment fault exception
- ⏳ More exception types (page fault, syscall)
- ⏳ Subroutine calls (JAL/JI)
- ⏳ Shifts and logical operations
- ⏳ Byte operations

### Unit Tests
- ✅ Decode all instruction formats
- ✅ MMU address translation
- ✅ Page permissions
- ✅ Context switching
- ✅ Load/store operations
- ✅ Alignment checking
- ⏳ Flag computations (carry, overflow)
- ⏳ Exception handling flow
- ⏳ SPR access permissions

## Design Rationale

### Why Integration Tests?

- Test the full emulator stack (fetch → decode → execute → memory)
- Catch regressions in overall behavior
- Easy to add new tests incrementally
- Human-readable when debugging (can use `-trace` flag)

### Why Unit Tests?

- Fast, focused tests for complex components
- Easy to achieve high code coverage
- Pinpoint exact failures
- Good for edge cases (e.g., sign extension, boundary conditions)

### Why Outcome-Based (vs Trace-Based)?

**Outcome-based** (current approach):
- ✅ Resilient to implementation changes
- ✅ Faster execution
- ✅ Clear pass/fail semantics
- ❌ Less visibility into failures

**Trace-based** (alternative):
- ✅ Detailed visibility into execution
- ✅ Catches unintended behavior changes
- ❌ Brittle when trace format changes
- ❌ Tedious to update for intentional changes

We chose outcome-based for maintainability, but traces can still be generated with the `-trace` flag for debugging.

## Troubleshooting

### Test binary not found
```
Test binary not found (run build_tests.sh first)
```
Run `./build_and_test.sh` to assemble test programs.

### Assembly errors
Check the assembler output. Common issues:
- Syntax errors in `.w4a` files
- Undefined labels
- Invalid immediate values

### Test failures
1. Run individual test with verbose output:
   ```bash
   go test -v -run TestName
   ```

2. For integration tests, run the binary manually with trace:
   ```bash
   ./emul -trace trace.txt testdata/arithmetic/add_basic.out
   cat trace.txt
   ```

3. Check the test result code (non-zero indicates which assertion failed)

## Future Enhancements

- **Fuzzing**: Random instruction generation to find edge cases
- **Golden trace files**: Optional trace comparison for critical tests
- **Performance benchmarks**: Track emulator speed over time
- **Coverage reports**: Measure instruction/branch coverage
- **CI Integration**: Automated testing on commit

## References

- [WUT-4 Architecture Specification](../asm/wut4arch.pdf)
- [WUT-4 Assembly Language](../asm/wut4asm.pdf)
