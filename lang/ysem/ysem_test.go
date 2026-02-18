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
