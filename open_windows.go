//go:build windows
// +build windows

package main

import "os"

// openRD opens a file for reading with appropriate flags for Windows.
// Since Windows doesn't support O_NOFOLLOW, we simply open the file with O_RDONLY.
func openRD(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDONLY, 0)
}
