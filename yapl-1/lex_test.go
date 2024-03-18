/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

import (
	//"os"
	"strings"
	"testing"
)

var tTypes []string = []string {"TT_STR", "TT_NUM", "TT_KEY", "TT_ERR", }

func mkGoString(t *testing.T, strIndex Word, len Word) string {
	var sb strings.Builder
	for i := Word(0); i < len; i++ {
		r := rune(strtab[strIndex+i])
		if r == ' ' {
			r = '_'
		}
		t.Logf("WRITE %d %d\n", strIndex+i, r)
		sb.WriteRune(rune(strtab[strIndex+i]))
	}
	return sb.String()
}

func TestLex01(t *testing.T) {
	Init()
	/*
	f, err := os.Open("t1.yapl-1")
	check(t, "opening test file t1.yapl-1", err)
	fd := Word(f.Fd())
	*/
	fd := Word(0)
	var tk Token
	for tk = GetToken(fd); tk != TT_EOF; tk = GetToken(fd) {
		tType := tk>>14
		tIndex := tk&0x3FFF
		t.Logf("line 0x%X: %s 0x%X ", GetLine(), tTypes[tType], tIndex)
		if tk >= TT_ERR {
			t.Logf("(error 0x%03X)\n", tk&0xFFF)
			break
		} else {
			s := mkGoString(t, symtab[tIndex].Val, Word(symtab[tIndex].Len))
			if symtab[tIndex].Len == 0 {
				t.Logf("(0-length symtab entry)\n")
			} else {
				t.Logf("(%s)\n", s)
			}
		}
	}
	assert(t, tk == TT_EOF, "lexer")
}
