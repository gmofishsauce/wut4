package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	ylexBin   string
	yparseBin string
	ysemBin   string
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "ysem-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	ylexBin = filepath.Join(tmp, "ylex")
	cmd := exec.Command("go", "build", "-o", ylexBin, ".")
	cmd.Dir = filepath.Join("..", "ylex")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build ylex: " + err.Error())
	}

	yparseBin = filepath.Join(tmp, "yparse")
	cmd = exec.Command("go", "build", "-o", yparseBin, ".")
	cmd.Dir = filepath.Join("..", "yparse")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build yparse: " + err.Error())
	}

	ysemBin = filepath.Join(tmp, "ysem")
	cmd = exec.Command("go", "build", "-o", ysemBin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build ysem: " + err.Error())
	}

	os.Exit(m.Run())
}

// runPipeline runs ylex | yparse | ysem on the given input file.
// Returns stdout, stderr, and the exit error (nil on success).
func runPipeline(t *testing.T, inputPath string) (string, string, error) {
	t.Helper()

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	basename := filepath.Base(inputPath)

	// ylex
	lexCmd := exec.Command(ylexBin, basename)
	lexCmd.Stdin = bytes.NewReader(inputData)
	var lexOut, lexErr bytes.Buffer
	lexCmd.Stdout = &lexOut
	lexCmd.Stderr = &lexErr
	if err := lexCmd.Run(); err != nil {
		return "", lexErr.String(), err
	}

	// yparse
	parseCmd := exec.Command(yparseBin)
	parseCmd.Stdin = &lexOut
	var parseOut, parseErr bytes.Buffer
	parseCmd.Stdout = &parseOut
	parseCmd.Stderr = &parseErr
	if err := parseCmd.Run(); err != nil {
		return "", parseErr.String(), err
	}

	// ysem
	semCmd := exec.Command(ysemBin)
	semCmd.Stdin = &parseOut
	var semOut, semErr bytes.Buffer
	semCmd.Stdout = &semOut
	semCmd.Stderr = &semErr

	err = semCmd.Run()
	return semOut.String(), semErr.String(), err
}

// TestImplicitIntConversionRejected verifies that ysem rejects an assignment
// of uint16 to int16 without an explicit cast. YAPL has no implicit integer
// type conversion; typesCompatible returns false for same-Kind/different-BaseType.
func TestImplicitIntConversionRejected(t *testing.T) {
	_, stderr, err := runPipeline(t, "testdata/err_uint16_to_int16.yapl")
	if err == nil {
		t.Fatal("expected ysem to reject implicit uint16->int16 assignment, but it succeeded")
	}
	if !strings.Contains(stderr, "type mismatch") {
		t.Errorf("expected 'type mismatch' in stderr, got: %s", stderr)
	}
}

// TestForLoopIR verifies that ysem correctly lowers all three parts of a for
// loop (init, condition, post) into the IR.
//
// Bug C4: reader.go expects the FOR body to start with three bare EMPTY-or-expr
// lines, but yparse emits INIT/COND/POST/DO section headers.  The reader
// discards the INIT header (treating it as an empty expression), misassigns
// the real init expression to Cond, and discards the real Cond and Post.
// Result: no signed less-than instruction appears in the IR.
func TestForLoopIR(t *testing.T) {
	ir, _, err := runPipeline(t, "testdata/for_loop.yapl")
	if err != nil {
		t.Fatalf("ysem failed unexpectedly: %v", err)
	}
	// The condition "i < n" with int16 operands must produce a signed
	// comparison.  With the bug the condition is the init assignment
	// (i = 0) which generates no comparison instruction.
	if !strings.Contains(ir, "LT.S") {
		t.Errorf("expected LT.S (signed less-than for loop condition) in IR\ngot:\n%s", ir)
	}
}

