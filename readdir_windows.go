//go:build windows

package main

import (
	"strings"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
)

func readDirRaw(path string, visit func(name []byte, isDir bool) bool) error {
	pattern := longPath(path)
	if !strings.HasSuffix(pattern, `\`) && !strings.HasSuffix(pattern, `/`) {
		pattern += `\`
	}
	pattern += "*"

	p, err := syscall.UTF16PtrFromString(pattern)
	if err != nil {
		return err
	}

	var data syscall.Win32finddata
	h, err := syscall.FindFirstFile(p, &data)
	if err != nil {
		return err
	}
	defer syscall.FindClose(h)

	var scratch [4 * 260]byte
	for {
		name16 := trimNUL(data.FileName[:])
		if len(name16) > 0 && !isDotName16(name16) {
			isDir := data.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY != 0 &&
				data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT == 0
			if !visit(appendUTF8(scratch[:0], name16), isDir) {
				return nil
			}
		}

		if err := syscall.FindNextFile(h, &data); err != nil {
			if err == syscall.ERROR_NO_MORE_FILES {
				return nil
			}
			return err
		}
	}
}

func trimNUL(s []uint16) []uint16 {
	for i, c := range s {
		if c == 0 {
			return s[:i]
		}
	}
	return s
}

func isDotName16(name []uint16) bool {
	if len(name) == 0 || name[0] != '.' {
		return false
	}
	return len(name) == 1 || (len(name) == 2 && name[1] == '.')
}

func appendUTF8(dst []byte, src []uint16) []byte {
	for i := 0; i < len(src); i++ {
		r := rune(src[i])
		if utf16.IsSurrogate(r) {
			if i+1 < len(src) {
				if dec := utf16.DecodeRune(r, rune(src[i+1])); dec != utf8.RuneError {
					dst = utf8.AppendRune(dst, dec)
					i++
					continue
				}
			}
			dst = utf8.AppendRune(dst, utf8.RuneError)
			continue
		}
		dst = utf8.AppendRune(dst, r)
	}
	return dst
}

func longPath(path string) string {
	if len(path) < 248 || strings.HasPrefix(path, `\\`) {
		return path
	}
	return `\\?\` + path
}
