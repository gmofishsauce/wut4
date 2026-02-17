package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runYlex runs ylex with the given source as stdin and the given filename
// argument.  It returns stdout, stderr, and whether the process exited zero.
func runYlex(t *testing.T, filename, src string) (stdout, stderr string, ok bool) {
	t.Helper()
	cmd := exec.Command(ylexBin, filename)
	cmd.Stdin = strings.NewReader(src)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err == nil
}

// TestBugRegression contains one subtest per warning in ylex-review.md.
// Each test asserts the CORRECT (fixed) behavior.  All four subtests are
// currently expected to FAIL because the bugs have not yet been fixed.
func TestBugRegression(t *testing.T) {
	// W1: Unterminated block comment should be a fatal error, not silently
	// accepted.  The lexer currently exits 0 with an empty token stream.
	t.Run("W1_unterminated_block_comment", func(t *testing.T) {
		_, stderr, ok := runYlex(t, "w1.yapl", "/* this comment never ends\n")
		if ok {
			t.Error("W1: ylex exited 0; want non-zero exit for unterminated block comment")
		}
		if !strings.Contains(stderr, "unterminated") {
			t.Errorf("W1: stderr %q; want it to contain \"unterminated\"", stderr)
		}
	})

	// W2: Identifiers longer than 15 characters should be a fatal error.
	// The lexer currently passes them through without complaint.
	t.Run("W2_identifier_too_long", func(t *testing.T) {
		// "averylongidentname" is 18 characters, over the 15-char limit.
		src := "var int16 averylongidentname;\n"
		_, stderr, ok := runYlex(t, "w2.yapl", src)
		if ok {
			t.Error("W2: ylex exited 0; want non-zero exit for identifier longer than 15 chars")
		}
		if !strings.Contains(stderr, "identifier") {
			t.Errorf("W2: stderr %q; want it to mention \"identifier\"", stderr)
		}
	})

	// W3: A negative shift count in a constant expression should produce a
	// clean error, not a runtime panic.  The lexer currently panics, which
	// exits non-zero but prints a goroutine trace instead of a lexer error.
	t.Run("W3_negative_shift_count", func(t *testing.T) {
		src := "#if 1 << -1\n#endif\n"
		_, stderr, ok := runYlex(t, "w3.yapl", src)
		if ok {
			t.Error("W3: ylex exited 0; want non-zero exit for negative shift count")
		}
		if strings.Contains(stderr, "goroutine") {
			t.Errorf("W3: ylex panicked instead of reporting a clean error:\n%s", stderr)
		}
		if !strings.Contains(stderr, "negative shift") {
			t.Errorf("W3: stderr %q; want it to contain \"negative shift\"", stderr)
		}
	})

	// W4: A var declaration without a terminating semicolon should be a fatal
	// error.  The lexer currently silently accepts it, unlike const which
	// already errors on the same condition.
	t.Run("W4_var_missing_semicolon", func(t *testing.T) {
		src := "var int16 x\n"
		_, stderr, ok := runYlex(t, "w4.yapl", src)
		if ok {
			t.Error("W4: ylex exited 0; want non-zero exit for var declaration missing semicolon")
		}
		if !strings.Contains(stderr, ";") {
			t.Errorf("W4: stderr %q; want it to mention missing \";\"", stderr)
		}
	})
}

var ylexBin string

func TestMain(m *testing.M) {
	// Build ylex to a temp directory
	tmp, err := os.MkdirTemp("", "ylex-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	ylexBin = filepath.Join(tmp, "ylex")
	cmd := exec.Command("go", "build", "-o", ylexBin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build ylex: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestLexer(t *testing.T) {
	entries, err := filepath.Glob("testdata/*.yapl")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no test files found in testdata/")
	}

	for _, input := range entries {
		name := strings.TrimSuffix(filepath.Base(input), ".yapl")
		expected := filepath.Join("testdata", name+".expected")

		if _, err := os.Stat(expected); os.IsNotExist(err) {
			t.Logf("SKIP: %s (no expected output)", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Read input file
			inputData, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}

			// Read expected output
			expectedData, err := os.ReadFile(expected)
			if err != nil {
				t.Fatal(err)
			}

			// Run ylex with filename arg and input on stdin
			cmd := exec.Command(ylexBin, name+".yapl")
			cmd.Stdin = bytes.NewReader(inputData)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("ylex failed: %v\nstderr: %s", err, stderr.String())
			}

			actual := stdout.String()
			want := string(expectedData)
			if actual != want {
				t.Errorf("output mismatch for %s\n--- expected ---\n%s\n--- actual ---\n%s", name, want, actual)
			}
		})
	}
}
