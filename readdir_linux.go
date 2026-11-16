//go:build linux

package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

const dirBufSize = 64 << 10

var dirBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, dirBufSize)
		return &b
	},
}

const (
	direntReclenOff = int(unsafe.Offsetof(syscall.Dirent{}.Reclen))
	direntTypeOff   = int(unsafe.Offsetof(syscall.Dirent{}.Type))
	direntNameOff   = int(unsafe.Offsetof(syscall.Dirent{}.Name))
)

func readDirRaw(path string, visit func(name []byte, isDir bool) bool) error {
	fd, err := openDir(path)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	bufp := dirBufPool.Get().(*[]byte)
	defer dirBufPool.Put(bufp)
	buf := *bufp

	for {
		n, err := syscall.Getdents(fd, buf)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if n <= 0 {
			return nil
		}

		for off := 0; off < n; {
			if off+direntNameOff > n {
				return nil
			}
			reclen := int(binary.NativeEndian.Uint16(buf[off+direntReclenOff:]))
			if reclen <= direntNameOff || off+reclen > n {
				return nil
			}
			typ := buf[off+direntTypeOff]
			name := buf[off+direntNameOff : off+reclen]
			off += reclen

			if i := bytes.IndexByte(name, 0); i >= 0 {
				name = name[:i]
			}
			if isDotName(name) {
				continue
			}

			isDir := typ == syscall.DT_DIR
			if typ == syscall.DT_UNKNOWN {
				fi, statErr := os.Lstat(joinPath(path, string(name)))
				if statErr != nil {
					continue
				}
				isDir = fi.IsDir()
			}

			if !visit(name, isDir) {
				return nil
			}
		}
	}
}

func openDir(path string) (int, error) {
	for {
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
		if err == syscall.EINTR {
			continue
		}
		return fd, err
	}
}

func isDotName(name []byte) bool {
	if len(name) == 0 || name[0] != '.' {
		return false
	}
	return len(name) == 1 || (len(name) == 2 && name[1] == '.')
}
