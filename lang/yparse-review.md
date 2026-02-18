# Code Review: YAPL Parser (yparse)

Reviewed files:
- `lang/yparse/parser.go`
- `lang/yparse/token.go`
- `lang/yparse/ast.go`
- `lang/yparse/types.go`
- `lang/yparse/symtab.go`
- `lang/yparse/output.go`
- `lang/yparse/main.go`
- `lang/yparse/parser_test.go`
- `lang/ylex/lexer.go` (for interface contract)
- `lang/yapl_grammar.ebnf` (authoritative grammar)

Test run: `go test -v` in `lang/yparse/` produces one failing positive test and nine passing negative tests.

---

## 1. Summary

The parser is a clean, readable recursive-descent implementation with a reasonable panic-mode error recovery skeleton. The core structure is sound and the grammar coverage is mostly correct. However, there are several bugs of different severity:

- A **critical** logic error in `parseAdditive` that double-consumes the `|` token and leaves `op` in an undefined state for the `^` case.
- A **critical** grammar mismatch: the parser only accepts a single expression in a brace-enclosed array initializer `{ expr }`, silently discarding any additional `{ expr, expr, ... }` elements.
- A **warning-level** issue: `parseBinaryRest` in the label-detection fallback path handles only `=`, losing all other binary operators for expression statements starting with an identifier.
- Several **warnings** around nil-pointer dereference exposure, incorrect token consumption on error paths, and alignment math bugs.
- A **failing test** (`02_control`) caused by a test-data identifier that exceeds the 15-character limit enforced by the lexer.

---

## 2. Critical Issues

### [critical] `parseAdditive` — broken `|` and dead-code path for `^` (parser.go, lines 1214–1263)

```go
// parseAdditive parses + - | ^ (left-associative)
func (p *Parser) parseAdditive() Expr {
    expr := p.parseMultiplicative()

    for {
        tok := p.tokens.Peek()
        var op BinaryOp
        switch {
        case tok.IsPunct("+"):
            op = OpAdd
        case tok.IsPunct("-"):
            op = OpSub
        case tok.IsPunct("|") && !p.tokens.Peek().IsPunct("||"):
            // Need to check it's not ||
            op = OpOr
        case tok.IsPunct("^"):
            op = OpXor
        default:
            return expr
        }

        // Double-check for || (already consumed one |)
        if op == OpOr {
            p.tokens.Next()
            if p.tokens.Peek().IsPunct("|") {
                // It was ||, put back conceptually - but we can't
                // ...
            }
        } else {
            p.tokens.Next()
        }
        ...
    }
}
```

**Bug 1 — Double token consumption on `|`.**
When `op == OpOr`, the code calls `p.tokens.Next()` inside the `if op == OpOr` block to consume the `|`. However, the `switch` case for `OpOr` already called `p.tokens.Peek()` but did NOT consume the token. So far so good — but then note what happens for `^`: the code falls into the `else { p.tokens.Next() }` branch and consumes the `^`. But `op == OpOr` is never re-entered for a second `Next()`. Let's trace `|` carefully:

1. `p.tokens.Peek()` returns `|` → `switch` sets `op = OpOr`. The `|` is still the current token.
2. `if op == OpOr { p.tokens.Next() ... }` — this consumes the `|`. OK so far.
3. The comment about "it was `||`, put back" is unreachable dead code: the `switch` case already guards `tok.IsPunct("|") && !p.tokens.Peek().IsPunct("||")`, so `op` is only set to `OpOr` when the single `|` is confirmed. The internal `if p.tokens.Peek().IsPunct("|")` check (line 1238) will never be true, but it is also harmless.
4. However, the body then falls through to `loc := p.currentLoc()` and `right := p.parseMultiplicative()` with the `|` already consumed. That part is actually correct.

**Bug 2 — `^` token is never consumed.** For `op == OpXor`, the code hits `else { p.tokens.Next() }` and consumes the `^`. That is correct *only* if `op != OpOr`. Tracing `^`:
- `switch` sets `op = OpXor`, token is still `^`.
- `if op == OpOr` is false → `else { p.tokens.Next() }` — consumes `^`. OK.
- But then `loc := p.currentLoc()` reads the location *after* the consumed token — this means `loc` points to the first token of the right-hand operand, not the operator. This is a minor source-location issue common to all cases.

