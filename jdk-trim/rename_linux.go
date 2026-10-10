// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import "golang.org/x/sys/unix"

func publish(fd int, src, dest string) error {
	return unix.Renameat2(fd, src, fd, dest, unix.RENAME_NOREPLACE)
}
