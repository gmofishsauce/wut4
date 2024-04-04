/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var LexDebug bool = true

// There are four types of tokens: user defined symbols like
// variable names and constant strings, language defined symbols
// ("keys"), numeric values, and error tokens. The types are
// encoded in the high order 2 bits, leaving 14 bits to be used
// for symbol table index (TT_STR, TT_KEY, and TT_NUM) or actual
// value (TT_ERR). (We cannot in general store constants in the
// token directly because only 14 bits are available, so we must
// create symbol table entries for numerical constants. The value
// of the constant is stored in the symbol table; numeric
// constants do not exist in the strings table.)
const (
	TT_USR Token = 0x0000      // user symbols from the source
	TT_KEY Token = 0x4000      // language defined symbols TODO maybe TT_LANG?
	TT_NUM Token = 0x8000      // numeric valued symbols
	TT_ERR Token = Token(ErrBase) // error tokens
)

// The target machine (WUT-4) doesn't have a barrel shifter, so it's
// helpful to avoid multiple-bit shifts where possible. We don't want
// e.g. (t >> 14) if we can help it, because this will have to compile
// to swap bytes; swap nybbles in low byte; shift right; shift right.
func IsUserTok(t Token) Bool {
	return (t&TT_USR) == TT_USR
}

func IsKeyTok(t Token) Bool {
	return (t&TT_KEY) == TT_KEY
}

func IsNumTok(t Token) Bool {
	return (t&TT_NUM) == TT_NUM
}

func IsErrTok(t Token) Bool {
	return (t&TT_ERR) == TT_ERR
}

// local functions

func isHash(b Word) Bool {
	return b == Word('#')
}

func isWhite(b Word) Bool {
	return b == Word(' ') || b == Word('\t') || b == Word('\n')
}

func isDigit(b Word) Bool {
	return b >= '0' && b <= '9'
}

func isLowerLetter(b Word) Bool {
	return b >= 'a' && b <= 'z'
}

func isUpperLetter(b Word) Bool {
	return b >= 'A' && b <= 'Z'
}

func isPunctuation(b Word) Bool {
	return b == '=' || b == '+' || b == '{' || b == '}' || b == ';'
}

// Convert the single-character numeric token to a value
func convert(b Byte) Word {
	return Word(b - Byte('0'))
}

func TokenAsSymIndex(t Token) SymIndex {
	return SymIndex(Word(t) & Word(^ErrBase))
}

var lineCount Word = 1

var tTypes []string = []string {"TT_USR", "TT_KEY", "TT_NUM", "TT_ERR", }

func PrintTok(t Token) {
	if IsErrTok(t) {
		Printf("; token: error %x%n", Word(t))
	} else {
		n := TokenAsSymIndex(t)
		if symtab[n].Len == 0 {
			Printf("; token: number %x%n", symtab[n].Val)
		} else {
			Printf("; token: string %c%n", strtab[symtab[n].Val])
		}
	}
}

func LineNumber() Word {
	return lineCount
}

var pbt Token = 0

func PushbackToken(t Token) {
	if pbt != 0 || t == 0 {
		panic("PushbackToken")
	}
	Printf("; Pushback: ")
	PrintTok(t)
	pbt = t
}

// The YAPL-1 language was created specifically to trivialize the
// lexer, because lexing is tedious and well understood by most
// everyone who might ever read this.
func GetToken(inFD Word) Token {
	tk := internalGetToken(inFD)
	if LexDebug {
		PrintTok(tk)
	}
	return tk
}

func internalGetToken(inFD Word) Token {
	if pbt != 0 {
		result := pbt
		pbt = 0
		return result
	}

	pos := StrtabAllocate()
	len := StrtabRemaining()
	var inComment = false
	var b Word
	var n StrIndex

	for {
		b = Getb(inFD)
		if b > 0xFF || len == 0 {
			break
		}

		if b == Word('\n') {
			inComment = false
			lineCount++
			continue
		}
		if inComment {
			continue
		}
		if isHash(b) {
			inComment = true
			continue
		}
		if isWhite(b) {
			continue
		}

		if isDigit(b) {
			val := convert(Byte(b))
			// Yeah, the cast from StrIndex to Word can't
			// change any bits. Compromises must be made.
			return TT_NUM|Token(SymEnter(Word(val), 0))
		}

		strtab[pos+n] = Byte(b)
		n++
		len--

		if isLowerLetter(b) {
			n := SymEnter(Word(pos), 1)
			return TT_USR|Token(n)
		}

		// All the language-defined symbols are entered into
		// the symbol table before lexing. This will work for
		// now, but for the YAPL-2 need to identify keywords
		// by their symbol table index < var FirstUserSymbol.
		if isUpperLetter(b) || isPunctuation(b) {
			return TT_KEY|Token(SymEnter(Word(pos), 1))
		}

		StrtabDiscard()
		return ErrorAsToken(ERR_LEX_INVAL)
	}
	if b == E_EOF {
		return TT_EOF
	} else if b > 0xFF {
		return ErrorAsToken(ERR_LEX_IO)
	}

	// Must be out of space
	return ErrorAsToken(ERR_INT_NOSTR)
}
