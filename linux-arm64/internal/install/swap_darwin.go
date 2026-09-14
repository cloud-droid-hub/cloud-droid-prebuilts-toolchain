// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func swapDirs(left, right string) error {
	return unix.RenamexNp(left, right, unix.RENAME_SWAP)
}

func moveNew(from *os.File, name string, root *os.File) error {
	return unix.RenameatxNp(int(from.Fd()), name, int(root.Fd()), name, unix.RENAME_EXCL)
}

func fdPath(file *os.File) (string, error) {
	buffer := make([]byte, 4096)
	_, _, errno := unix.Syscall(unix.SYS_FCNTL, file.Fd(), uintptr(unix.F_GETPATH), uintptr(unsafe.Pointer(&buffer[0])))
	if errno != 0 {
		return "", errno
	}
	for index, value := range buffer {
		if value == 0 {
			if index == 0 {
				return "", errors.New("file path is empty")
			}
			return string(buffer[:index]), nil
		}
	}
	return "", errors.New("file path is too long")
}
