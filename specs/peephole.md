# Analysis: Peephole Optimization vs. Register Allocator Improvement
by Claude Code

## Understanding the Current Approach

The code generator uses a "fully spilled" register allocation strategy. Every virtual register (t0, t1, t2, etc.) is assigned a stack slot, and the code generator:

1. Loads operands from stack into physical registers (R4, R5, R6)
2. Performs the operation
3. Stores the result back to stack

This creates "seams" - sequences like:
```asm
STW R4, [SP+8]     ; store result of previous operation
LDW R4, [SP+8]     ; immediately reload the same value for next operation
```

Or when copying between virtual registers:
```asm
LDW R4, [SP+8]     ; load source
STW R4, [SP+12]    ; store to dest
LDW R5, [SP+12]    ; next operation reloads from dest
```

## Option 1: Peephole Optimizer

A peephole optimizer examines small windows (2-4 instructions) and applies pattern-based transformations.

**Applicable patterns for this code generator:**

| Pattern | Replacement |
|---------|-------------|
| `STW Rx, [SP+N]; LDW Ry, [SP+N]` | `MV Ry, Rx` (eliminates memory round-trip) |
| `STW Rx, [SP+N]; LDW Rx, [SP+N]` | (eliminate load entirely) |
| `LDI Rx, 0` | `MV Rx, R0` (smaller encoding possible via ADI) |
| `MV Rx, Rx` | (eliminate) |
| `LDI Rx, K; ADD Ry, Rz, Rx` where K is small | `ADI Ry, Rz, K` |

**Advantages:**
- Simple to implement - works as a post-pass on generated assembly
- Orthogonal - no changes to the main code generator needed
- Low risk - existing code generation remains unchanged
- Historical precedent - the original PCC and Ritchie's C compiler used this successfully
- Catches patterns beyond just register allocation (constant folding, strength reduction)
- Easier to test - clear input/output pattern matching

**Disadvantages:**
- Limited scope - can only see local patterns, not value lifetimes
- Misses larger opportunities - can't reason about whether a value is used later
- Multiple passes may be needed - one optimization may enable another
- Won't reduce register pressure - still limited to using R4/R5/R6 as scratch

## Option 2: Improved Register Allocator

A proper register allocator would track value lifetimes and assign virtual registers to physical registers when possible, spilling only when necessary.

**Algorithm options (simplest to most complex):**

1. **Local allocation** - keep values in registers within basic blocks, spill at boundaries
2. **Linear scan** - single pass through code assigning registers based on live ranges
3. **Graph coloring** - build interference graph, color with K colors (registers)

**Advantages:**
- More effective - keeps values in registers across multiple operations
- Better code quality overall - fewer memory operations
- Addresses root cause rather than patching symptoms

**Disadvantages:**
- Significant implementation complexity
- Requires liveness analysis infrastructure
- More invasive - requires restructuring the code generator
- Register assignment bugs are subtle and hard to debug
- The WUT-4 has only 7 usable registers (R0 is zero, R7 is SP), limiting potential gains

## Historical Context

The early C compilers faced exactly this situation. The Portable C Compiler (PCC) by Steve Johnson used a simple register allocator combined with a peephole optimizer. Dennis Ritchie's original PDP-11 C compiler also included a peephole pass. These practical solutions worked well for the hardware constraints of the time.

Graph coloring register allocation (Chaitin, 1981) and linear scan (Poletto & Sarkar, 1999) came later and were developed for machines with more registers where the additional complexity paid off.

The WUT-4, being a 16-bit architecture with limited registers, is well-suited to the historical approach.

## Recommendation: Hybrid Approach

A **two-phase strategy** is recommended:

### Phase 1: Implement a Peephole Optimizer

Start here because:
1. It's low-risk and can be added without modifying working code
2. Implementation is straightforward (pattern matching on instruction stream)
3. Provides immediate, measurable improvement
4. The data gathered (which patterns occur most frequently) informs future decisions
5. Even with a better allocator later, the peephole optimizer remains useful

### Phase 2: Limited Register Allocation (if needed)

If the peephole optimizer proves insufficient, implement a simple "local" allocator:
1. Track which values are in registers within each basic block
2. Reuse registers when their previous value is dead
3. Spill to stack at basic block boundaries (branches, labels, calls)

This is far simpler than full graph coloring but catches most short-lived temporaries, which are the majority of cases.

## Implementation Sketch for Phase 1

The peephole optimizer would:
1. Read generated assembly into a list of instruction structures
2. Scan the list with a sliding window of 2-3 instructions
3. Match patterns and apply replacements
4. Repeat until no more optimizations apply (fixed-point)
5. Output the optimized assembly

A simple implementation might be 200-400 lines of Go code.

## Expected Impact

Based on the code patterns observed in `codegen.go`:
- Most binary operations (add, sub, and, or, xor) generate store-then-load sequences
- The `genCopy` function always stores then the next operation loads
- Function argument setup and result handling create additional seams

A peephole optimizer could reasonably eliminate 20-40% of load/store operations in typical code. A proper register allocator might achieve 50-70% reduction, but with significantly more implementation effort.

## Conclusion

For a project of this scale, the peephole optimizer approach offers the best return on investment. It was good enough for the C compilers that bootstrapped Unix, and it should be good enough for the WUT-4. If the results are unsatisfactory, the path to a simple local register allocator remains open, and the peephole optimizer will still provide value as a cleanup pass.
