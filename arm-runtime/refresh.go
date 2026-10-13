// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func refreshCopy(args []string, dest string) (returnErr error) {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	parent := filepath.Dir(abs)
	if err := safeDir(parent); err != nil {
		return err
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("source cache parent is invalid: %s: %w", parent, err)
	}
	if !ownDir(info) {
		owner := "unavailable"
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			owner = fmt.Sprint(stat.Uid)
		}
		return fmt.Errorf("source cache parent is invalid: %s: mode=%s uid=%s current_uid=%d",
			parent, info.Mode(), owner, os.Geteuid())
	}
	fd, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), parent)
	defer func() { returnErr = errors.Join(returnErr, file.Close()) }()
	opened, err := file.Stat()
	named, nameErr := os.Lstat(parent)
	if err != nil || nameErr != nil || !ownDir(opened) || !ownDir(named) ||
		!os.SameFile(info, opened) || !os.SameFile(info, named) ||
		info.Mode() != opened.Mode() || info.Mode() != named.Mode() {
		return errors.Join(errors.New("source cache parent identity changed"), err, nameErr)
	}
	// 既有父目录只报告模式差异，实际读写仍由文件系统判定。
	if opened.Mode() != os.ModeDir|0o755 {
		fmt.Fprintf(os.Stderr, "警告:源码缓存父目录%s实际模式=%s，建议模式=0755，保留原对象继续\n", parent, opened.Mode())
	}
	opened, err = file.Stat()
	named, nameErr = os.Lstat(parent)
	if err != nil || nameErr != nil || !ownDir(opened) || !ownDir(named) ||
		!os.SameFile(info, opened) || !os.SameFile(info, named) ||
		opened.Mode() != info.Mode() || named.Mode() != info.Mode() {
		return errors.Join(errors.New("source cache parent mode or identity changed"), err, nameErr)
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, root.Close()) }()
	opened, err = root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.Join(errors.New("source cache parent identity changed"), err)
	}
	name := filepath.Base(abs)
	lock := name + ".copy.lock"
	if err := root.Mkdir(lock, 0o755); err != nil {
		return fmt.Errorf("source cache lock is occupied or invalid: %s: %w", lock, err)
	}
	lockInfo, err := root.Lstat(lock)
	if err != nil {
		return err
	}
	defer func() {
		now, err := root.Lstat(lock)
		if err != nil || !os.SameFile(lockInfo, now) {
			returnErr = errors.Join(returnErr, errors.New("source cache lock identity changed"), err)
			return
		}
		returnErr = errors.Join(returnErr, root.Remove(lock))
	}()
	if err := dirMode(filepath.Join(parent, lock), lockInfo); err != nil {
		return err
	}
	old, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return run(args)
	}
	if err != nil || !ownDir(old) {
		return errors.Join(errors.New("source cache must be an owned ordinary directory"), err)
	}
	if old.Mode() != os.ModeDir|0o755 {
		fmt.Fprintf(os.Stderr, "警告:源码缓存目录%s实际模式=%s，建议模式=0755，保留原对象继续\n", abs, old.Mode())
	}
	check := append([]string(nil), args...)
	check[0] += "-check"
	err = run(check)
	if err == nil {
		return nil
	}
	var fault *copyFault
	if !errors.As(err, &fault) {
		return err
	}
	fmt.Fprintf(os.Stderr, "==> source cache check failed: %v\n", err)
	keep, err := os.MkdirTemp(parent, "arm-replaced-")
	if err != nil {
		return err
	}
	kept, err := os.Lstat(keep)
	if err != nil {
		return err
	}
	if err := dirMode(keep, kept); err != nil {
		return err
	}
	candidate := filepath.Join(keep, "source")
	fmt.Fprintf(os.Stderr, "==> source cache candidate retained at: %s\n", candidate)
	copyArgs := append([]string(nil), args...)
	for index := 1; index < len(copyArgs)-1; index++ {
		if copyArgs[index] == "--dest" {
			copyArgs[index+1] = candidate
			break
		}
	}
	if err := run(copyArgs); err != nil {
		return err
	}
	copyArgs[0] += "-check"
	if err := run(copyArgs); err != nil {
		return err
	}
	path := filepath.Join(filepath.Base(keep), "source")
	fresh, err := root.Lstat(path)
	if err != nil || !ownDir(fresh) || fresh.Mode().Perm() != 0o755 {
		return errors.Join(errors.New("source cache candidate identity or mode is invalid"), err)
	}
	before, err := root.Lstat(name)
	now, nameErr := os.Lstat(parent)
	holder, holdErr := root.Lstat(filepath.Base(keep))
	if err != nil || nameErr != nil || holdErr != nil || !os.SameFile(old, before) ||
		!os.SameFile(info, now) || !os.SameFile(kept, holder) {
		return errors.Join(errors.New("source cache identity changed before publication"), err, nameErr, holdErr)
	}
	if err := swapCopy(fd, path, name); err != nil {
		return err
	}
	now, err = root.Lstat(name)
	before, nameErr = root.Lstat(path)
	if err != nil || nameErr != nil || !os.SameFile(fresh, now) || !os.SameFile(old, before) {
		return errors.Join(errors.New("source cache identity changed during publication; both trees retained"), err, nameErr)
	}
	fmt.Printf("source_cache=%s old_cache=%s replaced=true\n", dest, candidate)
	return nil
}
