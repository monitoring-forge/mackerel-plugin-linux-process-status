package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/procfs"
)

// containsLine checks if a string contains a line with the given substring
func containsLine(s, substr string) bool {
	for line := range strings.SplitSeq(s, "\n") {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

func TestCpuJifferAt(t *testing.T) {
	dir := t.TempDir()

	// Create a pseudo /proc/stat
	statContent := `cpu  100 200 300 4000 50 60 70 0 0 0
cpu0 100 200 300 4000 50 60 70 0 0 0
intr 12345 5432 3456 2345 1234 1122 3344 5566 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 89 90 91 92 93 94 95 96 97 98 99 100
ctxt 54321
btime 1000000
processes 500
`
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(statContent), 0644); err != nil {
		t.Fatalf("failed to write stat file: %v", err)
	}

	result, err := cpuJifferAt(dir)
	if err != nil {
		t.Fatalf("cpuJifferAt returned error: %v", err)
	}

	// Expected: (User + Nice + System + Idle) / 100
	// = (100 + 200 + 300 + 4000) / 100 = 4600 / 100 = 46.0
	expected := float64(46)
	if result != expected {
		t.Errorf("cpuJifferAt = %f, want %f", result, expected)
	}
}

func TestCpuJifferAtInvalidPath(t *testing.T) {
	_, err := cpuJifferAt("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("cpuJifferAt on invalid path returned nil error, want non-nil error")
	}
}

func TestCpuJifferAtNoStatFile(t *testing.T) {
	dir := t.TempDir()

	// Create a directory without stat file
	// This should cause an error when trying to read /proc/stat
	_, err := cpuJifferAt(dir)
	if err == nil {
		t.Error("cpuJifferAt without stat file returned nil error, want non-nil error")
	}
}

func TestMemStatAt(t *testing.T) {
	dir := t.TempDir()

	// Create /proc/1/ directory structure
	procDir := filepath.Join(dir, "1")
	if err := os.MkdirAll(procDir, 0755); err != nil {
		t.Fatalf("failed to create proc dir: %v", err)
	}

	// Create /proc/1/stat file (required for p.Stat())
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte(
		`1 (bash) S 0 1 1 0 -1 4194304 100 0 0 0 100 200 0 0 20 0 1 0 100 12345 1024 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0`,
	), 0644); err != nil {
		t.Fatalf("failed to write stat file: %v", err)
	}

	// Create /proc/meminfo file (required for MemTotal)
	meminfoContent := `MemTotal:        8000000 kB
MemFree:         4000000 kB
MemAvailable:    5000000 kB
Buffers:          200000 kB
Cached:          2000000 kB
`
	if err := os.WriteFile(filepath.Join(dir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatalf("failed to write meminfo file: %v", err)
	}

	// Create FS and get Proc
	fs, err := procfs.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	p, err := fs.Proc(1)
	if err != nil {
		t.Fatalf("fs.Proc(1) failed: %v", err)
	}

	result, err := memStatAt(p, "test", 1234567890, dir)
	if err != nil {
		t.Fatalf("memStatAt failed: %v", err)
	}

	// Verify output contains expected metric lines
	expectedLines := []string{
		"process-status.mem_test.used",
		"process-status.mem_test.max",
		"process-status.mem_usage_test.percentage",
	}

	for _, line := range expectedLines {
		if !containsLine(result, line) {
			t.Errorf("memStatAt result missing line containing %q:\n%q", line, result)
		}
	}
}

func TestMemStatAtNilMemTotal(t *testing.T) {
	dir := t.TempDir()

	// Create /proc/1/ directory structure
	procDir := filepath.Join(dir, "1")
	if err := os.MkdirAll(procDir, 0755); err != nil {
		t.Fatalf("failed to create proc dir: %v", err)
	}

	// Create /proc/1/stat file
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte(
		`1 (bash) S 0 1 1 0 -1 4194304 100 0 0 0 100 200 0 0 20 0 1 0 100 12345 1024 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0`,
	), 0644); err != nil {
		t.Fatalf("failed to write stat file: %v", err)
	}

	// Create /proc/meminfo without MemTotal (should cause error)
	meminfoContent := `MemFree:         4000000 kB
MemAvailable:    5000000 kB
`
	if err := os.WriteFile(filepath.Join(dir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatalf("failed to write meminfo file: %v", err)
	}

	fs, err := procfs.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	p, err := fs.Proc(1)
	if err != nil {
		t.Fatalf("fs.Proc(1) failed: %v", err)
	}

	_, err = memStatAt(p, "test", 1234567890, dir)
	if err == nil {
		t.Error("memStatAt without MemTotal returned nil error, want non-nil error")
	}
}

func TestCpuStatAt(t *testing.T) {
	dir := t.TempDir()

	// Create /proc/1/ directory structure
	procDir := filepath.Join(dir, "1")
	if err := os.MkdirAll(procDir, 0755); err != nil {
		t.Fatalf("failed to create proc dir: %v", err)
	}

	// Create /proc/1/stat file
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte(
		`1 (bash) S 0 1 1 0 -1 4194304 100 0 0 0 100 200 0 0 20 0 1 0 100 12345 1024 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0`,
	), 0644); err != nil {
		t.Fatalf("failed to write stat file: %v", err)
	}

	// Create /proc/stat file
	statContent := `cpu  100 200 300 4000 50 60 70 0 0 0
cpu0 100 200 300 4000 50 60 70 0 0 0
`
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(statContent), 0644); err != nil {
		t.Fatalf("failed to write stat file: %v", err)
	}

	fs, err := procfs.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	p, err := fs.Proc(1)
	if err != nil {
		t.Fatalf("fs.Proc(1) failed: %v", err)
	}

	opt := &Opt{
		KeyPrefix: "test",
	}

	// First time execution - state file doesn't exist
	// Need to create workDir that doesn't have the state file
	workDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}

	// Override pluginutil.PluginWorkDir behavior by using temp dir
	result, err := cpuStatAt(p, opt, 1234567890, workDir, dir)
	if err != nil {
		t.Fatalf("cpuStatAt failed: %v", err)
	}

	// First time should return empty result
	if result != "" {
		t.Errorf("cpuStatAt first time returned non-empty result: %q", result)
	}

	// update proc/stat
	statContent = `cpu  110 220 330 4000 50 60 70 0 0 0
cpu0 100 200 300 4000 50 60 70 0 0 0
`
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(statContent), 0644); err != nil {
		t.Fatalf("failed to update stat file: %v", err)
	}

	// Update /proc/1/stat file
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte(
		`1 (bash) S 0 1 1 0 -1 4194304 100 0 0 0 110 220 0 0 20 0 1 0 100 12345 1024 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0`,
	), 0644); err != nil {
		t.Fatalf("failed to update stat file: %v", err)
	}

	// Now run cpuStatAt again, it should read the previous stats and compute usage
	result, err = cpuStatAt(p, opt, 1234567891, workDir, dir)
	if err != nil {
		t.Fatalf("cpuStatAt second time failed: %v", err)
	}

	if !containsLine(result, "process-status.cpu_test.percentage\t50.000000\t1234567891") {
		t.Errorf("cpuStatAt result missing line containing process-status.cpu_test.percentage:\n%q", result)
	}
}
