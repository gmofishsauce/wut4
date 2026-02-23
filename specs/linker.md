# Relocatable Object Files and a Linker for WUT-4/YAPL

## 1. Current Situation

The YAPL compiler pipeline produces a single absolute binary executable:

```
source.yapl → ylex → yparse → ysem → ygen → yasm → wut4.out
```

The assembler (`yasm`) operates in two passes over a single text file and
resolves all symbol references to their final addresses before writing output.
The output format (magic `0xDDD1`) contains a 16-byte header, a raw code
segment, and a raw data segment, with all addresses already baked in as
absolute 16-bit values.

This means every function and global variable referenced in a program must
appear in the same compilation unit. The workaround — concatenating sources
into combined files like `fib-combined.yapl` — is fragile and does not scale.

## 2. What Relocatable Compilation Enables

With relocatable object files and a linker:

- Each `.yapl` source file is compiled independently to a `.wo` (WUT-4 Object)
  file.
- The linker (`yld`) combines multiple `.wo` files into a single executable.
- A standard library can be pre-assembled and stored as `.wo` files, linked in
  on demand.
- The bootstrap/startup code (`_start`, SP initialization) lives in its own
  object file, linked once.
- Large programs can be split across files, reducing compile times when only
  one file changes.

## 3. The Relocation Problem on WUT-4

Before designing the format, it helps to enumerate exactly which instruction
sequences embed absolute addresses and therefore need relocation.

### 3.1 LDI (Load Immediate Address)

The `ldi` pseudo-instruction, as currently implemented, always emits two words:

```
LUI  rT, addr[15:6]     ; word1 = 0xA000 | (upper10 << 3) | rT
ADI  rT, rT, addr[5:0]  ; word2 = 0x8000 | (lower6  << 6) | (rT << 3) | rT
```

When `ygen` emits `ldi rX, GlobalVar` or `ldi rX, FunctionName`, the address
is a full 16-bit absolute value that cannot be known until link time.

### 3.2 JAL (Jump and Link — function calls)

The `jal` pseudo-instruction also always emits two words:

```
LUI  rT, addr[15:6]     ; word1 = 0xA000 | (upper10 << 3) | rT
JAL  rT, rS, addr[5:0]  ; word2 = 0xE000 | (lower6  << 6) | (rS << 3) | rT
```

Every inter-module function call requires a relocation.

### 3.3 Data Pointers in the Data Segment

When the compiler emits `.words label` in the `.data` section (e.g., a global
array of function pointers or a string pointer table), the 16-bit word value
is an absolute address that must be relocated.

### 3.4 Branch Instructions — No Relocation Needed

The `brz`, `brnz`, `brslt`, etc. instructions use a 10-bit PC-relative offset
(±512 bytes). Because `ygen` uses these only for intra-function control flow,
they always refer to labels within the same function, which will always be in
the same object file. Branch instructions **do not require relocation**.

### 3.5 Harvard Architecture Note

WUT-4 has separate I-space (code) and D-space (data), each 64KB. A function
address is an I-space address; a global variable address is a D-space address.
Relocation records must distinguish between these two address spaces, because
the linker assigns them independently.

## 4. Object File Format Design (WOF — WUT-4 Object Format)

The proposed format is binary, consistent with the existing executable format,
and deliberately minimal.

### 4.1 File Layout

```
[Header         — 16 bytes]
[Code section   — code_size bytes]
[Data section   — data_size bytes]
[Symbol table   — sym_count × 8 bytes]
[Relocation tbl — reloc_count × 8 bytes]
[String table   — string_table_size bytes, null-terminated names]
```

All multi-byte integers are little-endian (consistent with the rest of WUT-4).

### 4.2 Header (16 bytes)

| Offset | Size | Field              | Notes                                 |
|--------|------|--------------------|---------------------------------------|
| 0      | 2    | magic              | `0xDDD2` (distinct from `0xDDD1` exe) |
| 2      | 1    | version            | `1`                                   |
| 3      | 1    | flags              | bit 0 = bootstrap mode                |
| 4      | 2    | code_size          | bytes in code section                 |
| 6      | 2    | data_size          | bytes in data section                 |
| 8      | 2    | sym_count          | entries in symbol table               |
| 10     | 2    | reloc_count        | entries in relocation table           |
| 12     | 2    | string_table_size  | bytes in string table                 |
| 14     | 2    | reserved           | zero                                  |

