package main

import "golang.org/x/sys/unix"

func publish(src, dest string) error {
	return unix.RenameatxNp(unix.AT_FDCWD, src, unix.AT_FDCWD, dest, unix.RENAME_EXCL)
}

func swapCopy(fd int, src, dest string) error {
	return unix.RenameatxNp(fd, src, fd, dest, unix.RENAME_SWAP)
}
