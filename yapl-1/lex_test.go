/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

import (
	"os"
	"testing"
)

var tTypes []string = []string {"TT_USR", "TT_KEY", "TT_NUM", "TT_ERR", }

func TestLex01(t *testing.T) {
	Init()
	f, err := os.Open("t1.yapl-1")
	check(t, "opening test file t1.yapl-1", err)
	fd := Word(f.Fd())

	t1 := GetToken(fd)
	PushbackToken(t1)
	t2 := GetToken(fd)
	assert(t, t1==t2, "PushbackToken")

	var tk Token
	for tk = GetToken(fd); tk != TT_EOF; tk = GetToken(fd) {
		if tk >= TT_ERR {
			break
		}
	}
	assert(t, tk == TT_EOF, "lexer")
}
