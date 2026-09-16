// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const minFree uint64 = 50 << 30
const probeSpace uint64 = 64 << 20

func plain(info os.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() {
		return errors.New("file must be regular")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return errors.New("file must have one link")
	}
	return nil
}

type idSet map[[2]uint64]string

func (ids idSet) add(name string, info os.FileInfo) error {
	if err := plain(info); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	stat := info.Sys().(*syscall.Stat_t)
	id := [2]uint64{uint64(stat.Dev), uint64(stat.Ino)}
	if prior, ok := ids[id]; ok && prior != name {
		return fmt.Errorf("files share an identity: %s and %s", prior, name)
	}
	ids[id] = name
	return nil
}

func realDir(name string) error {
	name, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	for {
		info, err := os.Lstat(name)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("path is not a real directory: %s: %v", name, err)
		}
		parent := filepath.Dir(name)
		if parent == name {
			return nil
		}
		name = parent
	}
}

func openPlain(name string) (*os.File, os.FileInfo, error) {
	if err := realDir(filepath.Dir(name)); err != nil {
		return nil, nil, err
	}
	info, err := os.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if err := plain(info); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", name, err)
	}
	f, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}
	if err := stableFile(f, info); err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, info, nil
}

func stableFile(f *os.File, before os.FileInfo) error {
	current, err := f.Stat()
	if err != nil || plain(current) != nil || !os.SameFile(before, current) ||
		before.Size() != current.Size() || before.Mode() != current.Mode() || before.ModTime() != current.ModTime() {
		return fmt.Errorf("open file changed: %s", f.Name())
	}
	pathInfo, err := os.Lstat(f.Name())
	if err != nil || plain(pathInfo) != nil || !os.SameFile(current, pathInfo) {
		return fmt.Errorf("file path changed: %s", f.Name())
	}
	return nil
}

func room(free, need uint64) error {
	if free < minFree || need > free-minFree {
		return fmt.Errorf("free bytes %d cannot preserve 50 GiB with %d new bytes", free, need)
	}
	return nil
}

func freeSpace(dir string, need uint64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return err
	}
	return room(uint64(stat.Bavail)*uint64(stat.Bsize), need)
}

func absent(name string) error {
	if err := realDir(filepath.Dir(name)); err != nil {
		return err
	}
	_, err := os.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("destination already exists: %s", name)
}

func writeNew(name, body string) error {
	if err := absent(name); err != nil {
		return err
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(f, body)
	syncErr := f.Sync()
	info, statErr := f.Stat()
	closeErr := f.Close()
	return errors.Join(writeErr, syncErr, statErr, plain(info), closeErr)
}

func audit(root string, allow bool, ids idSet) error {
	if err := realDir(root); err != nil {
		return err
	}
	return filepath.WalkDir(root, func(name string, item os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return ids.add(name, info)
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		target, err := os.Readlink(name)
		if err != nil || !allow || !linkOK(filepath.ToSlash(rel), target) {
			return fmt.Errorf("unregistered filesystem link: %s", name)
		}
		if err := realDir(filepath.Dir(name)); err != nil {
			return err
		}
		info, err = os.Lstat(filepath.Join(filepath.Dir(name), target))
		if err != nil || plain(info) != nil {
			return fmt.Errorf("registered link target is not regular: %s", name)
		}
		return nil
	})
}
