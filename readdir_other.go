//go:build !linux && !windows

package main

import (
	"os"
	"unsafe"
)

func readDirRaw(path string, visit func(name []byte, isDir bool) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	for _, e := range entries {
		if !visit(nameBytes(e.Name()), e.IsDir()) {
			break
		}
	}
	return err
}

func nameBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
