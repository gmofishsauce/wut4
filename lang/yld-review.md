# Code Review: `lang/yld/`

## Summary

This is a clean, well-structured three-phase linker (symbol resolution → layout →
relocation) for the WUT-4 WOF format. The binary format parsing is correct, the
LUI+ADI and LUI+JAL bit-field arithmetic matches the spec and the assembler, and
the error reporting is consistent. The code is ready to use. There are no
correctness bugs on well-formed inputs, but a few reliability gaps exist for
corrupt or oversized input.

---

## Warning Issues

### W1. `layout()` silently overflows on large inputs — `linker.go:74`

```go
var codeOff, dataOff uint16
for _, obj := range ld.objects {
    obj.codeOffset = codeOff
    codeOff += obj.header.CodeSize   // wraps silently at 65535
```

`codeOff` is `uint16`. If the summed code sections exceed 65535 bytes, arithmetic
wraps silently and all subsequent `codeOffset` values are wrong — producing a
corrupt executable with no error message. WUT-4's 64KB address space means this
*could* happen. `relocate()` recomputes the same accumulation as `int`, so the
merged buffer would be the right size while the per-object offsets are wrong,
causing silent data corruption.

**Fix:** Accumulate in `int` (or `uint32`) inside the loop, error if the result
exceeds `0xFFFF`, then assign to `uint16`.

---

### W2. No check that `r.Offset` lies within the current object's section — `linker.go:124`

```go
patchOffset := int(patchBase) + int(r.Offset)
```

`patchLUIPlusADI`/`patchLUIPlusJAL` check `offset+4 <= len(mergedCode)` — but
`mergedCode` is the entire merged buffer, not just the current object's slice. A
corrupt `.wo` file with `r.Offset >= obj.header.CodeSize` causes the patch to
silently reach into the next object's code. The check should be:

```go
if int(r.Offset)+patchSize > int(obj.header.CodeSize) {
    return ..., fmt.Errorf("relocation offset 0x%04X out of section in %s", r.Offset, obj.path)
}
```

This applies to all relocation types (LDI, JAL, WORD16).

---

### W3. No opcode sanity check in patch functions — `linker.go:211, 230`

`patchLUIPlusADI` and `patchLUIPlusJAL` read `word1` to extract `rT` and
immediately overwrite it without checking that `word1` actually encodes a LUI
(opcode `0b101`, i.e. `0xA000..0xBFFF`). A mismatched relocation type — whether
from an assembler bug or a corrupt file — is silently accepted and the wrong
instruction fields are overwritten.

```go
if word1&0xE000 != 0xA000 {
    return fmt.Errorf("expected LUI at offset %d, got 0x%04X", offset, word1)
}
```

Similarly, `patchLUIPlusJAL` could verify `word2&0xE000 == 0xE000`.

---

### W4. Verbose output goes to stdout — `linker.go:49, 91, 169; main.go:58`

```go
fmt.Printf("  global %s: ...", ...)       // linker.go
fmt.Printf("Link successful: %s\n", ...)  // main.go
```

Build tool convention is that diagnostic/verbose output goes to stderr so it
doesn't pollute stdout when the tool is used in a pipeline. The
`"Link successful: ..."` print in `main.go` also fires unconditionally (without
`-v`), which is inconsistent with the guarded verbose prints elsewhere. Consider
making it conditional or moving it to stderr.

---

## Nits

**N1. `HEADER_SIZE` constant is not used in `reader.go`**

`output.go` defines `HEADER_SIZE = 16`, but `reader.go` uses the literal `16` at
`codeStart := 16`. These should share the constant. It belongs in `types.go`
where it is visible to both files.

**N2. No named constants for relocation section values**

The symbol table uses `SEC_UNDEF=0`, `SEC_CODE_WOF=1`, `SEC_DATA_WOF=2`. The
relocation table uses a *different* encoding: `0`=code section, `1`=data section.
The linker correctly uses the literal `0` at `linker.go:160` rather than (wrongly)
using `SEC_CODE_WOF`, but the asymmetry is a maintenance trap. Consider:

```go
const (
    RSEC_CODE = 0  // relocation record: refers to code section
    RSEC_DATA = 1  // relocation record: refers to data section
)
```

**N3. `ResolvedSym.name` is write-only**

The `name` field of `ResolvedSym` is set during symbol resolution but never read
back — only `value`, `section`, and `objIndex` are used in `relocate()`. Fine for
debuggability, but worth a comment explaining it is retained for future use or
verbose diagnostics.

---

## What's Well Done

- The single `strtabEnd > len(data)` bounds check in `reader.go` is sufficient
  because all intermediate section offsets are strictly increasing — elegant.
- Bit-field arithmetic in `patchLUIPlusADI` and `patchLUIPlusJAL` is correct and
  matches both the spec and `yasm/codegen.go`'s emission exactly. The
  `upper`/`lower` split and register preservation were verified for all 8 register
  values by the regression tests.
- The two-pass symbol resolution (collect globals, then verify undefined refs)
  cleanly separates concerns and gives good error messages.
- Guarding `copy(mergedData[obj.dataOffset:], obj.data)` on `DataSize > 0` is the
  right call.
- The `relocate()` path correctly handles both intra-file symbols (builds
  `ResolvedSym` on-the-fly from the local symbol table) and external references
  (looks up `globalSyms`).

---

## Regression Tests Written: `linker_test.go`

24 tests, all passing. Coverage:

| Area | Tests |
|------|-------|
| `readObjectFile` | Minimal, BadMagic, TooShort, WithCode, WithSymbol, WithRelocation |
| Linker phases | SingleObject (no code + with code), TwoObjects alignment, IntraFile JAL |
| Symbol resolution | JAL inter-file, LDI_Data inter-file, WORD16_Code inter-file |
| Error paths | UndefinedSymbol, DuplicateGlobal |
| Patch functions | ADI basic, nonzero lower, all registers, max address, out-of-bounds |
|                  | JAL basic, all rT×rS pairs, out-of-bounds |
| Output format | Header magic, sizes, reserved bytes, sections |

*Review written 2026-02-23.*
