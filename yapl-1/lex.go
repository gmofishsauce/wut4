/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

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

func LineNumber() Word {
	return lineCount
}

var pbt Token = 0

func PushbackToken(t Token) {
	if pbt != 0 || t == 0 {
		panic("PushbackToken")
	}
	pbt = t
}

// The YAPL-1 language was created specifically to trivialize the
// lexer, because lexing is tedious and well understood by most
// everyone who might ever read this.
func GetToken(inFD Word) Token {
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
			return TT_NUM|Token(SymEnter(true, val, 0))
		}

		strtab[pos+n] = Byte(b)
		n++
		len--

		if isLowerLetter(b) {
			return TT_USR|Token(SymEnter(true, pos, 1))
		}

		// All the language-defined symbols are entered into
		// the symbol table before lexing. This will work for
		// now, but for the YAPL-2 need to identify keywords
		// by their symbol table index < var FirstUserSymbol.
		if isUpperLetter(b) || isPunctuation(b) {
			return TT_KEY|Token(SymEnter(true, pos, 1))
		}

		StrtabDiscard()
		return ERR_LEX_INVAL
	}
	if b == E_EOF {
		return TT_EOF
	} else if b > 0xFF {
		return ERR_LEX_IO
	}

	// Must be out of space
	return ERR_INT_NOSTR
}
