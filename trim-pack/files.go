// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

type ctxRead struct {
	ctx context.Context
	in  io.Reader
}

func (r ctxRead) Read(buf []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.in.Read(buf)
}

func freeSpace(dir string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(dir, &stat); err != nil {
		return err
	}
	if uint64(stat.Bavail)*uint64(stat.Bsize) < 50*1024*1024*1024 {
		return errors.New("可用空间低于50GiB")
	}
	return nil
}

func safePath(name string) error {
	abs, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("输出路径不能通过符号链接：%s", dir)
		}
		if dir == filepath.Dir(dir) {
			return nil
		}
	}
}

func newFile(name string) (*os.File, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		if err := f.Chmod(0o644); err != nil {
			fmt.Fprintf(os.Stderr, "警告：设置0644失败：%s：%v\n", name, err)
		}
	}
	return f, err
}

// snapshot一次读取并固定完整源字节，归档分析只使用已核对的独立快照。
func snapshot(ctx context.Context, in input, root, tmp string) (_ *os.File, ret error) {
	name := filepath.Join(root, in.Name)
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { ret = errors.Join(ret, f.Close()) }()
	old, err := f.Stat()
	if err != nil || !old.Mode().IsRegular() {
		return nil, fmt.Errorf("原始源不是普通文件：%s：%v", in.Name, err)
	}
	if old.Mode().Perm() != 0o644 {
		fmt.Fprintf(os.Stderr, "警告：已有源权限为%04o，建议0644：%s\n", old.Mode().Perm(), name)
	}
	copy, err := newFile(filepath.Join(tmp, "source.bin"))
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			ret = errors.Join(ret, copy.Close())
		}
	}()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(copy, hash), ctxRead{ctx: ctx, in: f})
	if err != nil || n != old.Size() {
		return nil, fmt.Errorf("源读取失败：%s bytes=%d：%v", in.Name, n, err)
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != in.SHA256 {
		return nil, fmt.Errorf("源摘要不符：%s got=%s want=%s", in.Name, got, in.SHA256)
	}
	after, err := f.Stat()
	current, pathErr := os.Stat(name)
	if err != nil || pathErr != nil || !os.SameFile(old, after) || !os.SameFile(old, current) ||
		old.Size() != after.Size() || old.ModTime() != after.ModTime() {
		return nil, fmt.Errorf("读取期间源身份或内容变化：%s", in.Name)
	}
	if _, err := copy.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	ok = true
	return copy, nil
}

func openParent(dest string) (*os.File, error) {
	dir := filepath.Dir(dest)
	if err := safePath(dir); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(dir, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Getuid()) {
		f.Close()
		return nil, errors.New("输出父目录不属于当前用户")
	}
	if info.Mode().Perm() != 0o755 {
		fmt.Fprintf(os.Stderr, "警告：已有输出父目录权限为%04o，建议0755：%s\n", info.Mode().Perm(), dir)
	}
	return f, nil
}
