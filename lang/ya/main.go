// ya - YAPL compiler driver
//
// Usage: ya [flags] source.yapl
//
// Flags:
//   -o file    Write output to file (default: wut4.out)
//   -S         Stop after generating assembly (don't assemble)
//   -c         Compile only (generate .asm, don't assemble)
//   -k         Keep intermediate files (.lexout, .parseout, .ir, .asm)
//   -v         Verbose output
//
// The compiler pipeline:
//   source.yapl → ylex → yparse → ysem → ygen → yasm → binary
//
// Binary location:
//   If YAPL environment variable is set, binaries are found at:
//     $YAPL/ylex/ylex, $YAPL/yparse/yparse, $YAPL/ysem/ysem, $YAPL/ygen/ygen
//   Otherwise, binaries are found via PATH:
//     ylex, yparse, ysem, ygen, yasm

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	outputFile  = flag.String("o", "wut4.out", "output file name")
	asmOnly     = flag.Bool("S", false, "stop after generating assembly")
	compileOnly = flag.Bool("c", false, "compile only (same as -S)")
	keepFiles   = flag.Bool("k", false, "keep intermediate files")
	verbose     = flag.Bool("v", false, "verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] source.yapl\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "YAPL compiler driver\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	sourceFile := flag.Arg(0)
	if err := compile(sourceFile); err != nil {
		fmt.Fprintf(os.Stderr, "ya: %v\n", err)
		os.Exit(1)
	}
}

func compile(sourceFile string) error {
	// Verify source file exists
	if _, err := os.Stat(sourceFile); err != nil {
		return fmt.Errorf("cannot access %s: %v", sourceFile, err)
	}

	// Determine base name for intermediate files
	sourceDir := filepath.Dir(sourceFile)
	baseName := filepath.Base(sourceFile)
	ext := filepath.Ext(baseName)
	baseNoExt := strings.TrimSuffix(baseName, ext)

	// Find compiler components
	ylexPath, err := findBinary("ylex", "ylex")
	if err != nil {
		return err
	}
	parsePath, err := findBinary("yparse", "yparse")
	if err != nil {
		return err
	}
	semPath, err := findBinary("ysem", "ysem")
	if err != nil {
		return err
	}
	genPath, err := findBinary("ygen", "ygen")
	if err != nil {
		return err
	}

	// Only need assembler if we're going to use it
	var asmPath string
	if !*asmOnly && !*compileOnly {
		asmPath, err = findBinary("yasm", "yasm")
		if err != nil {
			return err
		}
	}

	// Read source file
	sourceData, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("reading source: %v", err)
	}

	// Stage 1: Lexer
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running lexer...\n")
	}
	lexOut, err := runStage(ylexPath, []string{sourceFile}, bytes.NewReader(sourceData))
	if err != nil {
		return fmt.Errorf("lexer failed: %v", err)
	}
	if *keepFiles {
		writeIntermediate(sourceDir, baseNoExt+".lexout", lexOut)
	}

	// Stage 2: Parser
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running parser...\n")
	}
	parseOut, err := runStage(parsePath, nil, bytes.NewReader(lexOut))
	if err != nil {
		return fmt.Errorf("parser failed: %v", err)
	}
	if *keepFiles {
		writeIntermediate(sourceDir, baseNoExt+".parseout", parseOut)
	}

	// Stage 3: Semantic analyzer
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running semantic analyzer...\n")
	}
	irOut, err := runStage(semPath, nil, bytes.NewReader(parseOut))
	if err != nil {
		return fmt.Errorf("semantic analyzer failed: %v", err)
	}
	if *keepFiles {
		writeIntermediate(sourceDir, baseNoExt+".ir", irOut)
	}

	// Stage 4: Code generator
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running code generator...\n")
	}
	asmOut, err := runStage(genPath, nil, bytes.NewReader(irOut))
	if err != nil {
		return fmt.Errorf("code generator failed: %v", err)
	}
	if *keepFiles || *asmOnly || *compileOnly {
		asmFile := filepath.Join(sourceDir, baseNoExt+".asm")
		if err := os.WriteFile(asmFile, asmOut, 0644); err != nil {
			return fmt.Errorf("writing assembly: %v", err)
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote %s\n", asmFile)
		}
	}

	// Stop here if -S or -c
	if *asmOnly || *compileOnly {
		return nil
	}

	// Stage 5: Assembler
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running assembler...\n")
	}

	// Write assembly to temp file for assembler
	tmpAsm, err := os.CreateTemp("", "ya-*.asm")
	if err != nil {
		return fmt.Errorf("creating temp file: %v", err)
	}
	tmpAsmName := tmpAsm.Name()
	defer os.Remove(tmpAsmName)

	if _, err := tmpAsm.Write(asmOut); err != nil {
		tmpAsm.Close()
		return fmt.Errorf("writing temp assembly: %v", err)
	}
	tmpAsm.Close()

	// Run assembler
	cmd := exec.Command(asmPath, tmpAsmName, "-o", *outputFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("assembler failed: %s", stderr.String())
		}
		return fmt.Errorf("assembler failed: %v", err)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Wrote %s\n", *outputFile)
	}

	return nil
}

// findBinary locates a compiler component binary.
// If YAPL env var is set, looks in $YAPL/<subdir>/<name>.
// Otherwise, looks in PATH for <name>.
func findBinary(subdir, name string) (string, error) {
	if yaplDir := os.Getenv("YAPL"); yaplDir != "" {
		path := filepath.Join(yaplDir, subdir, name)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("compiler component %s not found at %s", name, path)
		}
		return path, nil
	}

	// Look in PATH
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("compiler component %s not found in PATH (set YAPL env var to specify location)", name)
	}
	return path, nil
}

// runStage executes a compiler stage, returning its stdout
func runStage(path string, args []string, stdin io.Reader) ([]byte, error) {
	cmd := exec.Command(path, args...)
	cmd.Stdin = stdin

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}

// writeIntermediate writes an intermediate file
func writeIntermediate(dir, name string, data []byte) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ya: warning: could not write %s: %v\n", path, err)
		return
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "Wrote %s\n", path)
	}
}
