// YAPL Parser - Pass 2 of the YAPL compiler
// Reads token stream from stdin (Pass 1 output), writes AST to stdout

package main

import (
	"fmt"
	"os"
)

func main() {
	// Create token reader from stdin
	tokens := NewTokenReader(os.Stdin)

	// Create parser
	parser := NewParser(tokens)

	// Parse the program
	prog, symtab, errors := parser.Parse()

	// Report errors
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Fprintf(os.Stderr, "%d error(s)\n", len(errors))
		os.Exit(1)
	}

	// Get filename from token reader (set by #file directive)
	filename := tokens.CurrentFile()

	// Write output
	output := NewOutputWriter(os.Stdout)
	output.WriteProgram(prog, symtab, filename)
	output.Flush()
}
