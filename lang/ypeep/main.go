package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var rawLines []string
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ypeep: read error: %v\n", err)
		os.Exit(1)
	}

	lines := parseAll(rawLines)
	optimize(lines)

	out := bufio.NewWriter(os.Stdout)
	writeAll(out, lines)
	out.Flush()
}
