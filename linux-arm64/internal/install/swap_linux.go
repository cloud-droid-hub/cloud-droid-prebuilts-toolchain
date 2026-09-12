// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import "golang.org/x/sys/unix"

func swapDirs(left, right string) error {
	return unix.Renameat2(unix.AT_FDCWD, left, unix.AT_FDCWD, right, unix.RENAME_EXCHANGE)
}
