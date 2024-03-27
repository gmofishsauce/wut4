/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

var LexDebug bool = true

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

var lineCount Word = 1

var tTypes []string = []string {"TT_USR", "TT_KEY", "TT_NUM", "TT_ERR", }

func PrintTok(t Token) {
	if IsErrTok(t) {
		Printf("; token: error %x%n", Word(t))
	} else {
		n := Word(t&0x3FFF)
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
	var n Word

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
			return TT_NUM|Token(SymEnter(val, 0))
		}

		strtab[pos+n] = Byte(b)
		n++
		len--

		if isLowerLetter(b) {
			n := SymEnter(pos, 1)
			return TT_USR|Token(n)
		}

		// All the language-defined symbols are entered into
		// the symbol table before lexing. This will work for
		// now, but for the YAPL-2 need to identify keywords
		// by their symbol table index < var FirstUserSymbol.
		if isUpperLetter(b) || isPunctuation(b) {
			return TT_KEY|Token(SymEnter(pos, 1))
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
