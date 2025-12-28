package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	/* Disassembler mode */
	if os.Args[1] == "-d" {
		if len(os.Args) != 3 {
			printUsage()
			os.Exit(1)
		}
		err := disassemble(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	/* Assembler mode */
	inputFile := os.Args[1]
	outputFile := "out.bin"
	if len(os.Args) >= 3 {
		outputFile = os.Args[2]
	}

	err := assemble(inputFile, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  asm <input.asm> [output.bin]  - Assemble a file\n")
	fmt.Fprintf(os.Stderr, "  asm -d <binary-file>          - Disassemble a file\n")
}
