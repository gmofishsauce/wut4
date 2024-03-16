/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

import (
	"testing"
)

func equal(t *testing.T, ok bool, s string) {
	if !ok {
		t.Fatalf("%s failed", s)
	}
}

func TestSym01(t *testing.T) {
	i, n := StrtabAllocate()
	StrtabDiscard()
	j, m := StrtabAllocate()
	StrtabDiscard()
	equal(t, i == j, "TestSym01 1")
	equal(t, m == n, "TestSym01 1A")

	x := Byte('X')
	i, n = StrtabAllocate()
	strtab[i] = x
	StrtabCommit(1)
	j, m = StrtabAllocate()
	equal(t, j == i+1, "TestSym01 2")
	equal(t, x == strtab[i], "TestSym01 3")
	equal(t, i+1 == j, "TestSym01 4")
}

func TestSym02(t *testing.T) {
	i, _ := StrtabAllocate()
	strtab[i] = Byte('v')
	strtab[i+1] = Byte('a')
	strtab[i+2] = Byte('r')
	s1 := SymEnter(false, i, Byte(3))
	StrtabCommit(3)
	equal(t, s1 == SymLookup(i, Byte(3)), "TestSym02 1")

	s2 := SymEnter(false, Word(42), Byte(0))

	i, _ = StrtabAllocate()
	strtab[i] = Byte('d')
	strtab[i+1] = Byte('e')
	strtab[i+2] = Byte('f')
	_ = SymEnter(false, i, Byte(3))
	StrtabCommit(3)

	equal(t, symtab[s2].Val == symtab[NumLookup(Word(42))].Val, "TestSym02 3")

	s3 := SymEnter(false, Word(43), Byte(0))

	i, _ = StrtabAllocate()
	strtab[i] = Byte('v')
	strtab[i+1] = Byte('a')
	strtab[i+2] = Byte('r')
	s4 := SymEnter(true, i, Byte(3))

	equal(t, s1==s4, "TestSym02 4")
	equal(t, 1+symtab[s2].Val == symtab[s3].Val, "TestSym02 5")
}
