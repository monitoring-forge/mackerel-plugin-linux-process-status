package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func fileExists(dir, filename string) bool {
	_, err := os.Stat(filepath.Join(dir, filename))
	return err == nil
}

func writeStats(dir, filename string, ps *processStats) error {
	newFile, err := os.CreateTemp(dir, "process-status-")
	if err != nil {
		return err
	}

	je := json.NewEncoder(newFile)
	err = je.Encode(ps)
	if err != nil {
		newFile.Close()
		errRemove := os.Remove(newFile.Name())
		if errRemove != nil {
			log.Printf("Failed to remove temporary file: %s, error: %v", newFile.Name(), errRemove)
		}
		return err
	}

	err = newFile.Close()
	if err != nil {
		errRemove := os.Remove(newFile.Name())
		if errRemove != nil {
			log.Printf("Failed to remove temporary file: %s, error: %v", newFile.Name(), errRemove)
		}
		return err
	}

	return os.Rename(newFile.Name(), filepath.Join(dir, filename))
}

func readStats(dir, filename string) (*processStats, error) {
	filename = filepath.Join(dir, filename)

	file, err := openRD(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	ps := &processStats{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(ps)
	if err != nil {
		return nil, err
	}
	return ps, nil
}