### 4.3 Symbol Table Entry (8 bytes each)

| Offset | Size | Field        | Notes                                     |
|--------|------|--------------|-------------------------------------------|
| 0      | 2    | name_offset  | offset into string table                  |
| 2      | 2    | value        | offset within section (0 if undefined)    |
| 4      | 1    | section      | `0`=undefined, `1`=code, `2`=data         |
| 5      | 1    | visibility   | `0`=local (lowercase), `1`=global (upper) |
| 6      | 2    | reserved     | zero                                      |

A symbol with `section=0` (undefined) is an external reference — the linker
must find its definition in another object file.

YAPL's existing visibility convention maps directly: identifiers beginning with
an uppercase letter are `visibility=1` (globally visible); lowercase are
`visibility=0` (local to the translation unit). The linker enforces this: a
local symbol from object file A is never used to satisfy a reference from
object file B.

### 4.4 Relocation Table Entry (8 bytes each)

| Offset | Size | Field       | Notes                                      |
|--------|------|-------------|--------------------------------------------|
| 0      | 1    | section     | `0`=code section, `1`=data section         |
| 1      | 1    | type        | relocation type (see §4.5)                 |
| 2      | 2    | offset      | byte offset within the section             |
| 4      | 2    | sym_index   | index into symbol table                    |
| 6      | 2    | reserved    | zero                                       |

### 4.5 Relocation Types

| Code | Name            | Description                                              |
|------|-----------------|----------------------------------------------------------|
| 0x01 | `R_LDI_CODE`    | 2-word LUI+ADI sequence referencing a code-space address |
| 0x02 | `R_LDI_DATA`    | 2-word LUI+ADI sequence referencing a data-space address |
| 0x03 | `R_JAL`         | 2-word LUI+JAL sequence referencing a code-space address |
| 0x04 | `R_WORD16_CODE` | 16-bit word in data section, code-space address          |
| 0x05 | `R_WORD16_DATA` | 16-bit word in data section, data-space address          |

The code/data distinction is necessary because the linker tracks two
independent load-address spaces. A function pointer and a data pointer might
hold the same numeric value but refer to different physical memories.

**Linker patching procedure for `R_LDI_CODE` / `R_LDI_DATA` / `R_JAL`:**

At the given code offset, two consecutive words encode a 16-bit address `A`:
- word1 `= 0xA000 | (A[15:6] << 3) | rT` (the LUI; rT is in bits [2:0])
- word2 varies by type (ADI or JAL), with `A[5:0]` in bits [11:6]

The linker reads word1 to extract `rT`, computes the final address
`A = base + symbol.value`, then rewrites both words with the new `A`.
For `R_JAL`, word2 also contains `rS` in bits [5:3]; the linker preserves it.

**Linker patching procedure for `R_WORD16_CODE` / `R_WORD16_DATA`:**

At the given data section offset, write the final 16-bit address directly.

### 4.6 String Table

A contiguous block of null-terminated strings. Name offsets in the symbol
table are byte indices into this table. The first byte is always `\0` (the
null string, for unnamed/reserved entries).

### 4.7 Placeholder Values

When `yasm` emits an object file, it writes `0x0000` as the placeholder for
every unresolved external reference. The relocation table tells the linker
which words to patch. For internal references (symbols defined in the same
object file), `yasm` can resolve them immediately, emitting the section-relative
offset and recording no relocation — or it can always emit a relocation for
simplicity. The simpler approach (always emit relocations for any label
reference to a non-local label, or simply for any symbol with uppercase initial)
is recommended for the first implementation.

## 5. Changes to yasm

The assembler requires the largest set of changes. The good news is that
`yasm` already has most of the required infrastructure:

- It already tracks symbols with name, value, and segment.
- It already handles two-pass assembly with forward references.
- It already separates code and data segments.

### 5.1 New Command-Line Flag: `-c`

```
yasm -c foo.asm -o foo.wo    # produce object file
yasm foo.asm -o foo.out      # existing behavior: produce executable
```

