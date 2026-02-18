// Tests for ygen, Pass 4 of the YAPL compiler.
//
// The tests are organized into two groups:
//
//  1. Unit tests that call package-level functions directly (e.g. parseInt).
//  2. Integration tests that build the ygen binary, pipe IR through it, and
//     inspect the emitted assembly for structural correctness.
//
// To add a new integration test, place a .ir file under testdata/ and call
// runYgen with its content.

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ygenBin is the path to the ygen binary built by TestMain.
var ygenBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "ygen-test-")
	if err != nil {
		panic("MkdirTemp: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	ygenBin = filepath.Join(tmp, "ygen")
	cmd := exec.Command("go", "build", "-o", ygenBin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build ygen: " + err.Error())
	}

	os.Exit(m.Run())
}

// runYgen pipes irText through ygen and returns the assembly output.
// It fails the test if ygen exits with an error.
func runYgen(t *testing.T, irText string) string {
	t.Helper()
	cmd := exec.Command(ygenBin)
	cmd.Stdin = strings.NewReader(irText)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("ygen failed: %v\nstderr: %s", err, errBuf.String())
	}
	return out.String()
}

// runYgenFile reads an IR file from testdata/ and pipes it through ygen.
func runYgenFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile %s: %v", name, err)
	}
	return runYgen(t, string(data))
}

// ---------------------------------------------------------------------------
// Unit tests (call package functions directly)
// ---------------------------------------------------------------------------

// TestParseIntHex documents and catches a bug in parseInt: the function strips
// the "0x"/"0X" prefix and then re-checks for it (dead branch), so hex values
// are mis-parsed.
//
//   - parseInt("0x24") strips to "24" then calls ParseInt("24", 0, 32) = 24
//     instead of the correct 0x24 = 36.
//   - parseInt("0xFF") strips to "FF" then calls ParseInt("FF", 0, 32) which
//     fails (not a valid decimal/octal string) and returns 0 instead of 255.
//
// All current callers of parseInt pass decimal values, so the bug is latent.
// This test will fail until the function is fixed.
func TestParseIntHex(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"36", 36},
		{"0x24", 36},  // 0x24 == 36
		{"0X24", 36},  // uppercase prefix
		{"0xff", 255}, // lowercase hex digits
		{"0xFF", 255},
	}
	for _, c := range cases {
		got := parseInt(c.input)
		if got != c.want {
			t.Errorf("parseInt(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration tests (build and run ygen)
// ---------------------------------------------------------------------------

// TestSmokeHello runs ygen on the hello.ir integration IR and checks that
// the output contains the expected structural landmarks.
func TestSmokeHello(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "test", "hello.ir"))
	if err != nil {
		t.Fatalf("ReadFile hello.ir: %v", err)
	}
	asm := runYgen(t, string(data))

	for _, want := range []string{
		"_start:",  // bootstrap entry point
		"main:",    // main function label
		"Putstr:",  // Putstr function label
		"hello:",   // global data label
		".bytes",   // data initializer
		"jal main", // bootstrap call to main
		"hlt",      // halt after main returns
	} {
		if !strings.Contains(asm, want) {
			t.Errorf("expected %q in ygen output; not found\noutput:\n%s", want, asm)
		}
	}
}

// TestLinkAlwaysSaved verifies that every function emits code to save LINK on
// entry (lsp) and restore it on return (ssp).  LINK is unconditionally
// callee-saved per the ABI; skipping it for leaf functions is a future
// optimisation that has not yet been implemented.
func TestLinkAlwaysSaved(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "test", "hello.ir"))
	if err != nil {
		t.Fatalf("ReadFile hello.ir: %v", err)
	}
	asm := runYgen(t, string(data))

	// Count functions vs lsp/ssp pairs as a sanity check.
	lspCount := strings.Count(asm, "lsp r")
	sspCount := strings.Count(asm, "ssp r")
	if lspCount == 0 {
		t.Error("expected at least one 'lsp r' (LINK save) in output, found none")
	}
	if lspCount != sspCount {
		t.Errorf("LINK save/restore mismatch: %d lsp vs %d ssp", lspCount, sspCount)
	}
}

