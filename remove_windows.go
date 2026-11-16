//go:build windows

package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

func removeDir(path string) error {
	err := os.RemoveAll(path)
	if err == nil || os.IsNotExist(err) {
		return err
	}
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d == nil {
			return nil
		}
		if fi, infoErr := d.Info(); infoErr == nil && fi.Mode()&0200 == 0 {
			_ = os.Chmod(p, fi.Mode()|0200)
		}
		return nil
	})
	return os.RemoveAll(path)
}
