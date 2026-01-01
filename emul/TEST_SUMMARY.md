# WUT-4 Emulator Testing - Implementation Summary

## What Was Implemented

A comprehensive multi-layered regression testing framework for the WUT-4 emulator.

## Files Created

### Test Infrastructure
- **`emul_test.go`** - Integration test harness for running binary test programs
- **`decode_test.go`** - Unit tests for instruction decoder (all 5 instruction formats)
- **`memory_test.go`** - Unit tests for MMU and memory operations
- **`build_and_test.sh`** - Build script to assemble test programs and run all tests
- **`TESTING.md`** - Comprehensive testing documentation
- **`TEST_SUMMARY.md`** - This summary

### Test Programs (Assembly)
Located in `testdata/` with subdirectories by category:

#### Arithmetic Tests (`testdata/arithmetic/`)
- **`add_basic.w4a`** - Tests ADD instruction with multiple scenarios:
  - Simple addition
  - Zero result (Z flag test)
  - Addition with carry
  - Negative result (N flag test)

#### Memory Tests (`testdata/memory/`)
- **`load_store.w4a`** - Tests LDW/STW operations:
  - Basic store and load
  - Store/load with offset
  - Multiple stores don't interfere
  - Register 0 behavior

#### Branch Tests (`testdata/branch/`)
- **`conditional.w4a`** - Tests branching:
  - BEQ (branch if equal/zero)
  - BNE (branch if not equal)
  - BR (unconditional branch)
  - BCS (branch if carry set)
  - BCC (branch if carry clear)

#### Exception Tests (`testdata/exceptions/`)
- **`alignment_fault.w4a`** - Tests exception handling:
  - Alignment fault on odd address access
  - Exception handler installation
  - Exception vector routing

## Testing Convention

All integration tests follow a standard convention:
- **Success**: Write `0x0000` to physical address `0xFFFE` (word `0x7FFF`)
- **Failure**: Write non-zero error code to same address

The test harness reads this location after execution to determine pass/fail.

## Test Coverage

### Unit Tests (14 test suites)

#### Decoder Tests (`decode_test.go`)
✅ Base instructions (LDW, STW, ADI) with sign extension
✅ XOP (3-operand): ADD, SUB, AND
✅ YOP (2-operand): LSP, TST
✅ ZOP (1-operand): NOT, NEG
✅ VOP (0-operand): HLT, EI, DI
✅ Branch instructions with immediate decode
✅ LUI with 10-bit immediate

#### Memory/MMU Tests (`memory_test.go`)
✅ Virtual-to-physical address translation
✅ Page permissions (RWX, Execute-only, Invalid)
✅ Context switching (kernel vs user contexts)
✅ Load/store word operations
✅ Load/store byte operations
✅ Alignment fault detection
✅ Register 0 reads as zero during execution

### Integration Tests (4 programs)
✅ Arithmetic operations with flag updates
✅ Memory load/store with offsets
✅ Conditional and unconditional branches
✅ Exception handling (alignment fault)

## How to Run Tests

### Quick Start
```bash
./build_and_test.sh
```

### Run Unit Tests Only
```bash
go test -v
```

### Run Specific Test Suite
```bash
go test -v -run TestDecode
go test -v -run TestMMU
go test -v -run TestIntegration
```

### Run Individual Test
```bash
go test -v -run TestDecodeXOP
```

### Build and Run Integration Tests Manually
```bash
# Build assembler
cd ../asm && go build -o asm . && cd ../emul

# Assemble a test
../asm/asm testdata/arithmetic/add_basic.w4a testdata/arithmetic/add_basic.out

# Run emulator on test (with trace if needed)
go build -o emul .
./emul -trace trace.txt testdata/arithmetic/add_basic.out
```

## Test Results (as of implementation)

```
=== Unit Tests ===
TestDecodeBase ............... PASS (4 subtests)
TestDecodeXOP ................ PASS (3 subtests)
TestDecodeYOP ................ PASS (2 subtests)
TestDecodeZOP ................ PASS (2 subtests)
TestDecodeVOP ................ PASS (3 subtests)
TestDecodeBranch ............. PASS (3 subtests)
TestDecodeLUI ................ PASS (2 subtests)
TestMMUTranslation ........... PASS (6 subtests)
TestMMUPermissions ........... PASS (5 subtests)
TestLoadStoreWord ............ PASS
TestLoadStoreByte ............ PASS
TestAlignmentCheck ........... PASS
TestContextSwitching ......... PASS
TestRegister0ReadsZero ....... PASS

Total: 14 test suites, 35 subtests
Status: ALL PASSING ✓
```

## Design Decisions

### Why Outcome-Based Over Trace-Based?
- **Resilient**: Won't break when implementation changes internally
- **Maintainable**: Easy to update tests for intentional behavior changes
- **Fast**: No parsing of large trace files
- **Clear**: Simple pass/fail semantics

Traces can still be generated with `-trace` flag for debugging.

### Why Both Unit and Integration Tests?
- **Unit tests**: Fast, focused, good for edge cases and regression detection
- **Integration tests**: End-to-end validation, catch interaction bugs

Together they provide defense-in-depth.

### Test Program Memory Convention
Using address `0x7FFF` for test results is a convention that:
- Doesn't interfere with normal program operation (far from code/data)
- Is easy to check from the test harness
- Allows error codes to be returned (0 = success, non-zero = which test failed)

## Future Enhancements

### More Integration Tests
- [ ] All arithmetic instructions (SBB, ADC, NEG, etc.)
- [ ] Shift operations (SRA, SRL)
- [ ] Logical operations (XOR, OR, NOT)
- [ ] Byte operations (LDB, STB)
- [ ] Subroutine calls (JAL, JI, return patterns)
- [ ] All exception types (page fault, syscall, illegal instruction)
- [ ] Kernel/user mode switching
- [ ] SPR access (read/write special registers)
- [ ] I/O operations (console in/out)

### More Unit Tests
- [ ] Flag computation edge cases (overflow detection)
- [ ] Exception handler flow
- [ ] SPR access permissions
- [ ] Sign extension correctness
- [ ] Immediate value ranges

### Advanced Testing
- [ ] Fuzzing: Random instruction generation
- [ ] Golden trace files: Optional for critical functionality
- [ ] Performance benchmarks: Track emulator speed
- [ ] Coverage reports: Measure instruction coverage
- [ ] CI Integration: Automated testing

## Key Insights from Implementation

1. **Register 0 behavior**: The gen[x][0] array can be written but instructions read it as 0
2. **Double fault protection**: Exceptions in kernel mode with interrupts disabled halt the CPU
3. **Exception flow**: Exceptions set pendingException flag, handled in next cycle
4. **MMU design**: Separate code/data translation with context-based page tables
5. **Alignment**: Word operations must use even addresses

## Documentation

See `TESTING.md` for comprehensive testing guide including:
- Detailed explanation of test structure
- How to write new tests
- Troubleshooting guide
- Design rationale

## Conclusion

The testing framework provides:
- ✅ Systematic coverage of emulator functionality
- ✅ Fast unit tests for quick iteration
- ✅ Integration tests for end-to-end validation
- ✅ Easy extensibility for new tests
- ✅ Clear conventions and documentation

This foundation enables confident development and refactoring of the WUT-4 emulator.