// compareFalsePath is a helper that verifies the structural correctness of a
// comparison operator's false path in the generated assembly.
//
// For a binary comparison that produces 0 (false) or 1 (true), the emitted
// code must ensure R6 is set to 0 on the false path before the result is
// stored.  The bug this catches: genCompare for GT.S and GT.U jumps the false
// branches directly to doneLabel (the store instruction), bypassing the
// "ldi r6, 0" that sets the false result, so the stored value is undefined.
//
// The invariant tested: every conditional branch that represents the false
// case must target a label that is defined BEFORE the "ldi r6, 0" line in the
// output, meaning the false result assignment is reachable.  If the branch
// targets a label that appears AFTER "ldi r6, 0", the assignment is bypassed.
//
// falseBranchMnemonics lists the branch instruction(s) (without operands) that
// encode the false condition for the operator under test.
func compareFalsePath(t *testing.T, asm string, falseBranchMnemonics []string) {
	t.Helper()

	lines := strings.Split(asm, "\n")

	// Find the position of the "ldi r6, 0" false-result assignment.
	ldi0Line := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "ldi r6, 0" {
			ldi0Line = i
			break
		}
	}
	if ldi0Line < 0 {
		t.Fatal("expected 'ldi r6, 0' (false-case result) in output; not found")
	}

	// For each false-branch mnemonic, find the branch and its target label.
	for _, mnemonic := range falseBranchMnemonics {
		branchLine := -1
		var target string
		for i, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, mnemonic+" ") {
				branchLine = i
				target = strings.TrimSpace(strings.TrimPrefix(l, mnemonic+" "))
				break
			}
		}
		if branchLine < 0 {
			t.Errorf("expected '%s' (false-case branch) in output; not found", mnemonic)
			continue
		}

		// Find where the target label is defined.
		targetDef := target + ":"
		targetLine := -1
		for i, l := range lines {
			if strings.TrimSpace(l) == targetDef {
				targetLine = i
				break
			}
		}
		if targetLine < 0 {
			t.Errorf("branch '%s %s' target label not found in output", mnemonic, target)
			continue
		}

		// The target label must be defined at or before "ldi r6, 0" so that
		// execution can reach the false-result assignment.
		if targetLine > ldi0Line {
			t.Errorf(
				"BUG: '%s %s' (line %d) branches to %q (line %d) which is AFTER 'ldi r6, 0' (line %d); "+
					"the false path bypasses the zero-assignment and stores an undefined value",
				mnemonic, target, branchLine+1, target, targetLine+1, ldi0Line+1,
			)
		}
	}
}

// TestGtSFalsePath is a regression test for the GT.S comparison code
// generation bug.
//
// In genCompare (codegen.go), the OpGtS case branches the false conditions
// (a < b, a == b) directly to doneLabel, which is the store instruction.
// This bypasses the "ldi r6, 0" that sets the false result, so the value
// stored when a <= b is whatever R6 happened to hold — undefined behaviour.
//
// The test verifies that the "brslt" instruction's target label is defined
// before (not after) the "ldi r6, 0" instruction in the output, meaning the
// false path is reachable.
func TestGtSFalsePath(t *testing.T) {
	asm := runYgenFile(t, "gts_compare.ir")
	// For GT.S, brslt encodes "a < b" (false case) and brz encodes "a == b" (false case).
	compareFalsePath(t, asm, []string{"brslt", "brz"})
}

// TestGtUFalsePath is the GT.U analogue of TestGtSFalsePath.
//
// The OpGtU case has the identical structural bug: brult and brz (false
// conditions) jump to doneLabel, skipping "ldi r6, 0".
func TestGtUFalsePath(t *testing.T) {
	asm := runYgenFile(t, "gtu_compare.ir")
	// For GT.U, brult encodes "a < b unsigned" (false case) and brz encodes "a == b".
	compareFalsePath(t, asm, []string{"brult", "brz"})
}

// TestAllComparesProduceFalseResult runs ygen on a file containing all ten
// comparison operators and verifies that every function's output contains at
// least one "ldi r6, 0" (false-case result) and one "ldi r6, 1" (true-case
// result).  A missing ldi r6, 0 would mean the false path stores a stale value.
func TestAllComparesProduceFalseResult(t *testing.T) {
	asm := runYgenFile(t, "all_compares.ir")

	// Split output by function (each function starts with its label "name:")
	// and check each separately.
	funcs := []string{
		"cmp_eq", "cmp_ne",
		"cmp_lts", "cmp_les", "cmp_gts", "cmp_ges",
		"cmp_ltu", "cmp_leu", "cmp_gtu", "cmp_geu",
	}

	for _, fn := range funcs {
		// Extract the function body from the assembly.
		startMarker := fn + ":"
		startIdx := strings.Index(asm, startMarker)
		if startIdx < 0 {
			t.Errorf("function %s not found in output", fn)
			continue
		}
		// Find the next function start or end of output.
		nextFunc := strings.Index(asm[startIdx+len(startMarker):], "\n; Function:")
		var body string
		if nextFunc < 0 {
			body = asm[startIdx:]
		} else {
			body = asm[startIdx : startIdx+len(startMarker)+nextFunc]
		}

		if !strings.Contains(body, "ldi r6, 0") {
			t.Errorf("function %s: no 'ldi r6, 0' (false result) found in body:\n%s", fn, body)
		}
		if !strings.Contains(body, "ldi r6, 1") {
			t.Errorf("function %s: no 'ldi r6, 1' (true result) found in body:\n%s", fn, body)
		}
	}
}
