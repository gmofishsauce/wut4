/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

import (
	"os"
	"testing"
)

func TestLex01(t *testing.T) {
	f, err := os.Open("t1.yapl-1")
	check(t, "opening test file t1.yapl-1", err)
	fd := Word(f.Fd())
	Init(fd)

	t1 := GetToken()
	PushbackToken(t1)
	t2 := GetToken()
	assert(t, t1 == t2, "PushbackToken")

	var tk Token
	for tk = GetToken(); tk != TT_EOF; tk = GetToken() {
		if tk >= TT_ERR {
			break
		}
	}
	assert(t, tk == TT_EOF, "lexer")
}