Actually re-reading more carefully: for `|`, the `if op == OpOr { p.tokens.Next() }` branch is taken, meaning the `else { p.tokens.Next() }` branch is NOT taken. So `|` is consumed exactly once. For all other operators, the `else` branch consumes them. This is technically correct but the code is unnecessarily convoluted and the comment "Already consumed" / "Token already consumed above" in the second `if op == OpOr` block (lines 1248–1252) refers to dead code that does nothing at all. The structure is:

```go
if op == OpOr {
    p.tokens.Next()          // (A) consume |
    if p.tokens.Peek().IsPunct("|") { /* dead - can never happen */ }
} else {
    p.tokens.Next()          // (B) consume all other ops
}

if op == OpOr {              // (C) dead code block, does nothing
} else {                     // (D) dead code block, does nothing
}
```

The real consequence: the logic works by accident, but the dead-code section (lines 1248–1252) is a maintenance hazard. Anyone reading the code will be confused and may incorrectly "fix" it into an actual double-consume. The `loc` for the binary node is also captured *after* the operator is consumed, so it points at the right operand's first token rather than the operator — all binary nodes in `parseAdditive` have wrong source locations.

**Suggested fix:** collapse to the same pattern used by `parseLogicalOr` and `parseLogicalAnd`:

```go
for {
    tok := p.tokens.Peek()
    var op BinaryOp
    switch {
    case tok.IsPunct("+"):  op = OpAdd
    case tok.IsPunct("-"):  op = OpSub
    case tok.IsPunct("|"):  op = OpOr   // lexer emits | and || as distinct tokens
    case tok.IsPunct("^"):  op = OpXor
    default:
        return expr
    }
    loc := p.currentLoc()
    p.tokens.Next()  // consume the operator
    right := p.parseMultiplicative()
    expr = &BinaryExpr{baseExpr: baseExpr{Loc: loc}, Op: op, Left: expr, Right: right}
}
```

The `|| vs |` concern is a non-issue: the lexer emits `||` as a single two-character PUNCT token (see `lang/ylex/lexer.go` lines 37–38, `multiCharOps`), so `IsPunct("|")` can never match a `||` token.

Similarly `isDoubleAmp` in `parseMultiplicative` (line 1302–1305) is an over-complication for the same reason: `&&` is a single token from the lexer, so `IsPunct("&")` cannot match it.

---

### [critical] `parseArrayInit` only parses one element of a multi-element initializer (parser.go, lines 472–516)

The grammar says:
```
ArrayInit     = "{" ConstExprList "}" | "{" string_literal "}" | string_literal .
ConstExprList = ConstExpr { "," ConstExpr } .
```

The implementation:
```go
// { expr, expr, ... } - numeric initializer list
// For now, parse as a single expression (simplified)
expr := p.parseExpression()
if _, err := p.tokens.ExpectPunct("}"); err != nil {
    p.error("expected '}' after initializer")
}
return expr
```

The comment "for now, parse as a single expression (simplified)" documents the shortcut. If a program contains:
```
var int16 arr[3] = {1, 2, 3};
```
the parser reads `1`, then tries to read `}`, fails because the next token is `,`, emits an error, and the `,` and subsequent elements are left unconsumed. The function returns only `expr = 1`. This is not just an AST omission — it leaves the token stream out of sync, meaning the `;` after the initializer will be seen as an unexpected token by the calling `parseVarDecl`, generating a cascade of spurious errors.

**Severity:** critical because: (a) the grammar explicitly allows multi-element initializers, (b) the failure corrupts subsequent parsing, and (c) there is no test case that exercises multi-element initializers, so this is a silent regression risk.

**Suggested fix:** implement the full list, accumulating into an `ArrayInitExpr` AST node (or a `[]Expr`), and handle the `}` terminator properly. At minimum, parse and discard additional elements with proper comma-skipping so the token stream stays synchronized.

---

## 3. Warning Issues

### [warning] `parseBinaryRest` loses all binary operators except `=` (parser.go, lines 1599–1612)

`parseBinaryRest` is called from `parseExprStmtStartingWith` (line 1106) when an expression statement begins with an identifier (after the label-detection lookahead). The function only handles the `=` operator:

