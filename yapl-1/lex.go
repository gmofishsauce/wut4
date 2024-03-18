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

// Convert the single-character numeric token to a value
func convert(b Byte) Word {
	return Word(b - Byte('0'))
}

var lineCount Word

func GetLine() Word {
	return lineCount
}

// The YAPL-1 language was created specifically to trivialize the
// lexer, because lexing is tedious and well understood by most
// everyone who might ever read this.
func GetToken(inFD Word) Token {
	pos := StrtabAllocate()
	len := StrtabRemaining()
	var b Word
	var inComment = false

	for b := Getb(inFD); b != E_EOF && len > 0; b = Getb(inFD) {
		if b > 0xFF || b == 0 {
			return ERR_LEX_IO
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
		if isLowerLetter(b) {
			return TT_STR|Token(SymEnter(true, pos, 1))
		}
		if isUpperLetter(b) {
			return TT_STR|Token(SymEnter(true, pos, 1))
		}
		StrtabDiscard()
		return ERR_LEX_INVAL
	}
	if b == E_EOF {
		return TT_EOF
	}
	return ERR_INT_NOSTR
}
