package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
