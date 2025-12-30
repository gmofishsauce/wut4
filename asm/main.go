package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	disasm := flag.Bool("d", false, "disassemble mode")
	output := flag.String("o", "wut4.out", "output file")
	flag.Parse()

	if *disasm {
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Error: disassemble mode requires input file\n")
			os.Exit(1)
		}
		inputFile := flag.Arg(0)
		err := disassemble(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Error: assemble mode requires input file\n")
			os.Exit(1)
		}
		inputFile := flag.Arg(0)
		err := assemble(inputFile, *output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
