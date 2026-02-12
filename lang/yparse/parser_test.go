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
)

func TestMain(m *testing.M) {
	// Build both binaries to a temp directory
	tmp, err := os.MkdirTemp("", "yparse-test")
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
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build yparse: " + err.Error())
	}

	os.Exit(m.Run())
}

// runPipeline runs ylex | yparse on the given input file.
// Returns stdout, stderr, and the exit error (nil on success).
func runPipeline(t *testing.T, inputPath string) (string, string, error) {
	t.Helper()

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	basename := filepath.Base(inputPath)

	// Run ylex
	lexCmd := exec.Command(ylexBin, basename)
	lexCmd.Stdin = bytes.NewReader(inputData)
	var lexOut, lexErr bytes.Buffer
	lexCmd.Stdout = &lexOut
	lexCmd.Stderr = &lexErr

	if err := lexCmd.Run(); err != nil {
		// Lexer failure is itself a valid way for a negative test to fail
		return "", lexErr.String(), err
	}

	// Run yparse with ylex output as stdin
	parseCmd := exec.Command(yparseBin)
	parseCmd.Stdin = &lexOut
	var parseOut, parseErr bytes.Buffer
	parseCmd.Stdout = &parseOut
	parseCmd.Stderr = &parseErr

	err = parseCmd.Run()
	return parseOut.String(), parseErr.String(), err
}

func TestParserPositive(t *testing.T) {
	entries, err := filepath.Glob("testdata/[0-9]*.yapl")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no positive test files found in testdata/")
	}

	for _, input := range entries {
		name := strings.TrimSuffix(filepath.Base(input), ".yapl")
		t.Run(name, func(t *testing.T) {
			_, stderr, err := runPipeline(t, input)
			if err != nil {
				t.Errorf("expected success but got error: %v\nstderr: %s", err, stderr)
			}
		})
	}
}

func TestParserNegative(t *testing.T) {
	entries, err := filepath.Glob("testdata/err_*.yapl")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no negative test files found in testdata/")
	}

	for _, input := range entries {
		name := strings.TrimSuffix(filepath.Base(input), ".yapl")
		t.Run(name, func(t *testing.T) {
			_, stderr, err := runPipeline(t, input)
			if err == nil {
				t.Errorf("expected failure but got success (should have been rejected)")
			} else {
				t.Logf("correctly rejected: %s", strings.Split(stderr, "\n")[0])
			}
		})
	}
}
