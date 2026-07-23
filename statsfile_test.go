package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	// Existing file
	existFile := filepath.Join(dir, "exist.txt")
	if err := os.WriteFile(existFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !fileExists(dir, "exist.txt") {
		t.Error("fileExists(\"exist.txt\") = false, want true")
	}

	// Non-existing file
	if fileExists(dir, "nonexist.txt") {
		t.Error("fileExists(\"nonexist.txt\") = true, want false")
	}

	// Non-existing directory
	if fileExists(filepath.Join(dir, "nodir"), "file.txt") {
		t.Error("fileExists in non-existing dir = true, want false")
	}
}

func TestWriteReadStats(t *testing.T) {
	dir := t.TempDir()

	ps := &processStats{
		Now:    1234567890,
		SysCPU: 0.5,
		CPU:    25.0,
	}

	err := writeStats(dir, "stats.json", ps)
	if err != nil {
		t.Fatalf("writeStats failed: %v", err)
	}

	// Verify file exists
	if !fileExists(dir, "stats.json") {
		t.Fatal("stats file was not created")
	}

	// Verify the file content is valid JSON
	content, err := os.ReadFile(filepath.Join(dir, "stats.json"))
	if err != nil {
		t.Fatalf("failed to read stats file: %v", err)
	}

	var decoded processStats
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("failed to unmarshal stats JSON: %v", err)
	}

	if decoded.Now != ps.Now {
		t.Errorf("Now = %d, want %d", decoded.Now, ps.Now)
	}
	if decoded.SysCPU != ps.SysCPU {
		t.Errorf("SysCPU = %f, want %f", decoded.SysCPU, ps.SysCPU)
	}
	if decoded.CPU != ps.CPU {
		t.Errorf("CPU = %f, want %f", decoded.CPU, ps.CPU)
	}

	// Read back using readStats
	readPs, err := readStats(dir, "stats.json")
	if err != nil {
		t.Fatalf("readStats failed: %v", err)
	}

	if readPs.Now != ps.Now {
		t.Errorf("readStats Now = %d, want %d", readPs.Now, ps.Now)
	}
	if readPs.SysCPU != ps.SysCPU {
		t.Errorf("readStats SysCPU = %f, want %f", readPs.SysCPU, ps.SysCPU)
	}
	if readPs.CPU != ps.CPU {
		t.Errorf("readStats CPU = %f, want %f", readPs.CPU, ps.CPU)
	}
}

func TestWriteStatsToNonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	ps := &processStats{
		Now:    123,
		SysCPU: 0.1,
		CPU:    1.0,
	}

	err := writeStats(dir, "stats.json", ps)
	if err == nil {
		t.Error("writeStats in non-existent dir returned nil error, want non-nil error")
	}
}

func TestReadStatsFileNotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := readStats(dir, "nonexist.json")
	if err == nil {
		t.Error("readStats on non-existent file returned nil error, want non-nil error")
	}
}

func TestReadStatsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(badFile, []byte("not valid json {{{"), 0644); err != nil {
		t.Fatalf("failed to create invalid JSON file: %v", err)
	}

	_, err := readStats(dir, "invalid.json")
	if err == nil {
		t.Error("readStats on invalid JSON returned nil error, want non-nil error")
	}
}

func TestWriteStatsOverwrite(t *testing.T) {
	dir := t.TempDir()

	ps1 := &processStats{
		Now:    111,
		SysCPU: 1.0,
		CPU:    10.0,
	}

	if err := writeStats(dir, "stats.json", ps1); err != nil {
		t.Fatalf("first writeStats failed: %v", err)
	}

	ps2 := &processStats{
		Now:    222,
		SysCPU: 2.0,
		CPU:    20.0,
	}

	if err := writeStats(dir, "stats.json", ps2); err != nil {
		t.Fatalf("second writeStats failed: %v", err)
	}

	readPs, err := readStats(dir, "stats.json")
	if err != nil {
		t.Fatalf("readStats failed: %v", err)
	}

	if readPs.Now != ps2.Now {
		t.Errorf("After overwrite, Now = %d, want %d", readPs.Now, ps2.Now)
	}
	if readPs.SysCPU != ps2.SysCPU {
		t.Errorf("After overwrite, SysCPU = %f, want %f", readPs.SysCPU, ps2.SysCPU)
	}
	if readPs.CPU != ps2.CPU {
		t.Errorf("After overwrite, CPU = %f, want %f", readPs.CPU, ps2.CPU)
	}
}

func TestWriteStatsNoTempFileLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()

	// Ensure no leftover temp files
	initialTempFiles, _ := filepath.Glob(filepath.Join(dir, "process-status-*"))

	ps := &processStats{
		Now:    999,
		SysCPU: 0.1,
		CPU:    1.0,
	}

	// This should succeed and leave no temp files
	err := writeStats(dir, "stats.json", ps)
	if err != nil {
		t.Fatalf("writeStats failed: %v", err)
	}

	// Verify no temp files remain
	afterTempFiles, _ := filepath.Glob(filepath.Join(dir, "process-status-*"))
	if len(afterTempFiles) != len(initialTempFiles) {
		t.Errorf("temp files were left behind: before=%d, after=%d", len(initialTempFiles), len(afterTempFiles))
	}
}

func TestReadStatsWithCustomFilename(t *testing.T) {
	dir := t.TempDir()

	ps := &processStats{
		Now:    42,
		SysCPU: 0.01,
		CPU:    99.9,
	}

	filename := "custom_stats.json"
	if err := writeStats(dir, filename, ps); err != nil {
		t.Fatalf("writeStats failed: %v", err)
	}

	readPs, err := readStats(dir, filename)
	if err != nil {
		t.Fatalf("readStats failed: %v", err)
	}

	if readPs.Now != ps.Now || readPs.SysCPU != ps.SysCPU || readPs.CPU != ps.CPU {
		t.Error("readStats returned different values than written")
	}
}
