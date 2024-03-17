/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

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

// The YAPL-1 language was created specifically to trivialize the
// lexer, because lexing is tedious and well understood by most
// everyone who might ever read this. Normally, a lexer has a
// "current state" and a next character, and the new state and
// possible result token are a function of both. In YAPL-1, every
// token is exactly one character long. So at entry to the following
// function we are necessarily "between" tokens. We skip white space
// and then one of three things happen: we have token, we have an
// unexpected character (error), or we have EOF.
func GetToken(stdin Word) Token {
	pos := StrtabAllocate()
	len := StrtabRemaining()
	var b Word

	for b := getb(stdin); b != E_EOF && len > 0; b = getb(stdin) {
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
		StrtabDiscard()
		return ERR_LEX_INVAL
	}
	if b == E_EOF {
		return TT_EOF
	}
	return ERR_INT_NOSTR
}
