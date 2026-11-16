//go:build !windows

package main

import "os"

func removeDir(path string) error {
	return os.RemoveAll(path)
}
