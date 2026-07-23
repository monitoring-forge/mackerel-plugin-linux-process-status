//go:build !windows
// +build !windows

package main

import (
	"os"
	"syscall"
)

// openRD opens a file for reading with O_NOFOLLOW flag to prevent symlink attacks.
// This function is specific to Unix-like systems where syscall.O_NOFOLLOW is available.
func openRD(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}
