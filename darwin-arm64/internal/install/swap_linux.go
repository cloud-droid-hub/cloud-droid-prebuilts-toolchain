// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

func swapDirs(left, right string) error {
	return unix.Renameat2(unix.AT_FDCWD, left, unix.AT_FDCWD, right, unix.RENAME_EXCHANGE)
}

func moveNew(from *os.File, name string, root *os.File) error {
	return unix.Renameat2(int(from.Fd()), name, int(root.Fd()), name, unix.RENAME_NOREPLACE)
}

func fdPath(file *os.File) (string, error) {
	return os.Readlink("/proc/self/fd/" + strconv.FormatUint(uint64(file.Fd()), 10))
}
