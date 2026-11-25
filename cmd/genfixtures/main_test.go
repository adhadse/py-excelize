package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pyexcelize "github.com/adhadse/py-excelize"
	"github.com/adhadse/py-excelize/internal/fixtures"
)

func TestRunWritesFixtures(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-out", dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	want := []string{fixtures.FileCells, fixtures.FileControls, fixtures.FileEmpty, fixtures.FileEncrypted}
	if len(lines) != len(want) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for i, name := range want {
		exp := "wrote " + filepath.Join(dir, name)
		if lines[i] != exp {
			t.Errorf("line %d = %q, want %q", i, lines[i], exp)
		}
		if fi, err := os.Stat(filepath.Join(dir, name)); err != nil || fi.Size() == 0 {
			t.Errorf("%s not written: %v", name, err)
		}
	}
	wb, err := pyexcelize.Open(filepath.Join(dir, fixtures.FileEncrypted), pyexcelize.WithPassword(fixtures.Password))
	if err != nil {
		t.Fatal(err)
	}
	wb.Close()
}

func TestRunDefaultOutDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, "testdata", fixtures.FileCells)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Join("testdata", fixtures.FileEmpty)) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFailures(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		args       []string
		code       int
		stderrHas  string
		stdoutNone bool
	}{
		{"unwritable out", []string{"-out", filepath.Join(file, "x")}, 1, "", true},
		{"out is a file", []string{"-out", file}, 1, "", true},
		{"unknown flag", []string{"-bogus"}, 2, "flag provided but not defined", true},
		{"missing flag value", []string{"-out"}, 2, "needs an argument", true},
		{"help", []string{"-h"}, 0, "Usage of genfixtures", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, &stdout, &stderr); code != tc.code {
				t.Fatalf("exit = %d, want %d", code, tc.code)
			}
			if tc.stdoutNone && stdout.Len() != 0 {
				t.Fatalf("stdout = %q", stdout.String())
			}
			if stderr.Len() == 0 || !strings.Contains(stderr.String(), tc.stderrHas) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.stderrHas)
			}
		})
	}
}
