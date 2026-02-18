# ysem Code Review

## Summary

`ysem` is Pass 3 of the YAPL compiler: it reads the yparse AST, type-checks it, and emits
three-address IR.  The overall architecture is sound — a clean reader/analyzer/IR-generator
split, reasonable use of Go interfaces, and a well-defined IR format.

However, **the reader (`reader.go`) has at least five confirmed format mismatches against the
actual yparse output format**, several of which make large classes of programs completely
unparseable.  The IR generator has correctness gaps that would produce either invalid IR or
wrong code.  The existing test suite only covers one narrow error-path; none of the reader
mismatches are caught by it.

**Verdict: request changes.**  The critical reader bugs must be fixed before this pass can
compile any real YAPL program that uses for-loops, `->` field access, goto, or string
literals.

---

## §1 Critical Issues

### C1 — `STR` vs `STRLIT` keyword mismatch (reader.go)

**File:** `reader.go`, line 751 (`case "STRLIT":`) and `isExprKeyword` line 841.

yparse emits `STR "..."` for string literals in expressions (confirmed by running
`ylex | yparse` on a program with `return "hello"`):

```
RETURN 15
  STR "hello"
```

reader.go looks for `STRLIT` in the expression switch and in `isExprKeyword`.  Neither
`"STR"` appears in those lists.  Effect: every string literal expression silently produces
`nil` during AST reading.  The `isExprKeyword` miss also means that when a `RETURN`
statement peeks ahead to check for a return value, it will not recognise the `STR` line as
an expression and will mark the return as void — silently compiling a `return "hello"` as
a `RETURN` with no value.

**Fix:** rename `"STRLIT"` to `"STR"` in the `readExprFromLine` switch case and in
`isExprKeyword`.

---

### C2 — `ARROW` (→ field access) never handled (reader.go)

**File:** `reader.go`, `readExprFromLine` switch (line 742 ff.).

yparse emits `ARROW fieldname` (keyword is literally `ARROW`) for pointer-to-struct field
access (`p->x`), followed by the object expression as a child:

```
ARROW x
  ID p
```

reader.go has no `"ARROW"` case in the `readExprFromLine` switch.  It falls through and
returns `nil, nil`.  The object-expression line (`ID p`) is left un-consumed in the
stream, corrupting subsequent reads.

Additionally, `isExprKeyword` does not include `"ARROW"`, so peek-ahead logic that calls
`isExprKeyword` (e.g., in `RETURN` parsing) will also misclassify an ARROW line.

**Fix:** Add a `"ARROW"` case that mirrors `"FIELD"` but sets `IsArrow: true`.  See also C3.

---

### C3 — `FIELD` reads field name from the wrong line (reader.go)

**File:** `reader.go`, `readExprFromLine`, case `"FIELD"` (~line 813).

The actual yparse format for dot-access (`obj.field`) is:

```
FIELD fieldname
  <object expression>
```

The field name is on the **same line as the `FIELD` keyword** (`parts[1]`).  The object
expression is a child on subsequent line(s).

reader.go's current implementation:
1. Checks `parts[1] == "ARROW"` for `isArrow` — this will always be false here because
   `parts[1]` is the field name, not the string `"ARROW"`.
2. Reads the **object expression** correctly.
3. Then calls `r.nextLine()` a **second time** to read a "fieldname" line.  But this second
   call consumes whatever line comes *after* the object expression — which is typically the
   next statement.  The actual field name (`parts[1]`) is discarded.

Effect: the field name is taken from an unrelated subsequent line; that line is consumed
without being processed.  Every `.field` access corrupts the parse stream.

**Fix:** Use `fieldName = parts[1]` directly and remove the spurious second `r.nextLine()`
call.

---

### C4 — `FOR` statement entirely misparsed (reader.go)

**File:** `reader.go`, `readFor` (~line 693).

yparse emits `FOR` with explicit section headers `INIT`, `COND`, `POST`, `DO`:

```
FOR 8
  INIT
    ASSIGN
      ID val
      LIT 0
  COND
    BINARY LT
      ID val
      LIT 10
  POST
    ASSIGN
      ...
  DO
    EXPR 9
      ...
ENDFOR
```

reader.go's `readFor` instead expects three consecutive lines each of which is either
`EMPTY` or a bare expression keyword:

```go
for i := 0; i < 3; i++ {
    r.nextLine()
    line := strings.TrimSpace(r.line)
    if line != "EMPTY" {
        expr, _ := r.readExprFromLine(line, depth+1, lineNum)
```

