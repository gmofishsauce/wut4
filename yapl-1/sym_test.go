/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

import (
	"testing"
)

func assert(t *testing.T, ok bool, s string) {
	if !ok {
		t.Fatalf("%s failed", s)
	}
}

func TestSym01(t *testing.T) {
	j := StrtabAllocate()
	m := StrtabRemaining()
	StrtabDiscard()

	x := Byte('X')
	i := StrtabAllocate()
	n := StrtabRemaining()
	assert(t, j == i, "TestSym01 1")
	assert(t, m == n, "TestSym01 2")
	assert(t, n > 100 && n < 0xC000, "TestSym01 3")

	// Mock input
	strtab[i] = x

	index := SymEnter(i, 1)
	assert(t, index == symtabNext - 1, "TestSym01 4")
	assert(t, symtab[index].Len == 1, "TestSym01 5")
	assert(t, strtab[symtab[index].Val] == x, "TestSym01 6")
}

func TestSym02(t *testing.T) {
	i := StrtabAllocate()
	// Mock some input
	strtab[i] = Byte('v')
	strtab[i+1] = Byte('a')
	strtab[i+2] = Byte('r')
	s1 := SymEnter(i, Byte(3))
	assert(t, s1 == SymLookup(i, Byte(3)), "TestSym02 1")

	s2 := SymEnter(Word(42), Byte(0))

	j := StrtabAllocate()
	assert(t, j == i+3, "TestSym02 2")
	// Mock some more input
	strtab[j] = Byte('d')
	strtab[j+1] = Byte('e')
	strtab[j+2] = Byte('f')
	SymEnter(j, Byte(3))

	assert(t, symtab[s2].Val == symtab[NumLookup(Word(42))].Val, "TestSym02 3")

	s3 := SymEnter(Word(43), Byte(0))

	i = StrtabAllocate()
	strtab[i] = Byte('v')
	strtab[i+1] = Byte('a')
	strtab[i+2] = Byte('r')
	s4 := SymEnter(i, Byte(3))

	assert(t, s1==s4, "TestSym02 4")
	assert(t, 1+symtab[s2].Val == symtab[s3].Val, "TestSym02 5")
	assert(t, i == j+3, "TestSym02 6")
}
