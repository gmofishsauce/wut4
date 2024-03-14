/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

func isWhite(b Word) bool {
	return b == Word(' ') || b == Word('\t') || b == Word('\n')
}

func isDigit(b Word) bool {
	return b >= '0' && b <= '9'
}

var tokenID []Byte
var tokenLit Word

func GetToken(stdin Word) Token {
	for b := getb(stdin); b != E_EOF; b = getb(stdin) {
		if isWhite(b) {
			continue
		}
		if isDigit(b) {
			tokenLit = b - Word('0')
			return TT_EOF // XXX
		}
	}
	return TT_EOF
}
