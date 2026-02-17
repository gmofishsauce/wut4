# Code Review: `ylex` — YAPL Lexer (Pass 1)

Reviewed against: `ylex/lexer.go` (1233 lines), `ylex/lexer_test.go`, all `testdata/` files, and `yapl_grammar.ebnf`.

---

## Summary

The lexer is a clean, readable, hand-rolled Go program that correctly handles YAPL's main complexity: constant expression folding at lex time, conditional compilation, and the special-cased `const`/`var`/`struct` declaration forms. The structure is sound and the main loop is easy to follow. **Request changes** on the warning items before this is used as a reference implementation for self-hosting, as several of the issues below will be harder to diagnose in a YAPL-compiled binary than they are now.

---

## Warnings

### W1 — Unterminated block comment silently accepted (`skipWhitespace`, lines 116–130)

```go
for !(l.peek() == '*' && l.peekN(1) == '/') && l.peek() != 0 {
    l.advance()
}
if l.peek() != 0 {
    l.advance() // skip *
    l.advance() // skip /
}
```

When EOF is reached inside a `/* ... */` comment, the loop exits via `l.peek() == 0`, the `if` branch is skipped, and the function returns without error. The token stream ends silently mid-file; downstream passes will see a truncated input and report confusing errors far from the actual mistake.

**Fix:** After the loop, check whether `l.peek()` is zero and call `l.error("unterminated block comment")` if so.

---

### W2 — Identifier length limit not enforced (`scanIdentifier`, lines 174–180)

The grammar says `identifier = letter (letter | digit)*` with a note "max 15 chars". `scanIdentifier` reads until the pattern breaks with no length check. A 20-character identifier passes through silently.

This matters for self-hosting: a YAPL-compiled lexer will need the limit enforced somewhere. If it isn't, identifiers that appear to match in the source may be stored or compared as different strings depending on whether a future self-hosted pass truncates them.

**Fix:** Add a length check with `l.error(...)` if the identifier exceeds 15 characters.

---

### W3 — Negative shift count panics the process (`parseConstMult`, lines 661–668)

```go
} else if ch == '<' && l.peekN(1) == '<' {
    l.advance()
    l.advance()
    left = left << l.parseConstUnary()
```

`parseConstUnary` returns `int64`. In Go (since 1.13), a signed shift count is legal, but if the count is negative the runtime panics. A source line like:

```
#if 1 << -1
```

kills the lexer process with a stack trace instead of a clean error message. Same applies to `>>`.

**Fix:** Capture the shift count, check `if count < 0 { l.error("negative shift count") }`, before applying it.

---

### W4 — Missing semicolon silently accepted in `var` declarations (`handleVarDecl`, lines 1115–1119)

`handleConstDecl` (line 1037) errors on a missing `;`:
```go
if l.peek() != ';' {
    l.error("expected ';' after const declaration")
}
```

`handleVarDecl` uses a soft check:
```go
if l.peek() == ';' {
    l.advance()
    l.emitToken(PUNCT, ";")
}
```

A `var` with a missing semicolon produces no error from the lexer. The parser sees the next token instead of `;` and will emit a confusing error.

**Fix:** Mirror the `const` handler — error on missing `;` rather than silently skipping it.

---

### W5 — `struct` type specifier misparsed in `handleConstDecl` / `handleVarDecl`

Both handlers read the type name as a single identifier (e.g., `handleConstDecl` lines 960–968). The grammar allows `TypeSpecifier = "struct" identifier`, meaning `const struct Foo x = ...` is a valid declaration. In the current code:

1. `scanIdentifier()` reads `struct` → emitted as `KEY struct`. ✓
2. The handler then reads the *next* identifier as the variable name, getting `Foo` instead of `x`.
3. The actual variable name `x` is then seen where `[` or `=` is expected → garbled token stream.

This won't be hit until struct-typed constants are used, but it is a correctness gap relative to the grammar.

**Fix:** After reading the type keyword, if it is `struct`, read one more identifier as the struct tag name and emit it as `ID` before reading the variable name.

---

## Nits

### N1 — `l.error()` doesn't flush the output buffer (line 149–152)