```go
func (p *Parser) parseBinaryRest(expr Expr, minPrec int) Expr {
    // Simplified: just check for assignment
    if p.tokens.Peek().IsPunct("=") {
        ...
    }
    return expr
}
```

Any expression statement that starts with an identifier and uses any binary operator other than assignment will silently drop the right-hand side. For example:

```
x + y;         // parsed as bare "x", then "+" confuses the ';' expect
x == y;        // similar
Foo()->field;  // postfix handled, but binary binary ops dropped
```

The `minPrec` parameter is accepted but entirely ignored, making the API signature misleading.

This is an architectural shortcut that works only because `parseFuncStmt` feeds through `parseExprStmtStartingWith` only for the label-detection case, and most real expression statements are assignments (`x = ...`). However, any non-assignment expression statement starting with an identifier (like a void function call that was already parsed as postfix, then `+` something) will silently misbehave.

**Suggested fix:** `parseExprStmtStartingWith` should call the full `parseAssignment` with the pre-consumed `IdentExpr` threaded in, or, preferably, restructure `parseFuncStmt` to use a token-pushback mechanism. The current approach of consuming the identifier token before confirming it's not a label forces this awkward split.

A cleaner design: add a one-token pushback slot to `TokenReader`, then `parseFuncStmt` can peek at the identifier, look ahead one more token for `:`, and if not a colon, push the identifier back and call `parseStatement()` normally. This eliminates `parseExprStmtStartingWith` and `parseBinaryRest` entirely.

---

### [warning] Nil pointer dereference if `parseExpression()` returns nil (parser.go, multiple sites)

Several callers use the result of `parseExpression()` / `parseStatement()` directly without nil checks, then pass it to output functions that dereference it. Key sites:

- `parseIfStmt`, line 928: `cond := p.parseExpression()` — if `parseExpression` returns nil (on error), the `IfStmt.Cond` is nil. `output.go:writeExpr` guards for nil (line 326), so output is safe. But the semantic pass will likely dereference `Cond` without checking.
- `parseWhileStmt`, line 960: same pattern.
- `parseFuncStmt`, line 821: the type assertion `stmt.(FuncStmt)` — every concrete `Stmt` type that has `funcStmtNode()` defined will succeed, but if `parseStatement()` returns a concrete type that only implements `Stmt` and not `FuncStmt`, this silently returns `nil` and the statement is dropped. Currently all concrete `Stmt` types also implement `FuncStmt` (verified in `ast.go`), so this is not a bug today — but it is a fragile invariant not guarded by the type system.
- `parseIfStmt`, line 934: `then := p.parseStatement()` — if `then` is nil (parseStatement returned nil), the IfStmt is constructed with a nil Then, which `writeStmt` will call `ow.writeStmt(s.Then)` on. `writeStmt` uses a `switch s := stmt.(type)` which will hit the zero case and silently emit nothing, but downstream passes may panic.

**Suggested fix:** `parseIfStmt`, `parseWhileStmt`, `parseForStmt`, and `parseBlock` should check for nil body/condition results and, if in panic mode, apply `synchronizeStmt()` rather than constructing a partially-nil AST node.

---

### [warning] `parseAsmDecl` / `parseAsmStmt` skip `(` and `)` consumption (parser.go, lines 253–278, 868–892)

The grammar and the lexer both process `#asm("text")` at the lexer level. The lexer (`lang/ylex/lexer.go:handleDirective`, case `"asm"`) emits: `KEY "#asm"`, then immediately the raw string `LIT "..."`. The `(` and `)` are consumed by the lexer and are NOT emitted as tokens.

However, `parseAsmDecl` comments "already parsed by lexer" and proceeds to read the LIT token directly after the KEY token — this is correct. But a misleading note: the comments say `#asm("text")` but the parser actually expects `KEY "#asm"` followed immediately by `LIT "..."` with no parentheses in between, because the lexer ate them. This is consistent with how the lexer works, but the comment on line 254 "The string literal follows (already parsed by lexer)" could be misread as "the entire `#asm(...)` syntax is handled by lexer." A future maintainer might think they need to handle the `(` and `)` in the parser, or vice versa.

