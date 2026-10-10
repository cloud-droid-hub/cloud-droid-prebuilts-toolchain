// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

type input struct {
	Files []*os.File
	Infos []os.FileInfo
	Paths []string
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
			return fmt.Errorf("路径不能通过符号链接：%s", dir)
		}
		if dir == filepath.Dir(dir) {
			break
		}
	}
	return nil
}

func openFile(name string) (*os.File, os.FileInfo, error) {
	if err := safePath(name); err != nil {
		return nil, nil, err
	}
	before, err := os.Lstat(name)
	if err != nil || !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("输入不是普通文件：%s err=%v", name, err)
	}
	file, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil || !os.SameFile(before, info) {
		file.Close()
		return nil, nil, fmt.Errorf("打开期间输入身份变化：%s", name)
	}
	if info.Mode().Perm() != 0o644 {
		fmt.Fprintf(os.Stderr, "警告：已有文件权限为%04o，建议0644：%s\n", info.Mode().Perm(), name)
	}
	return file, info, nil
}

func openSource(spec plan, name string) (*input, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	paths := []string{name}
	if info.IsDir() {
		if len(spec.Sizes) == 0 {
			return nil, errors.New("此输入没有登记分片")
		}
		paths = nil
		for index := range spec.Sizes {
			paths = append(paths, filepath.Join(name, fmt.Sprintf("%s.part-%03d", spec.Source, index)))
		}
	}
	in := &input{}
	for index, name := range paths {
		file, got, err := openFile(name)
		if err != nil {
			in.close()
			return nil, err
		}
		in.Files = append(in.Files, file)
		in.Infos = append(in.Infos, got)
		in.Paths = append(in.Paths, name)
		if info.IsDir() && got.Size() != spec.Sizes[index] {
			in.close()
			return nil, fmt.Errorf("分片长度不符：%s got=%d want=%d", name, got.Size(), spec.Sizes[index])
		}
	}
	return in, nil
}

func (in *input) reader() io.Reader {
	var list []io.Reader
	for _, file := range in.Files {
		list = append(list, file)
	}
	return io.MultiReader(list...)
}

func (in *input) check() error {
	for index, file := range in.Files {
		old := in.Infos[index]
		info, err := file.Stat()
		path, pathErr := os.Lstat(in.Paths[index])
		if err != nil || pathErr != nil || !os.SameFile(old, info) || !os.SameFile(old, path) ||
			old.Size() != info.Size() || old.ModTime() != info.ModTime() {
			return fmt.Errorf("读取期间输入变化：%s", in.Paths[index])
		}
	}
	return nil
}

func (in *input) close() {
	for _, file := range in.Files {
		file.Close()
	}
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

func newFile(name string) (*os.File, error) {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		if err := file.Chmod(0o644); err != nil {
			fmt.Fprintf(os.Stderr, "警告：设置0644失败：%s：%v\n", name, err)
		}
	}
	return file, err
}

func copyFile(ctx context.Context, in *os.File, name string, size int64) error {
	out, err := newFile(name)
	if err != nil {
		return err
	}
	n, err := io.CopyN(out, ctxRead{ctx: ctx, in: in}, size)
	if err != nil || n != size {
		out.Close()
		return fmt.Errorf("写入分片%s：bytes=%d want=%d err=%v", name, n, size, err)
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
