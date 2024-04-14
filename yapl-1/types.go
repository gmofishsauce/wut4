/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

// This is a language compiler designed to execute in a 64k address space.
//
// This compiler differs from modern compilers in the following way: it is 
// built around a _symbol table_, which contains a unique entry for each
// string symbol or constant value in the program being compiled. Modern
// compilers generally load the entire source file as a string and hold 
// references into the string; in a 64k memory space, this not possible.
//
// The first pass of this compiler makes use of three data structures:
//
// 1) A string table, containing all unique strings stored end to end.
// 2) A symbol table, where each entry may contain either an index into
//    the string table or a literal 16-bit value "lex'd" from the code.
// 3) An abstract syntax tree, where each entry contains a symbol table
//    index and some other information.
// 4) In addition, lexical tokens often contain symbol table indexes.
//
// The compiler makes heavy use of type punning to represent errors.
//
// All indices (into the string table, the symbol table, or the list of
// AST nodes) may be in the range 0 .. 0xBFFF ("48k") only. This is not
// a serious limit given the _entire memory space_ is only 64k, and these
// are indices into arrays, not pointers. Lexical tokens may also encode
// a symbol table index.
//
// All error codes are in the range 0xC000 - 0xFFFF. The end effect is
// that the same set of error codes can be returned by any function that
// normally returns a token, a string table index, a symbol table index,
// or an AST node index. There is one set of error codes and messages
// distinguished values in the low order 12 bits when the high 2 bits
// are 0b11.

// Basic types for self-hosting on the WUT-4. The most basic type is Word.
// A Word can hold a Byte, an Addr, or a Bool. Words, Addrs, and Bytes are
// truthy: nonzero values are Bool "true" and 0 values are Bool "false".
// Arithmetic on Words is unsigned and carries are lost. Bytes are silently
// extended to Words by 0-extension for arithmetic with Words. Arithmetic
// on Bools is not permitted. For now, arithmetic operations on Addrs behave
// as Words; restrictions will be added to address arithmetic as the YAPL
// language evolves.

type Word uint16
type Byte uint8
type Addr Word
type Bool bool

// These are the only means of input and output, because they are the only
// means implemented in the WUT-4 emulator for eventual self hosting. All
// output from the compiler becomes part of the assembly language result,
// so messages are prefixed with ";" making them assembler comments.
const (
	STDIN Word = Word(0)
	STDOUT Word = Word(1)
)

// Begin types specific to the compiler

type Token Word

// There are some maximum size constants below. The ultimate max of these
// max sizes is 0xC000 because error codes are the values 0xC000 - 0xFFFF.
// In fact, likely none of these can be greater than 0x8000, if even that, 
// because the tables in this file need to be stored simultaneously in a
// 64k address space during the first pass. The symbol length max, now 15,
// cannot ultimately exceed 255.

// Symbol table entry. Values may be string table offsets in which case
// the length is relevant, or they maybe constant values where the length
// is 0.
type Syment struct {
	Val Word          // index in string table or lit value
	Len Byte          // Length of name or 0 for numeric constant
	Info Byte         // Type information
}

type SymIndex Word    // symbol table index
const SYMTAB_MAX SymIndex = 4096
const SYMLEN_MAX Word = 16

type StrIndex Word    // index in strings table
const STRTAB_MAX StrIndex = 8192

// AST nodes use RJ's data oriented tree design. The code knows which
// node types have children. If a node has children, its first child
// node is immediately to its right. Its size is 1 + the size of all
// its children, so the next non-child node is at its index + size.
// It's tempting to try for having the size be a Byte, but I'm afraid
// a Block may have more than 255 Statement-like children. So it goes.

type AstNode struct { // AST node
	Sym SymIndex      // index of symbol table entry
	Size Word         // size of this node (with all subnodes)
	Kind Byte
	Xtra Byte
}

type AstIndex Word
const AstMaxNode AstIndex = 2048

const (
	AstKindNone = iota  // reserved
	AstKindError        // Error placeholder node
	AstKindUsr          // user defined symbol
)

const (
	AstXtraDecl Byte = 1 // bitflag 1 this is a declaration
)

func AsWord(q any) Word {
	var w Word
	switch q.(type) {
	case AstIndex:
		w = Word(q.(AstIndex))
	case StrIndex:
		w = Word(q.(StrIndex))
	case SymIndex:
		w = Word(q.(SymIndex))
	case Token:
		w = Word(q.(Token))
	case Word:
		w = q.(Word)
	default:
		Assert(false, "AsWord(): unknown type")
	}
	return w
}