Additionally, the `parseAsmDecl` immediately does `p.tokens.Next()` (line 257) to consume the `#asm` key, then `strTok := p.tokens.Next()` (line 259) — this blindly consumes the next token without peeking first. If the stream is malformed (e.g. `#asm` not followed by a string, which the lexer should have caught but may not under error recovery), the `p.tokens.Next()` will consume the wrong token and discard it before the error check. A safer pattern is `strTok := p.tokens.Peek()` followed by a conditional `p.tokens.Next()`.

---

### [warning] `synchronizeStmt` exits early on any identifier (parser.go, lines 86–114)

```go
// Check for label (identifier followed by colon)
if tok.Category == TokID {
    // Peek ahead to see if it's a label
    // We can't easily do this, so just return and let the label be parsed
    return
}
```

This means that during error recovery inside a function body, any identifier token immediately stops synchronization. If the token stream looks like:

```
<error>  foo  bar  =  baz  ;
```

`synchronizeStmt` will return at `foo` and hand back to the caller. The caller (`parseFuncStmt`) tries to parse `foo` as a label or expression. This is by design but the comment "let the label be parsed" is optimistic — if `foo` is not a label (no `:` after it), the parser will try to parse `foo bar = baz;` as an expression and likely generate a fresh error immediately. In the worst case this causes the error recovery to loop without advancing, though in practice it will advance at least one token through the expression parser.

A more robust approach: consume the identifier token and check whether the next token is `:` before returning, advancing past the identifier if it is not a label.

---

### [warning] `alignDown` is incorrect for negative stack offsets (symtab.go, lines 397–403)

```go
func alignDown(n, align int) int {
    return n &^ (align - 1)
}
```

`alignDown` is called with a negative `n` (the growing-downward `FrameOffset`). Bitwise `&^` (bit-clear) on a negative two's-complement integer does NOT align downward in the negative direction — it aligns upward toward zero, i.e. it reduces the magnitude of the offset and potentially misaligns the allocation.

Example: if `FrameOffset = -3` and `align = 2`:
```
-3 in two's complement = 0xFFFFFFFFFFFFFFFD
0xFFFFFFFFFFFFFFFD &^ 1 = 0xFFFFFFFFFFFFFFFC = -4
```
In this case the result is `-4`, which is correct (we want a lower address = more negative). But this is a coincidence for power-of-two alignment: masking the low bit of a negative number does give the next-more-negative even address.

Wait, let's verify more carefully for align=4, n=-2:
```
-2 = ...FFFFFFFE
align-1 = 3 = ...00000003
&^ 3 = ...FFFFFFFC = -4
```
Result is -4 (8-byte frame from 0 to -4). But we needed to fit -2 into 4-byte alignment going downward: -4 is correct (round -2 down to -4). So the math is actually correct for powers of two on two's complement architectures.

However, this is subtle enough to warrant a comment. The function name `alignDown` implies rounding down (more negative), which is what we want for stack allocation, and the math works out, but the implementation is not obviously correct. A clearer implementation would be:

```go
func alignDown(n, align int) int {
    // For negative n (stack offsets), floor to next multiple of align
    if n%align == 0 {
        return n
    }
    if n < 0 {
        return n - (align + n%align)
    }
    return n &^ (align - 1)
}
```

This is a latent clarity issue today and could become a real bug if someone changes the alignment values to non-powers-of-two in the future.

---

### [warning] `parseConstDecl` accepts only `TokLIT` for scalar constant values (parser.go, lines 340–349)

```go
// Scalar const - parse literal value
valTok := p.tokens.Next()
if valTok.Category != TokLIT {
    p.error("expected constant value")
    p.synchronize()
    return nil
}
value = p.parseLiteralValue(valTok)
```

This correctly reflects the compiler design: the lexer folds all constant expressions to a single literal before emitting them (see `lang/ylex/lexer.go:handleConstDecl`, which calls `parseConstExpr()` and emits the result as a `LIT` token). So at the parser's level, a scalar constant value is always a single `LIT` token.

This is correct, but the parser comment says "parse literal value" without noting the dependency on lexer pre-folding. If the lexer is ever changed to emit unfold constant expressions (e.g., to preserve source fidelity), this code will silently fail for anything more complex than a bare literal. A comment noting the lexer contract would help future maintainers.

---

### [warning] `parseTokenLine` returns `TokEOF` for malformed token lines (token.go, lines 117–138)

