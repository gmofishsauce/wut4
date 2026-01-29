// YAPL Semantic Analyzer - Pass 3 of the YAPL compiler
// Reads AST from stdin (parser output), writes IR to stdout

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	// Read AST from stdin
	reader := NewASTReader(bufio.NewReader(os.Stdin))
	program, err := reader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading AST: %v\n", err)
		os.Exit(1)
	}

	// Create semantic analyzer
	analyzer := NewAnalyzer(program)

	// Perform semantic analysis and generate IR
	ir, errs := analyzer.Analyze()

	// Report errors
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		fmt.Fprintf(os.Stderr, "%d error(s)\n", len(errs))
		os.Exit(1)
	}

	// Write IR to stdout
	writer := bufio.NewWriter(os.Stdout)
	ir.Write(writer)
	writer.Flush()
}