### 5.2 Tracking External References

Currently, an undefined symbol at the end of pass 2 is a fatal error. In
`-c` mode, undefined symbols at the end of pass 2 become external references.
`yasm` must add them to the symbol table with `section=undefined`.

### 5.3 Emitting Relocations

Whenever `yasm` (in `-c` mode) evaluates a symbol reference in `genLDI`,
`genJAL`, or a `.words` directive and the symbol is global (uppercase) or
undefined, it records a relocation entry. The relocation type is determined
by:
- whether the instruction is LDI (`R_LDI_*`) or JAL (`R_JAL`)
- whether the symbol is in the code or data segment (`*_CODE` vs `*_DATA`);
  for undefined symbols, the type is inferred when the linker resolves it

A practical approach: record all symbol references during pass 2, then emit
the relocation table after the code and data buffers are finalized.

### 5.4 Writing the Object File

Replace `writeOutput` with a new `writeObjectFile` function that writes the
WOF format as described in §4.

### 5.5 Estimated Scope

The changes to `yasm` are self-contained and moderate in scope:
- ~100 lines to track relocations in the `Assembler` struct
- ~80 lines to collect relocations during `generateInstruction`
- ~100 lines to write the WOF header, symbol table, relocation table, and
  string table

Total for yasm: approximately **300 lines** of new or modified Go code.

## 6. The Linker (yld)

The linker is a new standalone program. Its algorithm is a straightforward
two-phase process.

### 6.1 Phase 1: Loading and Symbol Resolution

```
for each object file:
    read header, code, data, symbols, relocations, string table into memory
    for each global symbol (visibility=1, section≠undefined):
        add to global symbol table
        error if already defined by another object file
    for each undefined symbol (section=0):
        add to external reference list
after all files loaded:
    for each external reference:
        look up in global symbol table
        error if not found
```

### 6.2 Phase 2: Layout

```
code_offset[0] = 0
for i = 1 .. N-1:
    code_offset[i] = code_offset[i-1] + object[i-1].code_size
    (align to 2-byte boundary if necessary)

data_offset[0] = 0
for i = 1 .. N-1:
    data_offset[i] = data_offset[i-1] + object[i-1].data_size
    (align to 2-byte boundary if necessary)

total_code_size = sum of all code_size values
total_data_size = sum of all data_size values
```

### 6.3 Phase 3: Relocation

```
for each object file i:
    for each relocation r in object[i]:
        sym = r.sym_index → symbol table entry
        if sym.section == undefined:
            sym = global_symbol_table[sym.name]

        if sym.section == code:
            final_addr = code_offset[object_containing_sym] + sym.value
        else:
            final_addr = data_offset[object_containing_sym] + sym.value

        patch_offset = (r.section == code ? code_offset[i] : data_offset[i])
                       + r.offset

        apply patch at merged_buffer[patch_offset] using r.type
```

### 6.4 Phase 4: Output

Write the existing executable format (magic `0xDDD1`) with the merged code and
data buffers.

### 6.5 Estimated Scope

The linker is genuinely simple for WUT-4's address space:

- No complex section merging (just append)
- No segment alignment beyond 2 bytes
- Only five relocation types, all trivially patched
- 16-bit addresses that can never overflow
- No dynamic linking, no shared libraries, no GOT/PLT

Estimated size: **600–800 lines** of Go, including I/O, error handling, and a
symbol table implemented as a Go map.

## 7. Changes to ygen and the Bootstrap Problem

Currently `ygen` emits a `_start` block in every output file:

```asm
_start:
    ldi r7, 0xFFFF    ; initialize stack pointer
    jal main
    hlt
```

In a multi-file world, `_start` must appear exactly once. Two approaches:

**Option A — Explicit `#pragma bootstrap`** (already partially supported):
The programmer designates one file as the bootstrap file with `#pragma bootstrap`.
Only that file emits `_start`. All other files are compiled normally. The linker
is told which file provides the entry point.

**Option B — Separate runtime object**:
A pre-assembled `crt0.wo` provides `_start`. No YAPL source file emits it.
The driver links `crt0.wo` automatically unless the user passes `--no-crt0`.