// TestStringLiteralReturn verifies that ysem accepts &"string" as a @byte
// return value and emits an ADDR instruction for the anonymous string global.
//
// Bug C1: yparse emits "STR ..." for string literals; reader.go looks for
// "STRLIT".  The mismatch causes the literal to be silently dropped, turning
// the return into a void return.  ysem then rejects the non-void function
// with "non-void function must return a value".
func TestStringLiteralReturn(t *testing.T) {
	ir, stderr, err := runPipeline(t, "testdata/string_return.yapl")
	if err != nil {
		t.Fatalf("ysem failed: %v\nstderr: %s", err, stderr)
	}
	// A string literal in an expression must produce an ADDR instruction.
	if !strings.Contains(ir, "ADDR") {
		t.Errorf("expected ADDR instruction for string literal in IR\ngot:\n%s", ir)
	}
}

// TestArrowFieldAccess verifies that ysem correctly handles the -> operator
// and generates a register-indirect memory load for the field value.
//
// Bug C2: yparse emits "ARROW fieldname" (the keyword is literally "ARROW")
// for pointer-to-struct field access; reader.go has no ARROW case and returns
// nil, leaving the object-expression line unconsumed and corrupting the stream.
// Bug C3: reader.go also reads the field name from the wrong line for dot
// access (it reads a second line instead of using parts[1]).
// Both bugs convert the return expression to a void return, causing ysem to
// reject the non-void function with a type error.
func TestArrowFieldAccess(t *testing.T) {
	ir, stderr, err := runPipeline(t, "testdata/arrow_field.yapl")
	if err != nil {
		t.Fatalf("ysem failed: %v\nstderr: %s", err, stderr)
	}
	// p->y dereferences a pointer, so the IR must contain a load through a
	// virtual register rather than a direct SP-relative or global address.
	if !strings.Contains(ir, "LOAD.W [t") {
		t.Errorf("expected register-indirect LOAD.W (pointer dereference) in IR\ngot:\n%s", ir)
	}
}

// TestGotoTarget verifies that the JUMP instruction generated for a goto
// statement targets the label name, not the source line number.
//
// Bug C5: yparse emits "GOTO label linenum" (label first, line number second).
// reader.go parses parts[1] as the line number (Atoi("done") == 0) and
// parts[2] as the label name, so the JUMP target becomes the stringified line
// number instead of the label.
func TestGotoTarget(t *testing.T) {
	ir, _, err := runPipeline(t, "testdata/goto_label.yapl")
	if err != nil {
		t.Fatalf("ysem failed unexpectedly: %v", err)
	}
	if !strings.Contains(ir, "JUMP done") {
		t.Errorf("expected 'JUMP done' in IR; with the bug the target is the line number\ngot:\n%s", ir)
	}
}

// TestLogicalAndShortCircuit verifies that the right-hand operand of &&
// is only evaluated when the left-hand operand is non-zero.
//
// Bug C6: genBinary evaluates both operands unconditionally before the switch
// statement, so the short-circuit jump only governs the result register, not
// whether the right-hand call occurs.
//
// The test checks the IR structure: there must be a JUMPZ instruction between
// "CALL alwayszero" and "CALL alwaysone" so that alwaysone is skipped when
// alwayszero returns 0.
func TestLogicalAndShortCircuit(t *testing.T) {
	ir, _, err := runPipeline(t, "testdata/land_shortcircuit.yapl")
	if err != nil {
		t.Fatalf("ysem failed unexpectedly: %v", err)
	}

	lines := strings.Split(ir, "\n")
	callZeroLine := -1
	callOneLine := -1
	for i, l := range lines {
		if strings.Contains(l, "CALL alwayszero") {
			callZeroLine = i
		}
		if strings.Contains(l, "CALL alwaysone") {
			callOneLine = i
		}
	}
	if callZeroLine < 0 || callOneLine < 0 {
		t.Fatalf("expected both CALL alwayszero and CALL alwaysone in IR\ngot:\n%s", ir)
	}

	// For short-circuit &&, there must be a JUMPZ between the two calls so
	// that alwaysone() is skipped when alwayszero() returns 0.
	// With the bug both calls appear before any JUMPZ.
	found := false
	for i := callZeroLine + 1; i < callOneLine; i++ {
		if strings.Contains(lines[i], "JUMPZ") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected JUMPZ between CALL alwayszero and CALL alwaysone for short-circuit &&\ngot IR:\n%s", ir)
	}
}