Actual execution trace for the example above:
- `i=0`: reads `"INIT"` → not `"EMPTY"` → `readExprFromLine("INIT")` → nil → `stmt.Init = nil`
- `i=1`: reads `"ASSIGN"` → parses `ASSIGN{val, 0}` (consuming `ID val` and `LIT 0`) →
  **`stmt.Cond = AssignExpr{val=0}`** (wrong: this is the init expression)
- `i=2`: reads `"COND"` → nil → `stmt.Post = nil`

Body reading then starts at `"BINARY LT"`, which is not a statement keyword → silently
skipped.  The actual condition expression, post expression, and `DO` header are all
consumed and discarded.  The body statements (if any) come after `DO` and are read
correctly, but init=nil, cond=wrong, post=nil.

**Fix:** `readFor` needs to parse the `INIT`, `COND`, `POST`, `DO` section headers
explicitly before reading each sub-expression.

---

### C5 — `GOTO` label and line number are swapped (reader.go)

**File:** `reader.go`, `readStmt`, case `"GOTO"` (~line 587).

yparse emits `GOTO label linenum` — label **first**, line number **second**:

```
GOTO done 11
```

reader.go parses `parts[1]` as `lineNum` and `parts[2]` as `label`:

```go
if len(parts) > 1 {
    lineNum, _ = strconv.Atoi(parts[1])   // "done" → 0
}
if len(parts) > 2 {
    label = parts[2]                        // "11" → label = "11"
}
```

Effect: every `goto` targets the stringified line number (e.g., `"11"`) instead of the
intended label, and the line-number field is silently zeroed.

**Fix:** swap the assignments: `label = parts[1]`, `lineNum, _ = strconv.Atoi(parts[2])`.

---

### C6 — `&&` and `||` always evaluate both operands (ir.go)

**File:** `ir.go`, `genBinary` (~line 621).

```go
func (g *IRGen) genBinary(e *BinaryExpr) string {
    left  := g.genExpr(e.Left)   // ALWAYS emits code for left
    right := g.genExpr(e.Right)  // ALWAYS emits code for right
    t := g.newTemp()
    ...
    case OpLAnd:
        ...
        g.emitJumpZ(left, endLabel)
        g.emitJumpZ(right, endLabel)
```

Both operands are unconditionally evaluated before the switch.  The short-circuit jump
logic operates on already-computed registers, so the branch only skips assigning `1` to `t`
— it does not skip evaluation of the right side.

YAPL is explicitly C-like; the language spec says `&&` and `||` are short-circuit
operators.  This matters for:

- Side-effectful right operands (e.g., `f() && g()`).
- Null-pointer guards: `ptr != 0 && @ptr > 0` will always dereference `ptr`, crashing
  when `ptr == 0`.

**Fix:** For `OpLAnd`/`OpLOr`, generate the right-operand code *inside* a conditional
block after the left-side check.

---

## §2 Warnings

### W1 — String-literal expressions emit `ADDR _strN` referencing undefined labels (ir.go)

**File:** `ir.go`, `genExpr`, case `*LiteralExpr` (~line 521).

```go
if e.IsStr {
    g.emit("ADDR", t, fmt.Sprintf("_str%d", g.tempCount))
}
```

`g.tempCount` has already been incremented by `g.newTemp()`, so the label name may be
off-by-one.  More importantly, no `DATA` entry for `_str0`, `_str1`, etc. is ever emitted;
ygen will reference undefined labels.  Until C1 is fixed string literals in expressions
never reach this code, but the IR generator still needs a concrete plan for them.

---

### W2 — `ARROW` not in `isExprKeyword` (reader.go)

**File:** `reader.go`, `isExprKeyword` (~line 839).

Already noted under C2 but worth calling out separately: the peek-ahead in `RETURN`
parsing calls `isExprKeyword` to decide whether the next line is a return value.  `"ARROW"`
is not in that list, so `return p->field;` would always be parsed as a void return.

---

### W3 — `genIndex` and `genField` always emit `LOAD.W` regardless of element type (ir.go)

**Files:** `ir.go`, `genIndex` (~line 956) and `genField` (~line 963).

```go
g.emit("LOAD.W", t, fmt.Sprintf("[%s]", addr))
```

When the array element type or struct field type is `uint8` (byte), `LOAD.BU` should be
used instead.  `LOAD.W` on a byte-typed array would read two bytes instead of one,
producing wrong values.  The nearby `genStore` path does check for byte types
(`isByteType`), so the inconsistency is likely unintentional.

---

### W4 — `genIdentLoad` always emits `LOAD.W` for globals (ir.go)

