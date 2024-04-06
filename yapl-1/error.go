/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// All error codes are in the range 0xC000 - 0xFFFF. The end effect is
// that the same set of error codes can be returned by any function that
// normally returns a token, a string table index, a symbol table index,
// or an AST node index. There is one set of error codes and messages
// distinguished values in the low order 12 bits when the high 2 bits
// are 0b11.

const ErrBase Word = 0xC000

// Error types defined in the compiler are encoded in the low 12 bits.
// A few error codes are in the range 0xFFFn; these are compatible with
// the error encoding convention and can be returned directly as errors.
const ( // Error subtypes
	ERR_LEX Word = 0x100     // 0x100 .. 0x1FF lexer errors
	ERR_PARSE Word = 0x200   // 0x200 .. 0x2FF syntax errors
	ERR_TYPE Word = 0x300    // 0x300 .. 0x3FF type errors
	ERR_IR Word = 0x400      // 0x400 .. 0x4FF IR errors
	ERR_GEN Word = 0x500     // 0x500 .. 0x5FF code gen errors
	ERR_SYM Word = 0x600     // 0x600 .. 0x6FF symbol table errors
	ERR_INT Word = 0x700     // 0x700 .. 0x6FF internal errors
	ERR_SYS Word = 0xFF00    // 0xFF00..0xFFFF system (e.g. I/O) errors
)

// This takes advantage of casting an "external" (WUT-4 "errno") error
// as a compiler type. This casting is always allowed, because the
// entire SYS ("errno") error space is reserved in the space of compiler
// error codes.
const TT_EOF Token = Token(E_EOF) // 0xFFFF io.go

// It would make complete sense to have a structure type holding an error
// code and a constant message string and a two-part lookup scheme that
// would yield the messages for an error code in constant time. But in
// keeping with the strategy here, I expect to have support for
// 1-dimensional arrays a long time before I have support for structures.
// And my plan for YAPL-1 is: do everything by linear search; improve the
// data structures lookup algorithms later. To add an error code, add it
// to the constant definitions; then add it to the array of error codes;
// then add a short string message to the messages in the correct position.
//
// This is programming for big kids. Don't screw up.

const ( // Lexer errors
	ERR_LEX_INVAL Word = ErrBase|ERR_LEX|1   // 0xC101 invalid character
	ERR_LEX_IO Word = ErrBase|ERR_LEX|2      // 0xC102 i/o error on input
	ERR_LEX_UNEXP Word = ErrBase|ERR_LEX|3   // 0xC103 unexpected char
)

const ( // Parse errors
	ERR_PARSE_ERR Word = ErrBase|ERR_PARSE|1 // 0xC201 "parse error" (TBD)
)

const ( // Symbol table errors
	ERR_SYM_REDEF Word = ErrBase|ERR_SYM|1   // 0xC601 symbol redefined
	ERR_SYM_NODEF Word = ErrBase|ERR_SYM|2   // 0xC602 symbol undefined
)

const ( // internal errors, e.g. out of space
	ERR_INT_NOSTR Word = ErrBase|ERR_INT|1   // 0xC701 string table full
	ERR_INT_NOSYM Word = ErrBase|ERR_INT|2   // 0xC702 symbol table full
	ERR_INT_TOOBIG Word = ErrBase|ERR_INT|3  // 0xC703 symbol or string too long
	ERR_INT_BUG Word = ErrBase|ERR_INT|4     // 0xC704 unspecified internal error
	ERR_INT_INIT Word = ErrBase|ERR_INT|5    // 0xC705 initialization error
	ERR_INT_CAST Word = ErrBase|ERR_INT|6    // 0xC706 bad cast
)

var errorTable []Word = []Word {
	ERR_LEX_INVAL,
	ERR_LEX_IO,
	ERR_LEX_UNEXP,
	ERR_PARSE_ERR,
	ERR_SYM_REDEF,
	ERR_SYM_NODEF,
	ERR_INT_NOSTR,
	ERR_INT_NOSYM,
	ERR_INT_TOOBIG,
	ERR_INT_BUG,
	ERR_INT_INIT,
	ERR_INT_CAST,
}

var errorMessages []string = []string {
	"invalid character",
	"i/o error on input",
	"unexpected char",
	"syntax error",
	"symbol redefined",
	"symbol undefined",
	"string table full",
	"symbol table full",
	"symbol or string too long",
	"unspecified internal error",
	"initialization error",
	"bad cast",
}

func LookupError(code Word) string {
	for i, val := range errorTable {
		if val == code {
			return errorMessages[i]
		}
	}
	return "internal error: unknown error code"
}

// Error severities
const ERR_CONTINUE = Word(1)
const ERR_FATAL    = Word(2)

// This is the while point of all the fussing
func PrintErr(fmt string, code Word, sev Word, val Word) {
	Printf("; Error: line %x: %s: ", LineNumber(), LookupError(code))
	Printf(fmt, val)
	Printf("%n")
	if sev == ERR_FATAL {
		Exit(2)
	}
}

func IsError(w Word) Bool {
	return w >= ErrBase
}

func ErrorAsToken(e Word) Token {
	if e < ErrBase {
		e = ERR_INT_BUG
	}
	return Token(e)
}

func ErrorAsSymIndex(e Word) SymIndex {
	if e < ErrBase {
		e = ERR_INT_BUG
	}
	return SymIndex(e)
}

func ErrorAsStrIndex(e Word) StrIndex {
	if e < ErrBase {
		e = ERR_INT_BUG
	}
	return StrIndex(e)
}

func ErrorAsAstIndex(e Word) AstNodeIndex {
	if e < ErrBase {
		e = ERR_INT_BUG
	}
	return AstNodeIndex(e)
}
