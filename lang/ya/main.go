// ya - YAPL compiler driver
//
// Usage: ya [flags] source.yapl
//        ya [flags] source.wo ...    (link mode)
//
// Flags:
//   -o file    Write output to file (default: wut4.out or <base>.wo with -c)
//   -S         Stop after generating assembly (don't assemble)
//   -c         Compile to relocatable object file (.wo)
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
//     ylex, yparse, ysem, ygen, yasm, yld

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
	outputFile  = flag.String("o", "", "output file name (default: wut4.out, or <base>.wo with -c)")
	asmOnly     = flag.Bool("S", false, "stop after generating assembly")
	compileOnly = flag.Bool("c", false, "compile to relocatable object file (.wo)")
	keepFiles   = flag.Bool("k", false, "keep intermediate files")
	verbose     = flag.Bool("v", false, "verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] source.yapl\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s [flags] file.wo ...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "YAPL compiler driver\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	/* Detect link mode: all arguments end in .wo */
	allWO := true
	for _, arg := range flag.Args() {
		if !strings.HasSuffix(arg, ".wo") {
			allWO = false
			break
		}
	}

	if allWO {
		/* Link mode: invoke yld directly */
		if err := link(flag.Args()); err != nil {
			fmt.Fprintf(os.Stderr, "ya: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "ya: compile mode requires exactly one source file\n")
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
	if !*asmOnly {
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

	// Prepend boot.asm for bootstrap programs so the stack page is mapped
	// before the crt0-equivalent startup code runs.
	if hasBootstrapPragma(sourceData) {
		bootAsmPath, err := findBootAsm()
		if err != nil {
			return err
		}
		bootData, err := os.ReadFile(bootAsmPath)
		if err != nil {
			return fmt.Errorf("reading boot.asm: %v", err)
		}
		asmOut = append(bootData, asmOut...)
	}

	if *keepFiles || *asmOnly {
		asmFile := filepath.Join(sourceDir, baseNoExt+".asm")
		if err := os.WriteFile(asmFile, asmOut, 0644); err != nil {
			return fmt.Errorf("writing assembly: %v", err)
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote %s\n", asmFile)
		}
	}

	// Stop here if -S
	if *asmOnly {
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

	// Determine output file name and assembler arguments
	outFile := *outputFile
	var asmArgs []string
	if *compileOnly {
		// Produce .wo object file
		if outFile == "" {
			outFile = filepath.Join(sourceDir, baseNoExt+".wo")
		}
		asmArgs = []string{"-c", "-o", outFile, tmpAsmName}
	} else if hasBootstrapPragma(sourceData) {
		// Bootstrap program: yasm produces binary directly (legacy path)
		if outFile == "" {
			outFile = "wut4.out"
		}
		asmArgs = []string{"-o", outFile, tmpAsmName}
	} else {
		// Normal program: assemble to temp .wo, then link with crt0
		tmpWO, err := os.CreateTemp("", "ya-*.wo")
		if err != nil {
			return fmt.Errorf("creating temp object file: %v", err)
		}
		tmpWOName := tmpWO.Name()
		tmpWO.Close()
		defer os.Remove(tmpWOName)

		woArgs := []string{"-c", "-o", tmpWOName, tmpAsmName}
		cmd := exec.Command(asmPath, woArgs...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("assembler failed: %s", stderr.String())
			}
			return fmt.Errorf("assembler failed: %v", err)
		}

		return link([]string{tmpWOName})
	}

	// Run assembler (compile-only or bootstrap path)
	cmd := exec.Command(asmPath, asmArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("assembler failed: %s", stderr.String())
		}
		return fmt.Errorf("assembler failed: %v", err)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Wrote %s\n", outFile)
	}

	return nil
}

// link prepends crt0.wo and invokes yld to link the given .wo files into an executable
func link(woFiles []string) error {
	crt0, err := findCrt0()
	if err != nil {
		return err
	}

	allFiles := append([]string{crt0}, woFiles...)
	return runLinker(allFiles)
}

// runLinker invokes yld with the exact file list provided
func runLinker(woFiles []string) error {
	yldPath, err := findBinary("yld", "yld")
	if err != nil {
		return err
	}

	outFile := *outputFile
	if outFile == "" {
		outFile = "wut4.out"
	}

	args := append(woFiles, "-o", outFile)
	if *verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command(yldPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("linker failed: %s", stderr.String())
		}
		return fmt.Errorf("linker failed: %v", err)
	}

	return nil
}

// hasBootstrapPragma returns true if the source contains #pragma bootstrap
func hasBootstrapPragma(src []byte) bool {
	return bytes.Contains(src, []byte("#pragma bootstrap"))
}

// findCrt0 locates the crt0.wo startup file.
// Looks at $YAPL/../lib/crt0.wo, then <bindir>/../lib/crt0.wo.
func findCrt0() (string, error) {
	// Try $YAPL/../lib/crt0.wo
	if yaplDir := os.Getenv("YAPL"); yaplDir != "" {
		p := filepath.Join(yaplDir, "..", "lib", "crt0.wo")
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}

	// Try <directory of ya binary>/../lib/crt0.wo
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "lib", "crt0.wo")
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}

	return "", fmt.Errorf("crt0.wo not found; set YAPL env var to repo root or install lib/crt0.wo alongside binaries")
}

// findBootAsm locates the boot.asm startup file for bootstrap programs.
// Looks at $YAPL/../lib/boot.asm, then <bindir>/../lib/boot.asm.
func findBootAsm() (string, error) {
	// Try $YAPL/../lib/boot.asm
	if yaplDir := os.Getenv("YAPL"); yaplDir != "" {
		p := filepath.Join(yaplDir, "..", "lib", "boot.asm")
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}

	// Try <directory of ya binary>/../lib/boot.asm
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "lib", "boot.asm")
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}

	return "", fmt.Errorf("boot.asm not found; set YAPL env var to repo root or install lib/boot.asm alongside binaries")
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
	// Forward stderr to terminal (for #pragma message etc.)
	// while also capturing it for error reporting
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

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
