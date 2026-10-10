// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func sameBytes(a, b io.Reader) (bool, error) {
	left := make([]byte, 64<<10)
	right := make([]byte, len(left))
	for {
		n, first := io.ReadFull(a, left)
		m, second := io.ReadFull(b, right)
		if first != nil && first != io.EOF && first != io.ErrUnexpectedEOF {
			return false, first
		}
		if second != nil && second != io.EOF && second != io.ErrUnexpectedEOF {
			return false, second
		}
		if n != m || !bytes.Equal(left[:n], right[:m]) {
			return false, nil
		}
		if first != nil || second != nil {
			return first == second, nil
		}
	}
}

func writePlain(name string, src *io.SectionReader, size int64, mode os.FileMode) error {
	mode = hostMode(mode)
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	n, err := io.Copy(file, src)
	if err != nil || n != size {
		return fmt.Errorf("copy size mismatch: got=%d want=%d error=%v", n, size, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	equal, err := sameBytes(src, file)
	if err != nil || !equal {
		return fmt.Errorf("copy bytes differ: %v", err)
	}
	if err := file.Sync(); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Mode().Perm() != mode {
		return fmt.Errorf("copy file mode changed: %s", name)
	}
	return regular(info)
}

func hostMode(mode os.FileMode) os.FileMode {
	if mode&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

func regular(info os.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() {
		return fmt.Errorf("expected regular file")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return fmt.Errorf("file must have exactly one link")
	}
	return nil
}

func safeDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	base := string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(abs, base), base) {
		if part == "" {
			continue
		}
		base = filepath.Join(base, part)
		info, err := os.Lstat(base)
		if os.IsNotExist(err) {
			made := os.Mkdir(base, 0o755)
			if made != nil && !os.IsExist(made) {
				return made
			}
			info, err = os.Lstat(base)
			if err == nil && made == nil {
				if err := dirMode(base, info); err != nil {
					return err
				}
			}
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("directory is not independent: %s", base)
		}
	}
	return nil
}

// dirMode仅修改本次新建目录，核对属主及打开前后的原路径身份。
func dirMode(name string, want os.FileInfo) (returnErr error) {
	fd, err := unix.Open(name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), name)
	defer func() { returnErr = errors.Join(returnErr, file.Close()) }()
	opened, err := file.Stat()
	named, nameErr := os.Lstat(name)
	if err != nil || nameErr != nil || !ownDir(opened) || !ownDir(named) ||
		!os.SameFile(want, opened) || !os.SameFile(want, named) {
		return errors.Join(fmt.Errorf("new directory identity changed: %s", name), err, nameErr)
	}
	if err := file.Chmod(0o755); err != nil {
		return err
	}
	opened, err = file.Stat()
	named, nameErr = os.Lstat(name)
	if err != nil || nameErr != nil || !ownDir(opened) || !ownDir(named) ||
		!os.SameFile(want, opened) || !os.SameFile(want, named) ||
		opened.Mode().Perm() != 0o755 || named.Mode().Perm() != 0o755 {
		return errors.Join(fmt.Errorf("new directory mode or identity changed: %s", name), err, nameErr)
	}
	return nil
}

func ownDir(info os.FileInfo) bool {
	if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

func openData(name string) (*os.File, error) {
	if err := safeDir(filepath.Dir(name)); err != nil {
		return nil, err
	}
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if err := regular(info); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	fd, err := unix.Open(name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	now, err := file.Stat()
	if err != nil || regular(now) != nil || !os.SameFile(info, now) {
		file.Close()
		return nil, fmt.Errorf("input identity changed: %s", name)
	}
	return file, nil
}

func writeData(name string, src io.Reader, size int64, mode os.FileMode, sum [32]byte) error {
	mode = hostMode(mode)
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), src)
	if err != nil {
		return err
	}
	if n != size || string(hash.Sum(nil)) != string(sum[:]) {
		return fmt.Errorf("content size or SHA-256 mismatch")
	}
	if err := file.Sync(); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Mode().Perm() != mode {
		return fmt.Errorf("output file mode changed: %s", name)
	}
	return regular(info)
}

func writeText(name, body string, mode os.FileMode) error {
	return writeData(name, strings.NewReader(body), int64(len(body)), mode, sha256.Sum256([]byte(body)))
}

func space(dir string) error {
	return spaceNeed(dir, 0)
}

// freeOf 读取目录所在文件系统的可用字节数；测试注入固定值，不真正填满磁盘。
var freeOf = diskFree

// warnOut 空间警告的输出目标，默认标准错误。
var warnOut io.Writer = os.Stderr

func diskFree(dir string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

// spaceNeed 读取可用空间；写入need字节后不足50 GiB时只打印警告并继续（#382）。
// 读取失败仍返回错误，由调用方中止。
func spaceNeed(dir string, need uint64) error {
	free, err := freeOf(dir)
	if err != nil {
		return err
	}
	if line := spaceLow(free, need); line != "" {
		fmt.Fprintln(warnOut, line)
	}
	if need == 0 {
		fmt.Printf("free_bytes=%d\n", free)
	}
	return nil
}

// spaceLow 返回可用空间不足50 GiB的警告文本，空间充足时返回空串。
func spaceLow(free, need uint64) string {
	if free >= 50<<30 && need <= free-(50<<30) {
		return ""
	}
	if need == 0 {
		return fmt.Sprintf("警告：可用空间 %d 字节低于50GB下限，继续复制", free)
	}
	return fmt.Sprintf("警告：可用空间 %d 字节写入 %d 字节后低于50GB下限，继续复制", free, need)
}