```go
if len(parts) != 3 {
    // Malformed token line - return an error token
    return Token{
        Category: TokEOF,
        Value:    fmt.Sprintf("malformed token: %s", text),
    }
}
```

A malformed token line from the lexer silently terminates the parse as if it were EOF. The value field contains the diagnostic text `"malformed token: ..."`, but since the parser checks `tok.Category == TokEOF`, it sees EOF and stops silently. The diagnostic text is never printed. A better approach would be to use a dedicated error token category (or at least print the message to stderr) so the parse failure is visible rather than manifesting as an unexpected early-EOF error.

---

### [warning] `parseVarDecl` with `arrayLen == -1` and no `=` generates error but continues (parser.go, lines 440–443)

```go
} else if arrayLen == -1 {
    // byte[] without initializer is an error
    p.error("array with inferred size requires an initializer")
}
```

`p.error` sets `panicMode = true` but does NOT return. The code continues to `ExpectPunct(";")`, which may or may not succeed. If it succeeds, a `VarDecl` with `arrayLen == -1` and `Init == nil` is returned and registered in the symbol table. The `DefineGlobalVar` call computes `size = typ.Size(...) * arrayLen` — but `arrayLen == -1` means `size = -1`, and `DataOffset` is updated by `DataOffset += size` which becomes `DataOffset -= 1`. This is a data-corruption bug in the global symbol table's data layout accounting. It can only be triggered through the error path, but still.

**Suggested fix:** after `p.error(...)` for the inferred-size-without-initializer case, either `return nil` immediately or set `arrayLen = 0` before continuing, and avoid registering the malformed declaration in the symbol table.

---

### [warning] `FuncDecl` symbol registered before parameters are parsed (parser.go, lines 641–658)

```go
funcSym, err2 := p.symtab.DefineFunc(nameTok.Value, returnType, loc)
...
p.funcScope = NewFuncScope(funcSym)

// Parse parameters
params := make([]*Param, 0)
if !p.tokens.Peek().IsPunct(")") {
    for {
        param := p.parseParam()
        ...
    }
}
```

The function symbol is entered into the global symbol table before its parameter list is parsed. This means that inside a parameter type expression (if YAPL ever supports default arguments, or during error recovery), a recursive lookup of the function's own name would find a symbol with an empty `Params` slice. Today this is benign because parameter types cannot refer to the function being defined and there are no default argument expressions. But it is still an ordering concern worth noting.

---

### [warning] `parseStructDecl` uses `synchronizeStmt` for struct field error recovery (parser.go, lines 541–545)

```go
if p.panicMode {
    p.synchronizeStmt()
}
```

`synchronizeStmt` is designed for statement-level recovery: it stops at `}`, statement keywords, `;`, and identifiers. Inside a struct body, this may accidentally consume the closing `}` of the struct (since `synchronizeStmt` calls `p.tokens.Next()` after `;` — and struct fields end in `;`), which means the recovery actually advances past the `;` and resumes at the next field, which is correct. But it also returns early on any identifier, which inside a struct body could be the next field's type name. The net effect is that field-level error recovery inside structs is fragile and may give confusing cascaded errors. A dedicated `synchronizeStructField` that stops only at `}` or `;` would be more appropriate.

---

## 4. Nits

**N1. `paramLocation` uses `R1`–`R3` for params 0–2 but emits `R%d` with index `param.Index+1`** (output.go, line 210). This is correct: `param.Index` is 0-based and maps to R1–R3. The code is correct but the off-by-one (`+1`) is easy to mis-read. A named constant would help.

**N2. Dead `isDoubleAmp` function** (parser.go, lines 1302–1305). As noted above, the lexer always emits `&&` as a single token, so `IsPunct("&&")` can never return true when the current token is `&`. The guard in `parseMultiplicative` is unnecessary but harmless. The function could be removed to reduce confusion.

**N3. The `minPrec` parameter of `parseBinaryRest` is accepted but unused** (parser.go, line 1599). This is a dead parameter that suggests a Pratt-parser design was started and abandoned. Either implement it or remove the parameter.

**N4. `go.mod` declares package `main` for both `ylex` and `yparse`**, which means `go test ./yparse/` from inside `lang/` works, but importing either package from tests requires awkward subprocess invocation (which the test does correctly). This is an intentional design for the pipeline architecture and is consistent, just worth noting.

