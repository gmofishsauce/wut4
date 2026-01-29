// YAPL Code Generator - Pass 4 of the YAPL compiler
// Reads IR from stdin (sem output), writes WUT-4 assembly to stdout

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	// Read IR from stdin
	parser := NewIRParser(bufio.NewReader(os.Stdin))
	prog, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing IR: %v\n", err)
		os.Exit(1)
	}

	// Create emitter for output
	emitter := NewEmitter(bufio.NewWriter(os.Stdout))

	// Generate code
	codegen := NewCodeGen(prog, emitter)
	codegen.Generate()

	// Flush output
	emitter.Flush()
}