**File:** `ir.go`, `genIdentLoad` (~line 600).

```go
if _, exists := g.analyzer.globals[name]; exists {
    g.emit("LOAD.W", t, fmt.Sprintf("[%s]", name))
```

Same issue as W3 — byte-type global variables would be loaded as words.

---

### W5 — `genAddrOf` for locals emits non-IR `ADD.W SP, n` (ir.go)

**File:** `ir.go`, `genAddrOf` (~line 773).

```go
g.emit("ADD.W", t, "SP", fmt.Sprintf("%d", offset))
```

This emits `t = ADD.W SP, 4` where `"SP"` is a bare string and `4` is a decimal integer.
Neither is a valid virtual register in the IR spec; `ADD.W` takes two virtual register
operands.  ygen would need undocumented special-casing to handle this form.  The IR spec
defines `[SP+n]` address forms for loads/stores but not for address-of.  Consider defining
a dedicated `FRAMEADDR n` or `LOCALADDR` IR instruction for this purpose, or establishing
a convention in IR_FORMAT.md.

---

### W6 — Logical-NOT uses bare `"0"` as a virtual-register operand (ir.go)

**File:** `ir.go`, `genUnary`, case `OpLNot` (~line 752).

```go
g.emit("EQ.W", t, operand, "0")
```

`"0"` is a string literal, not a virtual register.  The IR spec does not define inline
integer constants as instruction operands (only `CONST.W`/`CONST.B` load them into
registers).  ygen must handle this as a special case.

---

### W7 — `LOCALS` byte-count calculation uses `+2` for all types (ir.go)

**File:** `ir.go`, `Write` (~line 1036).

```go
for _, l := range f.Locals {
    if l.Offset + 2 > localBytes {
        localBytes = l.Offset + 2
    }
}
```

This assumes every local is 2 bytes wide.  A `uint8` local at offset 5 would report
`localBytes = 7` instead of `6`.  `IRLocal` doesn't store the type size, so a lookup
would be needed; alternatively, pass the byte-count directly from the `FuncDef.FrameSize`
or compute it from the type string.

---

### W8 — `lookupType` returns scalar type for `ArrayLen == -1` const arrays (analyzer.go)

**File:** `analyzer.go`, `lookupType` (~line 488).

```go
if c.ArrayLen > 0 {
    return &Type{Kind: TypeArray, ElemType: c.Type, ArrayLen: c.ArrayLen}
}
return c.Type
```

`-1` is the sentinel for "size inferred from initializer".  When `ArrayLen == -1`, the
condition `c.ArrayLen > 0` is false, so the scalar base type is returned instead of an
array type.  Any expression that takes the address of such a const array (or indexes it)
will see the wrong type.  Same issue exists at line 480 for local variables.

---

### W9 — `SETPARAM` instruction is not in the IR specification (ir.go)

**File:** `ir.go`, `genStore` (~line 877).

```go
g.emit("SETPARAM", "", fmt.Sprintf("%d", idx), value)
```

`SETPARAM` does not appear in `IR_FORMAT.md`.  If ygen handles it, the contract is
undocumented.  If ygen does not handle it, stores to function parameters silently
disappear.  Either document the instruction or use an existing IR form.

---

### W10 — `genConst` emits `BYTES 0` when `ArrayLen == 0` and `InitBytes` is non-empty (ir.go)

**File:** `ir.go`, `genConst` (~line 198).

```go
if c.ArrayLen != 0 || len(c.InitBytes) > 0 {
    arrayLen := c.ArrayLen
    if arrayLen == -1 && len(c.InitBytes) > 0 {
        arrayLen = len(c.InitBytes)
    }
    ...
    irType = fmt.Sprintf("BYTES %d", arrayLen)  // "BYTES 0" if arrayLen still 0
```

If `c.ArrayLen` is `0` (e.g., the parser sets it to 0 for inferred-size arrays, as
`output.go` line 98 shows) **and** `c.InitBytes` is non-empty, the sentinel check
`arrayLen == -1` is never true, `arrayLen` stays 0, and we emit `BYTES 0` with `size = 0`.
The fix is to also handle the `arrayLen == 0 && len(c.InitBytes) > 0` case identically to
`arrayLen == -1`.

---

### W11 — `Type.Size()` returns 0 for `TypeStruct` (ast.go)

**File:** `ast.go`, `Type.Size()` (~line 159).

```go
case TypeStruct:
    return 0 // need to look up in struct table
```