```go
func (l *Lexer) error(msg string) {
    fmt.Fprintf(os.Stderr, "%s:%d: error: %s\n", l.filename, l.line, msg)
    os.Exit(1)
}
```

`l.output` is a `*bufio.Writer`. Any tokens already emitted but not yet flushed are silently discarded on exit. A downstream pass reading the pipe may block or see a truncated stream and report a secondary error that obscures the real one. This is the same buffered-writer-on-error problem that affects many pipeline compilers.

**Fix:** Call `l.output.Flush()` before `os.Exit(1)`, or (better) write a helper that flushes and exits so the pattern is easy to follow.

---

### N2 — `#asm` and unknown `#pragma` names are syntax-checked even in skipped blocks

`handleDirective()` is correctly called unconditionally (to handle nested `#if`/`#endif`), but `#asm`'s argument parsing and `#pragma`'s name validation also run when `l.skipping` is true. A valid program like:

```
#if 0
#pragma unknownThing
#endif
```

produces a fatal error even though the block is dead. GCC similarly errors on malformed preprocessor directives in dead blocks; YAPL's behavior is defensible, but it is worth documenting as intentional so it doesn't get "fixed" to silently swallow errors everywhere.

---

### N3 — Test coverage gaps

The existing tests are well-structured and cover the main paths. Missing coverage:

| Scenario | Risk |
|---|---|
| Unterminated `/* ... */` | W1 above is currently untested |
| `#if` nesting (3+ levels) | Stack logic is untested beyond 2 levels |
| `sizeof` in constant expression | `sizeofType` and the `sizeof(` parsing path are untested |
| Shift and bitwise operators in `#if` | `parseConstMult` shift/and paths untested |
| Identifiers > 15 characters | W2 above is untested |
| `#asm` in a `#if 0` block | N2 above is untested |
| Character literals with all escape types | `\a \b \f \r \v \x41` not in any test |

---

## What Works Well

- **Main loop structure.** The `Run()` loop is short and the branching is clear. Directives are processed unconditionally before the `l.skipping` gate — the right design for `#else`/`#endif` to work correctly inside false blocks.

- **Operator precedence in constant expressions.** The recursive-descent parser (`parseConst{Or,And,Cmp,Add,Mult,Unary,Primary}`) exactly mirrors the grammar's `ConstExpr` hierarchy. `||`/`&&`/bitwise operators are all at the correct levels relative to each other.

- **Multi-character operator scanning.** Trying longer operators first (lines 910–923) is the right approach and correctly handles `<<` vs `<`, `->` vs `-`, etc.

- **`emitLineDirective` deduplication.** Tracking `lastEmitLine` avoids emitting redundant `#line` markers when multiple tokens appear on the same line. This keeps the token stream compact, which matters for the 64KB per-pass target.

- **Division/modulo by zero guarded.** Lines 647–653 produce a friendly error instead of a panic. This is the right pattern; the shift case (W3) should match it.

- **`#pragma bootstrap` and `#pragma message` are correctly skipped in false blocks.** The test in `pragma.yapl` / `pragma.expected` verifies the common case.

- **Test harness design.** Building the binary into a temp directory and driving it as a subprocess is the right level of integration testing for a pipeline pass. It catches argument handling, process exit codes, and the exact output format, not just internal logic.

---

## Self-Hosting Notes

These are not bugs today but will become implementation problems when rewriting in YAPL:

- **`map[string]int64` for the constant table.** Go maps won't exist. A fixed-size open-addressed hash table or a linear-scan array (probably sufficient given typical YAPL program size) will be needed.
- **`strings.Builder`.** Every use needs a fixed-size stack buffer with overflow detection.
- **`bufio.Reader.Peek(n)`.** The `peekN` abstraction is clean. In YAPL, a ring buffer with a two-byte lookahead window would replace it with minimal complexity.
- **Recursive constant expression parser.** Each level of nesting is a stack frame. WUT-4's 64KB D-space is shared with the frame and the constant table; deep nesting of constant expressions (unlikely in practice) could overflow the stack. A depth limit check at the top of `parseConstExpr` would make the failure mode a clean error rather than a crash.
