package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type dirInfo struct {
	path          string
	dirNames      []string
	files         bitset
	dirs          bitset
	suffixMask    uint64
	dirSuffixMask uint64
	dirSuffixHits []string
}

func readDirInfo(path string) (*dirInfo, error) {
	d := &dirInfo{path: path}
	err := readDirRaw(path, func(name []byte, isDir bool) bool {
		if isDir {
			s := string(name)
			d.dirNames = append(d.dirNames, s)
			if id, ok := markerIDs[s]; ok {
				d.dirs.set(id)
			}
			if m := suffixMask(name, &dirSuffixTable); m != 0 {
				d.dirSuffixMask |= m
				d.dirSuffixHits = append(d.dirSuffixHits, s)
			}
			return true
		}

		if id, ok := markerIDs[string(name)]; ok {
			d.files.set(id)
		}
		if m := suffixMask(name, &fileSuffixTable); m != 0 {
			d.suffixMask |= m
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dirInfo) target(t string) (string, bool) {
	if t == "" {
		return d.path, true
	}
	if strings.Contains(t, "/") {
		p := filepath.Join(d.path, filepath.FromSlash(t))
		fi, err := os.Lstat(p)
		if err != nil || !fi.IsDir() {
			return "", false
		}
		return p, true
	}
	if id, ok := markerIDs[t]; ok {
		if !d.dirs.has(id) {
			return "", false
		}
		return joinPath(d.path, t), true
	}
	if slices.Contains(d.dirNames, t) {
		return joinPath(d.path, t), true
	}
	return "", false
}

func joinPath(dir, name string) string {
	if strings.HasSuffix(dir, string(os.PathSeparator)) {
		return dir + name
	}
	return dir + string(os.PathSeparator) + name
}

func dirNonEmpty(path string) bool {
	found := false
	_ = readDirRaw(path, func([]byte, bool) bool {
		found = true
		return false
	})
	return found
}

func suffixMask(name []byte, table *[maxSuffixLen][]suffixEntry) uint64 {
	dot := bytes.LastIndexByte(name, '.')
	if dot <= 0 {
		return 0
	}
	ext := name[dot:]
	if len(ext) >= maxSuffixLen {
		return 0
	}
	var mask uint64
	for _, s := range table[len(ext)] {
		if bytes.EqualFold(ext, s.suffix) {
			mask |= s.bit
		}
	}
	return mask
}