This method is called when computing array sizes (`t.ElemType.Size() * t.ArrayLen`) and
sizeof expressions.  A `sizeof(struct_t)` always returns 0; a `struct_t[10]` array has
computed size 0.  A lookup in `Analyzer.structs` is needed; the method signature may need
to change or a separate helper should be added.

---

### W12 — No enforcement of implementation limits (analyzer.go)

**File:** `analyzer.go`, `typeCheckFunc` and `buildSymbolTables`.

`IR_FORMAT.md` documents hard limits: 16 parameters, 32 locals, 256-byte frame, 32 struct
fields, 16 nesting depth.  None of these are checked.  Exceeding them will silently
produce IR that ygen cannot handle.

---

### W13 — `readConst` hardcodes `Uint16Type` for all scalar constants (reader.go)

**File:** `reader.go`, `readConst` (~line 234).

```go
return &ConstDef{
    Name:  parts[1],
    Type:  Uint16Type, // constants default to uint16 for now
    Value: value,
}, nil
```

yparse's `CONST` line carries only name and value, not a type string, so the type must
be inferred or the format extended.  In the meantime, an `int16` constant will be
type-checked as `uint16`, which could suppress correct type-mismatch errors (e.g., assigning
a negative constant to a `uint16` would silently pass).

---

### W14 — Symbol-table errors lack source location (analyzer.go)

**File:** `analyzer.go`, `buildSymbolTables` (~line 97).

Duplicate-definition errors use `a.error(...)` (no file/line) rather than `a.errorAt(...)`.
The specified error format is `filename:line: error: message`; these errors omit both.

---

## §3 Test Coverage Gaps

The existing `TestImplicitIntConversionRejected` is the only test.  It tests one
type-checking path (uint16 → int16 assignment rejection) and passes today.  None of the
five critical reader bugs (C1–C5) are exercised.

The following test scenarios would each catch a distinct critical bug.  I've listed them
as potential regression tests; **please confirm before I write any of them**, per the review
instructions.

| # | Scenario | Bug detected |
|---|----------|--------------|
| T1 | Function containing a `for` loop — check that init, cond, post all appear in the IR | C4 |
| T2 | `return "literal-string"` — check that ysem succeeds and IR contains a LOAD/ADDR for the string | C1 |
| T3 | `p->field` access — check that IR contains field offset arithmetic | C2, C3 |
| T4 | `goto label` — check that the JUMP target in IR equals the label name | C5 |
| T5 | `f() && g()` where `g()` has a side effect — check that `g` is not called when `f` is 0 | C6 (behavioral) |

---

## §4 Nits

**N1 — Redundant `bufio.NewReader` wrapping (main.go line 14).**
`NewASTReader` accepts `io.Reader` and wraps it in `bufio.Scanner` internally; passing a
pre-wrapped `bufio.NewReader` just adds a second buffer layer.  Use
`NewASTReader(os.Stdin)` directly.

**N2 — Dead branch in `typesCompatible` (analyzer.go ~line 518).**
The comment "Allow integral types to mix" on the `t1.Kind != t2.Kind` path is misleading:
all integral types share `Kind == TypeBase`, so two integrals can never differ in `Kind`.
The branch is unreachable for integral-vs-integral comparisons.

**N3 — `CONST.W` used for byte literals (ir.go ~line 525).**
The IR spec defines both `CONST.W` and `CONST.B`.  Byte-typed integer literals emit
`CONST.W` instead of `CONST.B`.

**N4 — Errors silently discarded with `_` throughout reader.go.**
Many `r.readStmt(line, depth+1)` call-sites use `s, _ :=`.  Propagating errors up would
make malformed-input diagnostics much more actionable.

**N5 — `bufio.Scanner` default 64 KB token limit (reader.go).**
Very long string initializers in the AST stream could hit the scanner's default buffer
limit.  Consider calling `r.scanner.Buffer(make([]byte, 1<<20), 1<<20)` in
`NewASTReader`.

---

## §5 What Works Well

- The three-phase structure (build symbol tables → type-check → generate IR) is clean and
  easy to follow.
- `typesCompatible` handles array-to-pointer decay and `@void` wildcard correctly.
- `adaptLiteralToType` is a nice pattern that eliminates noisy explicit-cast requirements
  for unambiguous literals.
- The loop-label stack (`loopLabels`/`loopCont`) correctly handles nested loops and
  properly places the `continue` target before the post-expression for `for` loops.
- `formatInitBytes` correctly escapes non-printable bytes, quotes, and backslashes.
- The test harness in `ysem_test.go` builds all three upstream passes from source, which
  is the right approach — it tests the real pipeline rather than a synthetic AST.