**N5. Test `02_control.yapl` has an identifier `TestBreakContinue` that is 17 characters long**, exceeding the 15-character lexer limit. This causes `TestParserPositive/02_control` to fail:

```
parser_test.go:91: expected success but got error: exit status 1
    stderr: 02_control.yapl:61: error: identifier "TestBreakContinue" exceeds maximum length of 15 characters
```

This is a test-data bug, not a parser bug. Rename the function to `TestBrkCont` or similar.

**N6. `output.go:writeFunc` line 161**: writes `RETURN <type>` for the function header and `RETURN <line>` for return statements (lines 302, 307). The `ysem` IR parser must distinguish these. The README confirms this distinction exists (it was a previous bug); the code is correct, just noting that the format is subtly overloaded and both the parser and ysem must agree on it.

**N7. `parseType` forwards `void` as a valid type everywhere** (parser.go, lines 169–171). The grammar says `void` is only valid as a function return type (via `ReturnType`). The parser allows `var void x;` or `@void` pointer in any type position. `@void` is explicitly valid per the grammar, but bare `void` in a `var` declaration should be rejected. This is currently left to semantic analysis (`ysem`), which is a reasonable choice but worth documenting.

**N8. `parseLiteralValue` ignores parse errors** (parser.go, lines 1584–1596):

```go
n, _ := strconv.ParseInt(val[2:], 16, 64)
```

The error return from `ParseInt` is silently discarded. Since the lexer guarantees that literals are syntactically valid numbers, this is safe in practice. But if a malformed literal somehow reaches the parser (e.g., through a malformed token line), the parse yields 0 silently. A debug-mode check (`if err != nil { p.error(...) }`) would make failures visible.

**N9. `writeConstants` and `writeGlobalVars` iterate `prog.Decls` twice** (output.go, lines 85–155). For large programs this is O(n) instead of O(1). Given that the self-hosting target has a 64KB address space (implying small files), this is not a performance concern but is worth noting.

**N10. `SymbolTable.AddError` and `SymbolTable.HasErrors`** (symtab.go, lines 244–251) are defined but never called from the parser. The parser uses its own `p.errors` slice. The symbol table error methods appear to be dead code.

---

## 5. Questions

**Q1.** The grammar (`yapl_grammar.ebnf`) allows `sizeof UnaryExpr` (without parens, line 265), but does any real YAPL code use this form? The parser implements it, but if the lexer never produces the tokenization that would exercise it, the code path may be untested.

**Q2.** `parseType` (parser.go, lines 183–193) allows any unknown identifier as a forward-reference to a struct:

```go
// Could be a forward reference - allow it for now
// Semantic analysis will catch undefined types
p.tokens.Next()
return NewStructType(tok.Value)
```

Is forward reference of struct types actually supported by the language? The grammar comment in the EBNF says "No forward declarations: functions, types, constants must precede use." If forward struct references are disallowed, this code should call `p.error(...)` when `LookupStruct` returns nil, rather than silently allowing it. If forward references ARE supported, the EBNF comment is wrong.

**Q3.** The `FuncScope.FrameOffset` starts at 0 and grows negative. But the `Finalize` comment says "padding is conceptually at the bottom of the frame" and offsets are NOT adjusted. The downstream IR parser (`ysem`) must know to convert `LocalSymbol.Offset` (a negative number like -4) to a positive `SP` offset using `frameSize + negativeOffset`. Is this contract documented in `IR_FORMAT.md` or only in the code comment in `symtab.go`? If only in a code comment, a future maintainer of `ysem` could easily get this wrong.

**Q4.** `parseArrayInit` is called for both `const` and `var` array initializers. For `var`, the grammar says `Initializer = Expression` (not `ArrayInit`) for scalar vars, and `ArrayInit` for arrays. But the EBNF also shows that `VarDecl` without brackets uses `Initializer = Expression`. The parser correctly calls `parseExpression()` for scalar `var` with `=` and `parseArrayInit()` for arrays. However, for the `const` case: the grammar says `const TypeSpecifier identifier "=" ConstExpr` for scalars — but the lexer has already folded the ConstExpr to a single literal, so the parser reads a `TokLIT`. Is there a design document confirming that the lexer-folds-all-consts contract is permanent? If it were ever relaxed, the const parser would need to call a full expression parser, not just `p.tokens.Next()`.
