package main

import "golang.org/x/sys/unix"

func publish(src, dest string) error {
	return unix.Renameat2(unix.AT_FDCWD, src, unix.AT_FDCWD, dest, unix.RENAME_NOREPLACE)
}

func swapCopy(fd int, src, dest string) error {
	return unix.Renameat2(fd, src, fd, dest, unix.RENAME_EXCHANGE)
}
