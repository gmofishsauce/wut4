// ya - YAPL compiler driver
//
// Usage: ya [flags] file ...
//
// Input files:
//   .yapl files are compiled; .wo files are passed to the linker.
//   -c, -S, and -k are incompatible with .wo input files.
//   -o and -c are incompatible with each other.
//   #pragma bootstrap may only appear in the first source file.
//
// Flags:
//   -o file    Write output to file (incompatible with -c)
//   -S         Stop after generating assembly (incompatible with .wo inputs)
//   -c         Compile to relocatable object file (.wo) (incompatible with -o, .wo inputs)
//   -k         Keep intermediate files (incompatible with .wo inputs)
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
	outputFile  = flag.String("o", "", "output file name (incompatible with -c)")
	asmOnly     = flag.Bool("S", false, "stop after generating assembly")
	compileOnly = flag.Bool("c", false, "compile to relocatable object file (.wo)")
	keepFiles   = flag.Bool("k", false, "keep intermediate files")
	verbose     = flag.Bool("v", false, "verbose output")
	doOptimize  = flag.Bool("O", false, "run peephole optimizer (ypeep) on generated assembly")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] file ...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "YAPL compiler driver\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Validate incompatible flag combinations
	if *compileOnly && *outputFile != "" {
		fmt.Fprintf(os.Stderr, "ya: -o and -c are incompatible\n")
		os.Exit(1)
	}

	// Split args into YAPL source files and .wo object files
	var yaplFiles, inputWOs []string
	for _, arg := range flag.Args() {
		if strings.HasSuffix(arg, ".wo") {
			inputWOs = append(inputWOs, arg)
		} else {
			yaplFiles = append(yaplFiles, arg)
		}
	}

	// Validate .wo inputs with incompatible flags
	if len(inputWOs) > 0 && (*compileOnly || *asmOnly || *keepFiles) {
		fmt.Fprintf(os.Stderr, "ya: .wo object files cannot be used with -c, -S, or -k\n")
		os.Exit(1)
	}

	// Validate bootstrap pragma: only the first source file may have it
	if len(yaplFiles) > 1 {
		for _, src := range yaplFiles[1:] {
			data, err := os.ReadFile(src)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ya: cannot read %s: %v\n", src, err)
				os.Exit(1)
			}
			if hasBootstrapPragma(data) {
				fmt.Fprintf(os.Stderr, "ya: #pragma bootstrap may only appear in the first source file\n")
				os.Exit(1)
			}
		}
	}

	// Link-only mode: only .wo files given
	if len(yaplFiles) == 0 {
		if err := link(inputWOs); err != nil {
			fmt.Fprintf(os.Stderr, "ya: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// -c or -S: compile each source file individually, no link step
	if *compileOnly || *asmOnly {
		for _, src := range yaplFiles {
			if err := compile(src); err != nil {
				fmt.Fprintf(os.Stderr, "ya: %s: %v\n", src, err)
				os.Exit(1)
			}
		}
		return
	}

	// Single source file, no .wo inputs: single-file path (preserves bootstrap-to-binary)
	if len(yaplFiles) == 1 && len(inputWOs) == 0 {
		if err := compile(yaplFiles[0]); err != nil {
			fmt.Fprintf(os.Stderr, "ya: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Multi-file compile+link: compile each source to a temp .wo, then link all
	compiledWOs := make([]string, 0, len(yaplFiles))
	for _, src := range yaplFiles {
		wo, err := compileForLink(src)
		if err != nil {
			for _, f := range compiledWOs {
				os.Remove(f)
			}
			fmt.Fprintf(os.Stderr, "ya: %s: %v\n", src, err)
			os.Exit(1)
		}
		compiledWOs = append(compiledWOs, wo)
	}
	defer func() {
		for _, f := range compiledWOs {
			os.Remove(f)
		}
	}()

	allWOs := append(compiledWOs, inputWOs...)
	if err := link(allWOs); err != nil {
		fmt.Fprintf(os.Stderr, "ya: %v\n", err)
		os.Exit(1)
	}
}

// pipelineResult holds the output of the compilation pipeline for a single source file.
type pipelineResult struct {
	asmOut       []byte
	hasBootstrap bool
	baseNoExt    string
	sourceDir    string
}

// runPipeline runs ylex→yparse→ysem→ygen (plus optional peephole optimizer and
// boot.asm prepend) for a single YAPL source file. It handles -k and -S file
// writing. Returns the assembly bytes and metadata needed by the caller.
func runPipeline(sourceFile string) (*pipelineResult, error) {
	// Verify source file exists
	if _, err := os.Stat(sourceFile); err != nil {
		return nil, fmt.Errorf("cannot access %s: %v", sourceFile, err)
	}

	// Determine base name for intermediate files
	sourceDir := filepath.Dir(sourceFile)
	baseName := filepath.Base(sourceFile)
	ext := filepath.Ext(baseName)
	baseNoExt := strings.TrimSuffix(baseName, ext)

	// Find compiler components
	ylexPath, err := findBinary("ylex", "ylex")
	if err != nil {
		return nil, err
	}
	parsePath, err := findBinary("yparse", "yparse")
	if err != nil {
		return nil, err
	}
	semPath, err := findBinary("ysem", "ysem")
	if err != nil {
		return nil, err
	}
	genPath, err := findBinary("ygen", "ygen")
	if err != nil {
		return nil, err
	}

	var ypeepPath string
	if *doOptimize {
		ypeepPath, err = findBinary("ypeep", "ypeep")
		if err != nil {
			return nil, err
		}
	}

	// Read source file
	sourceData, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("reading source: %v", err)
	}

	// Stage 1: Lexer
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running lexer...\n")
	}
	lexOut, err := runStage(ylexPath, []string{sourceFile}, bytes.NewReader(sourceData))
	if err != nil {
		return nil, fmt.Errorf("lexer failed: %v", err)
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
		return nil, fmt.Errorf("parser failed: %v", err)
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
		return nil, fmt.Errorf("semantic analyzer failed: %v", err)
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
		return nil, fmt.Errorf("code generator failed: %v", err)
	}

	// Prepend boot.asm for bootstrap programs
	hasBootstrap := hasBootstrapPragma(sourceData)
	if hasBootstrap {
		bootAsmPath, err := findBootAsm()
		if err != nil {
			return nil, err
		}
		bootData, err := os.ReadFile(bootAsmPath)
		if err != nil {
			return nil, fmt.Errorf("reading boot.asm: %v", err)
		}
		asmOut = append(bootData, asmOut...)
	}

	// Stage 5 (optional): Peephole optimizer
	if *doOptimize {
		if *verbose {
			fmt.Fprintf(os.Stderr, "Running peephole optimizer...\n")
		}
		asmOut, err = runStage(ypeepPath, nil, bytes.NewReader(asmOut))
		if err != nil {
			return nil, fmt.Errorf("peephole optimizer failed: %v", err)
		}
	}

	// Write .asm file if -k or -S
	if *keepFiles || *asmOnly {
		asmFile := filepath.Join(sourceDir, baseNoExt+".asm")
		if err := os.WriteFile(asmFile, asmOut, 0644); err != nil {
			return nil, fmt.Errorf("writing assembly: %v", err)
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote %s\n", asmFile)
		}
	}

	return &pipelineResult{
		asmOut:       asmOut,
		hasBootstrap: hasBootstrap,
		baseNoExt:    baseNoExt,
		sourceDir:    sourceDir,
	}, nil
}

// runAssembler invokes yasm with the given arguments.
func runAssembler(asmPath string, args []string) error {
	cmd := exec.Command(asmPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("assembler failed: %s", stderr.String())
		}
		return fmt.Errorf("assembler failed: %v", err)
	}
	return nil
}

// compile compiles a single YAPL source file using single-file mode logic.
// In -S mode: writes <base>.asm and stops.
// In -c mode: assembles to <base>.wo.
// With #pragma bootstrap: assembles directly to a binary (legacy path).
// Otherwise: assembles to a temp .wo and calls link().
func compile(sourceFile string) error {
	res, err := runPipeline(sourceFile)
	if err != nil {
		return err
	}

	if *asmOnly {
		return nil
	}

	asmPath, err := findBinary("yasm", "yasm")
	if err != nil {
		return err
	}

	// Write assembly to temp file for assembler
	tmpAsm, err := os.CreateTemp("", "ya-*.asm")
	if err != nil {
		return fmt.Errorf("creating temp file: %v", err)
	}
	tmpAsmName := tmpAsm.Name()
	defer os.Remove(tmpAsmName)

	if _, err := tmpAsm.Write(res.asmOut); err != nil {
		tmpAsm.Close()
		return fmt.Errorf("writing temp assembly: %v", err)
	}
	tmpAsm.Close()

	if *verbose {
		fmt.Fprintf(os.Stderr, "Running assembler...\n")
	}

	if *compileOnly {
		outFile := *outputFile
		if outFile == "" {
			outFile = filepath.Join(res.sourceDir, res.baseNoExt+".wo")
		}
		if err := runAssembler(asmPath, []string{"-c", "-o", outFile, tmpAsmName}); err != nil {
			return err
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote %s\n", outFile)
		}
		return nil
	}

	if res.hasBootstrap {
		// Bootstrap program: yasm produces binary directly (legacy path)
		outFile := *outputFile
		if outFile == "" {
			outFile = "wut4.out"
		}
		if err := runAssembler(asmPath, []string{"-o", outFile, tmpAsmName}); err != nil {
			return err
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote %s\n", outFile)
		}
		return nil
	}

	// Normal: assemble to temp .wo, then link
	tmpWO, err := os.CreateTemp("", "ya-*.wo")
	if err != nil {
		return fmt.Errorf("creating temp object file: %v", err)
	}
	tmpWOName := tmpWO.Name()
	tmpWO.Close()
	defer os.Remove(tmpWOName)

	if err := runAssembler(asmPath, []string{"-c", "-o", tmpWOName, tmpAsmName}); err != nil {
		return err
	}
	return link([]string{tmpWOName})
}

// compileForLink compiles a YAPL source file to a temporary .wo object file.
// The caller is responsible for removing the returned temp file.
func compileForLink(sourceFile string) (string, error) {
	res, err := runPipeline(sourceFile)
	if err != nil {
		return "", err
	}

	asmPath, err := findBinary("yasm", "yasm")
	if err != nil {
		return "", err
	}

	tmpAsm, err := os.CreateTemp("", "ya-*.asm")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %v", err)
	}
	tmpAsmName := tmpAsm.Name()
	defer os.Remove(tmpAsmName)

	if _, err := tmpAsm.Write(res.asmOut); err != nil {
		tmpAsm.Close()
		return "", fmt.Errorf("writing temp assembly: %v", err)
	}
	tmpAsm.Close()

	if *verbose {
		fmt.Fprintf(os.Stderr, "Running assembler...\n")
	}

	tmpWO, err := os.CreateTemp("", "ya-*.wo")
	if err != nil {
		return "", fmt.Errorf("creating temp object file: %v", err)
	}
	tmpWOName := tmpWO.Name()
	tmpWO.Close()

	if err := runAssembler(asmPath, []string{"-c", "-o", tmpWOName, tmpAsmName}); err != nil {
		os.Remove(tmpWOName)
		return "", err
	}
	return tmpWOName, nil
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

	args := []string{"-o", outFile}
	if *verbose {
		args = append(args, "-v")
	}
	args = append(args, woFiles...)

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
// Looks at $YAPL/../lib/boot.asm, then <bindir>/boot.asm, then <bindir>/../lib/boot.asm.
func findBootAsm() (string, error) {
	// Try $YAPL/../lib/boot.asm
	if yaplDir := os.Getenv("YAPL"); yaplDir != "" {
		p := filepath.Join(yaplDir, "..", "lib", "boot.asm")
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}

	exe, err := os.Executable()
	if err == nil {
		bindir := filepath.Dir(exe)

		// Try <bindir>/boot.asm (flat install: boot.asm alongside binaries)
		p := filepath.Join(bindir, "boot.asm")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}

		// Try <bindir>/../lib/boot.asm
		p = filepath.Join(bindir, "..", "lib", "boot.asm")
		if _, err := os.Stat(p); err == nil {
			return filepath.Clean(p), nil
		}
	}

	return "", fmt.Errorf("boot.asm not found; set YAPL env var to repo root or install boot.asm alongside binaries")
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