Option B is cleaner and more conventional. Option A requires less new
infrastructure. Either works. Option A is recommended for the first
implementation since `#pragma bootstrap` already exists.

`ygen` change: when **not** in bootstrap mode, do not emit `_start`. This is a
small change (~10 lines) but requires care: currently the emitter always emits
the startup block. A new flag (or absence of `#pragma bootstrap`) should
suppress it.

## 8. Changes to the ya Driver

The driver needs two new modes:

```
ya -c source.yapl -o source.wo    # compile to object file
ya source1.wo source2.wo -o out   # link object files to executable
```

The existing `ya source.yapl` continues to work by compiling and immediately
linking (compile-and-link in one step, as today).

The driver changes are plumbing: pass `-c` to `yasm`, invoke `yld` at the end.
Estimated scope: **50–100 lines**.

## 9. Overall Complexity Assessment

| Component     | Change type | Estimated effort  |
|---------------|-------------|-------------------|
| yasm          | Modify      | ~300 lines        |
| yld (linker)  | New program | ~700 lines        |
| ygen          | Small fix   | ~10–20 lines      |
| ya driver     | Modify      | ~50–100 lines     |
| **Total**     |             | **~1100 lines**   |

This is a well-scoped project. The hardest part is not the linker algorithm
(which is simple) but getting the relocation emission in `yasm` correct —
specifically, knowing exactly which instructions contain symbol references that
will be external at link time, and recording the right relocation type
(code-space vs data-space).

## 10. Design Decisions and Trade-offs

### 10.1 Binary vs ASCII Object Format

All existing inter-pass formats in this compiler are ASCII. An ASCII object
file format is conceivable (hex-dump the code sections, describe relocations as
text lines), but binary is strongly preferable for a linker:
- Cleaner to parse
- Smaller files
- More natural for self-hosting (reading binary in YAPL is simpler than lexing
  a custom text format)
- Consistent with the existing executable format

### 10.2 Always-Relocatable vs Conditional

In the proposed design, `yasm -c` always emits relocations for any non-local
symbol reference, even if the symbol is defined in the same file. This is
slightly wasteful but simplifies the assembler: it doesn't need to distinguish
"will this symbol be visible to the linker?" It just always records the
relocation and lets the linker notice that the symbol is already resolved
within the same object file (or optimize by letting `yasm` resolve intra-file
references directly, adding relocations only for undefined symbols).

The simpler first approach: emit relocations only for undefined symbols.
Intra-file symbols are fully resolved by `yasm` as today.

### 10.3 Section Layout Order

The current executable format places code before data. The linker should
preserve this: all code sections first (object 0 code, object 1 code, …),
then all data sections. This matches what the emulator and any debugging tools
expect.

### 10.4 The Bootstrap Object and `_start`

For the self-hosting goal, having `_start` in a dedicated `crt0.wo` is more
correct. However, for near-term development, the `#pragma bootstrap` approach
requires the smallest change to the existing pipeline.

### 10.5 No Archive / Library Format

A library format (like Unix `.a` archives) would allow selective linking of
only needed object files. For WUT-4's 64KB address space this is unlikely to
matter. A simple list of `.wo` files on the command line is sufficient.

## 11. What This Does NOT Require

To be explicit about scope: this design does **not** require:
- Position-independent code (PIC) — WUT-4's 64KB fits easily in one binary
- Segment alignment beyond 2 bytes
- Weak symbols
- Common (BSS) symbols (though `.space` zero-fill is equivalent)
- Dynamic linking
- Debug information sections
- ELF or any standard format

## 12. Recommended Implementation Order

1. **Define the WOF format** precisely (this document serves as the spec).
2. **Modify yasm** to produce `.wo` files with `-c` flag.
3. **Write yld** (the linker).
4. **Fix ygen** to suppress `_start` when not in bootstrap mode.
5. **Update ya** to support `-c` and multi-file linking.
6. **Test** with a two-file program (e.g., split `fib-combined.yapl` into
   `fib.yapl` + `utoa.yapl` and link them).

---

*Analysis written 2026-02-22. No code changes have been made.*

Human comment: this document does not cover necessary modifications
That must be made to the emulator to allow it to load the new binary format.
These modifications are probably simple but they are necessary.
