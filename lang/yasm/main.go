package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	disasm := flag.Bool("d", false, "disassemble mode")
	objMode := flag.Bool("c", false, "produce relocatable object file (.wo)")
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
		var inputName string
		var input string

		if flag.NArg() < 1 {
			/* No file argument - read from stdin (pipe) */
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
			inputName = "<stdin>"
			input = string(data)
		} else {
			inputName = flag.Arg(0)
			data, err := os.ReadFile(inputName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			input = string(data)
		}

		err := assembleMode(inputName, input, *output, *objMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
