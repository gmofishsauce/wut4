/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

func isWhite(b Word) bool {
	return b == Word(' ') || b == Word('\t') || b == Word('\n')
}

func isDigit(b Word) bool {
	return b >= '0' && b <= '9'
}

func GetToken(stdin Word) Token {
	return TT_INVALID
/*
	for b := getb(stdin); b != E_EOF; b = getb(stdin) {
		if isWhite(b) {
			continue
		}
		if isDigit(b) {
			result.T = 
			return SymEnter(
		}
	}
*/
}
