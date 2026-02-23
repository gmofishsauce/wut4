// yld - WUT-4 linker
//
// Usage: yld [flags] file1.wo file2.wo ...
//
// Flags:
//   -o file    Write output to file (default: wut4.out)
//   -v         Verbose output

package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	output := flag.String("o", "wut4.out", "output file")
	verbose := flag.Bool("v", false, "verbose output")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] file.wo ...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "WUT-4 linker — combines .wo object files into an executable\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	ld := newLinker(*verbose)

	for _, path := range flag.Args() {
		if *verbose {
			fmt.Printf("Loading %s\n", path)
		}
		obj, err := readObjectFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "yld: %v\n", err)
			os.Exit(1)
		}
		ld.addObject(obj)
	}

	mergedCode, mergedData, err := ld.link()
	if err != nil {
		fmt.Fprintf(os.Stderr, "yld: %v\n", err)
		os.Exit(1)
	}

	if err := writeExecutable(*output, mergedCode, mergedData); err != nil {
		fmt.Fprintf(os.Stderr, "yld: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Link successful: %s\n", *output)
	fmt.Printf("Code: %d bytes, Data: %d bytes\n", len(mergedCode), len(mergedData))
}
