# ygen Code Review

**Files reviewed:** `main.go`, `ir_types.go`, `ir_parser.go`, `regalloc.go`,
`codegen.go`, `emit.go`

---

## Summary

`ygen` is a clean, readable stack-machine code generator.  The overall
architecture — IR parser → CodeGen → Emitter — is well-structured, and the
large-frame prologue/epilogue handling is notably careful.  There are two real
bugs: one critical (wrong code generated for `>` comparisons) and one dormant
but broken (hex parsing in `parseInt`).  The entire `regalloc.go` file is dead
code.  Everything else is either a nit or a known architectural simplification.

**Verdict: request changes** (critical bug C1 must be fixed before shipping).

---

## Critical Issues

### C1 — `GT.S` and `GT.U` comparisons store an undefined value on the false path

**File:** `codegen.go` — `genCompare`, lines 1079–1091

**Problem.**  For `OpGtS` (and symmetrically `OpGtU`) the false-condition
branches jump directly to `doneLabel`, which is the final store instruction.
This bypasses the `ldi r6, 0` that is supposed to set the false result, so R6
retains its value from the prologue (the caller's saved R6) when `a <= b`.

Annotated output from `testdata/gts_compare.ir`:

```asm
    tst r4, r5
    brslt L_cmp_d1        ; a <  b (false) → jumps to store — R6 never set!
    brz   L_cmp_d1        ; a == b (false) → same bug
    br    L_cmp_t0        ; a >  b (true)
    ldi r6, 0             ; dead code: unreachable for GT.S
    br    L_cmp_d1
L_cmp_t0:
    ldi r6, 1             ; true result
L_cmp_d1:                 ; ← doneLabel — both false branches land here
    stw r6, r7, 4         ; stores stale R6 when a <= b
```

The correct fix is to introduce a `falseLabel` placed just before `ldi r6, 0`
and have the false-condition branches target it instead of `doneLabel`:

```go
case OpGtS:
    falseLabel := cg.emit.NewLabel("cmp_f")
    cg.emit.Brslt(falseLabel)   // a < b  → false
    cg.emit.Brz(falseLabel)     // a == b → false
    cg.emit.Br(trueLabel)       // a > b  → true
    // ... existing false/true label + ldi block; falseLabel goes before ldi r6,0
```

The same fix applies to `OpGtU` (replace `Brslt`/`Brz` with `Brult`/`Brz`).

The other four comparison operators (`EQ`, `NE`, `LT.*`, `LE.*`, `GE.*`) are
all correct: their false conditions fall through naturally to the `ldi r6, 0`.

**Regression tests:** `TestGtSFalsePath`, `TestGtUFalsePath` in `ygen_test.go`
(both currently **failing**).

---

## Warning Issues

### W1 — `parseInt` is broken for hex input

**File:** `ir_parser.go`, lines 407–416

```go
func parseInt(s string) int {
    s = strings.TrimPrefix(s, "0x")
    s = strings.TrimPrefix(s, "0X")
    if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") { // ← always false
        v, _ := strconv.ParseInt(s[2:], 16, 32)
        return int(v)
    }
    v, _ := strconv.ParseInt(s, 0, 32)
    return int(v)
}
```

After stripping the `0x`/`0X` prefix the second `HasPrefix` check can never be
true — it is dead code flagged by the static analyser.  The result:

| Input  | Got | Want |
|--------|-----|------|
| `"0x24"` | 24  | 36   |
| `"0xFF"` | 0   | 255  |

The fix is simply to remove the two `TrimPrefix` calls and let
`strconv.ParseInt(s, 0, 32)` do the right thing (it already handles the `0x`
prefix when `base == 0`):

```go
func parseInt(s string) int {
    v, _ := strconv.ParseInt(s, 0, 32)
    return int(v)
}
```

**Impact today:** all current callers pass decimal values
(`FRAMESIZE`, `LOCAL` offset, `PARAM` index, `STRUCT` size/align), so no
current IR triggers the bug.  It will bite the first time any of those fields
is written in hex.

**Note:** `parseInt64` (used for `CONST` values) and `parseValue` in
`codegen.go` (used for instruction operands) both use `strconv.ParseInt(s, 0,
…)` directly and are correct.

**Regression test:** `TestParseIntHex` in `ygen_test.go` (currently **failing**).

### W2 — `regalloc.go` is entirely dead code

**File:** `regalloc.go` (217 lines)

`CodeGen` never creates or references a `RegAllocator`.  The code generator
uses a different strategy: every virtual register gets a dedicated stack slot
(`virtRegSlots map[string]int`) and all values are loaded/stored through R4/R5/R6
as needed.  `RegAllocator` is a separate, fully implemented but disconnected
module.

Keeping dead code risks:
- Future maintainers assuming it is in use and relying on its behaviour.
- `spillAndAllocate` records a spill slot but emits no `stw` to write the
  register's value there — if `RegAllocator` were ever wired in, spills would
  silently lose data.

**Recommendation:** Delete `regalloc.go` until/unless a register-allocation
pass is planned, at which point it should be developed alongside the codegen
integration rather than ahead of it.

### W3 — Virtual-register spill space sized by register number, not count

**File:** `codegen.go`, lines 388–409

```go
virtRegSpace := (maxVirtReg + 1) * 2
```

`maxVirtReg` is the highest *index* seen in `tN` destination names.  If a
function uses `t0`, `t5`, and `t50`, `virtRegSpace` is 102 bytes even though
only 3 slots are needed.  Pass 3 currently numbers virtual registers
sequentially from 0, so this is not a problem today, but it is a latent
over-allocation if that ever changes.

`getVirtRegSlot` compounds this by computing the slot as `nextVirtSlot + n*2`
(using the register's number as an index into the array), so gaps in the
numbering waste frame space.

---

## Nits

**N1** — `ir_types.go` line 147: `R0 = 0 // Zero / LINK`.  LINK is SPR 0, not
R0.  R0 is the hardwired-zero general register.  Suggest: `// Hardwired zero`.

**N2** — `codegen.go` line 1259: a local `min(a, b int) int` function is
defined.  Go 1.21 introduced `min` as a builtin; the local definition shadows
it.  This is harmless but confusing.  The go.mod already declares `go 1.21`.
Remove the local definition.

**N3** — `codegen.go` line 35: `alignmentPadding int` is computed and
immediately set to 0 (line 429) but never read.  Remove the field.

**N4** — `ir_parser.go` line 290: the `default` case inside the function
header parser silently treats unrecognised header keywords as IR instructions.
This masks typos and malformed IR.  It should `return nil, fmt.Errorf(...)`.

**N5** — `emit.go` lines 113–135: `LdwLarge` and `StwLarge` are defined but
never called.  All callers use `emitLoadStack`/`emitStoreStack` in `codegen.go`.
Delete or use.

**N6** — Large-frame prologue `else` branch (`codegen.go` lines 492–496) uses
R5 as scratch to pre-save R4 when `preSaveOffset < -64`.  Since
`preSaveOffset` is always `-8` (derived from the fixed `savedRegSize + linkSaveSpace
= 8`), this branch is unreachable.  If it were reached, it would corrupt R5
before R5 is saved — a latent data-corruption bug.  Either add an assertion or
a comment explaining why it is unreachable.

---

## Questions

1. **`BYTE` data type in `genDataSection`**: single-byte globals (`type=="BYTE"`)
   fall through to the `default` case and emit `.bytes <value>`.  Does the
   assembler accept `.bytes` for a single byte?  If it expects `.byte`
   (singular), globals like `DATA x STATIC BYTE 1 0x05` would be mis-assembled.

2. **`SETPARAM` semantics**: `genSetParam` writes through to the caller's stack
   area for parameters with index ≥ 3 (`offset = cg.totalFrameSize +
   2*(paramIndex-3)`).  Is modifying caller-stack parameter slots part of the
   defined calling convention, or should it be a local copy?

3. **Bootstrap stack size**: `_start` sets SP to `0x1000` (4096).  For
   programs with deep call chains or large frames (e.g. `BigFunc` uses 194
   bytes per call), the 4 KB stack may be tight.  Is this a documented
   constraint, or should it be larger?
