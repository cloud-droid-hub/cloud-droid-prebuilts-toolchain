// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

const rustPrefix = "rust-1.97.1/lib/rustlib/aarch64-unknown-linux-gnu/"

type tarItem struct {
	kind byte
	link string
	mode os.FileMode
	off  int64
	size int64
	sum  [32]byte
}

type tarTree struct {
	data  *os.File
	dir   string
	root  *os.Root
	drop  func() error
	items map[string]tarItem
	raw   bool
}

func newTree(work string) (*tarTree, error) {
	dir, drop, err := workpath.MkdirTemp("arm-content-")
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, errors.Join(err, drop())
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errors.Join(err, drop())
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.Join(errors.New("归档内容工作目录身份改变"), err, root.Close(), drop())
	}
	data, err := root.OpenFile("content.bin", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err != nil {
		return nil, errors.Join(err, root.Close(), drop())
	}
	if err := data.Chmod(0o644); err != nil {
		return nil, errors.Join(err, data.Close(), root.Close(), drop())
	}
	file, err := data.Stat()
	named, nameErr := root.Lstat("content.bin")
	if err != nil || nameErr != nil || regular(file) != nil || regular(named) != nil ||
		!os.SameFile(file, named) || file.Mode().Perm() != 0o644 || named.Mode().Perm() != 0o644 {
		return nil, errors.Join(errors.New("归档内容工作文件身份或模式改变"), err, nameErr, data.Close(), root.Close(), drop())
	}
	return &tarTree{data: data, dir: dir, root: root, drop: drop, items: make(map[string]tarItem)}, nil
}

func (tree *tarTree) close() error {
	return errors.Join(tree.data.Close(), tree.root.Close(), tree.drop())
}

func tarName(name string) (string, error) {
	if strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\\x00\r\n") {
		return "", fmt.Errorf("invalid archive name %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("archive name escapes root: %q", name)
		}
	}
	return path.Clean(name), nil
}

func rustKeep(name string) bool {
	return strings.HasPrefix(name, rustPrefix+"lib/") || name == rustPrefix+"lib"
}

func debKeep(name string) bool {
	for _, base := range []string{
		"usr/include", "usr/lib/aarch64-linux-gnu", "lib/aarch64-linux-gnu",
		"usr/lib/gcc/aarch64-linux-gnu", "usr/share/pkgconfig", "usr/share/common-licenses",
	} {
		if name == base || strings.HasPrefix(name, base+"/") {
			return true
		}
	}
	return name == "lib/ld-linux-aarch64.so.1" || name == "usr/lib/ld-linux-aarch64.so.1" ||
		(strings.HasPrefix(name, "usr/share/doc/") && path.Base(name) == "copyright")
}

func (tree *tarTree) readTar(src io.Reader, keep func(string) bool) error {
	tr := tar.NewReader(src)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		name, err := tarName(h.Name)
		if err != nil {
			return err
		}
		if h.Typeflag == tar.TypeLink {
			return fmt.Errorf("archive hard link rejected: %s", name)
		}
		if h.Typeflag != tar.TypeDir && h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeSymlink {
			return fmt.Errorf("archive special entry rejected: %s", name)
		}
		if !keep(name) {
			continue
		}
		item := tarItem{kind: h.Typeflag, link: h.Linkname, mode: os.FileMode(h.Mode).Perm(), size: h.Size}
		if item.kind == tar.TypeReg {
			if h.Size > 0 {
				if err := spaceNeed(tree.dir, uint64(h.Size)); err != nil {
					return err
				}
			}
			item.off, err = tree.data.Seek(0, io.SeekEnd)
			if err != nil {
				return err
			}
			var n int64
			if tree.raw {
				n, err = io.Copy(tree.data, tr)
			} else {
				hash := sha256.New()
				n, err = io.Copy(io.MultiWriter(tree.data, hash), tr)
				copy(item.sum[:], hash.Sum(nil))
			}
			if err != nil || n != h.Size {
				return fmt.Errorf("archive content %s: size=%d expected=%d error=%v", name, n, h.Size, err)
			}
		}
		if old, ok := tree.items[name]; ok {
			equal, err := tree.sameItem(old, item)
			if err != nil {
				return err
			}
			if !equal {
				return fmt.Errorf("conflicting archive entry: %s", name)
			}
			continue
		}
		tree.items[name] = item
	}
}

func (tree *tarTree) sameItem(a, b tarItem) (bool, error) {
	if a.kind != b.kind || a.link != b.link || a.mode != b.mode || a.size != b.size {
		return false, nil
	}
	if tree.raw && a.kind == tar.TypeReg {
		return sameBytes(io.NewSectionReader(tree.data, a.off, a.size), io.NewSectionReader(tree.data, b.off, b.size))
	}
	return a.sum == b.sum, nil
}

func linkName(name, target string) (string, error) {
	if target == "" || strings.ContainsAny(target, "\\\x00\r\n") {
		return "", fmt.Errorf("invalid alias at %s", name)
	}
	var dest string
	if strings.HasPrefix(target, "/") {
		dest = path.Clean(strings.TrimPrefix(target, "/"))
	} else {
		dest = path.Clean(path.Join(path.Dir(name), target))
	}
	if dest == ".." || strings.HasPrefix(dest, "../") || path.IsAbs(dest) {
		return "", fmt.Errorf("alias escapes archive root: %s", name)
	}
	return dest, nil
}

func (tree *tarTree) resolve(name string, seen map[string]bool) (string, tarItem, error) {
	if seen[name] {
		return "", tarItem{}, fmt.Errorf("archive alias cycle: %s", name)
	}
	seen[name] = true
	defer delete(seen, name)
	if item, ok := tree.items[name]; ok {
		if item.kind != tar.TypeSymlink {
			return name, item, nil
		}
		dest, err := linkName(name, item.link)
		if err != nil {
			return "", tarItem{}, err
		}
		return tree.resolve(dest, seen)
	}
	for key := range tree.items {
		if strings.HasPrefix(key, name+"/") {
			return name, tarItem{kind: tar.TypeDir}, nil
		}
	}
	for base := path.Dir(name); base != "."; base = path.Dir(base) {
		if item, ok := tree.items[base]; ok && item.kind == tar.TypeSymlink {
			if seen[base] {
				return "", tarItem{}, fmt.Errorf("archive ancestor alias cycle: %s", base)
			}
			seen[base] = true
			defer delete(seen, base)
			dest, err := linkName(base, item.link)
			if err != nil {
				return "", tarItem{}, err
			}
			return tree.resolve(dest+strings.TrimPrefix(name, base), seen)
		}
	}
	return "", tarItem{}, fmt.Errorf("archive alias target missing: %s", name)
}

func (tree *tarTree) gather(src, dest string, files map[string]tarItem, seen map[string]bool) error {
	name, item, err := tree.resolve(src, make(map[string]bool))
	if err != nil {
		return err
	}
	if item.kind == tar.TypeReg {
		if old, ok := files[dest]; ok {
			equal, err := tree.sameItem(old, item)
			if err != nil {
				return err
			}
			if !equal {
				return fmt.Errorf("conflicting output: %s", dest)
			}
		}
		files[dest] = item
		return nil
	}
	if seen[name] {
		return fmt.Errorf("directory alias cycle: %s", name)
	}
	seen[name] = true
	defer delete(seen, name)
	kids := make(map[string]bool)
	for key := range tree.items {
		if tail, ok := strings.CutPrefix(key, name+"/"); ok {
			kids[strings.SplitN(tail, "/", 2)[0]] = true
		}
	}
	for kid := range kids {
		if err := tree.gather(path.Join(name, kid), path.Join(dest, kid), files, seen); err != nil {
			return err
		}
	}
	return nil
}

func (tree *tarTree) outputFiles(prefix string) (map[string]tarItem, error) {
	files := make(map[string]tarItem)
	for name, item := range tree.items {
		if item.kind == tar.TypeDir {
			continue
		}
		out, ok := strings.CutPrefix(name, prefix)
		if !ok || out == "" || out == "." {
			return nil, fmt.Errorf("output prefix mismatch: %s", name)
		}
		if err := tree.gather(name, out, files, make(map[string]bool)); err != nil {
			return nil, err
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no target data selected")
	}
	return files, nil
}

func (tree *tarTree) emit(dest, prefix string) error {
	files, err := tree.outputFiles(prefix)
	if err != nil {
		return err
	}
	if err := safeDir(filepath.Dir(dest)); err != nil {
		return err
	}
	var total int64
	for _, item := range files {
		total += item.size
	}
	if err := spaceNeed(filepath.Dir(dest), uint64(total)); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(dest), "arm-stage-")
	if err != nil {
		return err
	}
	stageInfo, err := os.Lstat(stage)
	if err != nil {
		return err
	}
	if err := dirMode(stage, stageInfo); err != nil {
		return err
	}
	defer func() {
		now, err := os.Lstat(stage)
		if err == nil && os.SameFile(stageInfo, now) {
			_ = os.RemoveAll(stage)
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		item := files[name]
		file := filepath.Join(stage, filepath.FromSlash(name))
		if err := safeDir(filepath.Dir(file)); err != nil {
			return err
		}
		src := io.NewSectionReader(tree.data, item.off, item.size)
		var err error
		if tree.raw {
			err = writePlain(file, src, item.size, item.mode)
		} else {
			err = writeData(file, src, item.size, item.mode, item.sum)
		}
		if err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	now, err := os.Lstat(stage)
	if err != nil || !os.SameFile(stageInfo, now) || !ownDir(now) || now.Mode().Perm() != 0o755 {
		return errors.Join(errors.New("归档输出工作目录身份或模式改变"), err)
	}
	if err := publish(stage, dest); err != nil {
		return err
	}
	fmt.Printf("target=%s files=%d bytes=%d symlinks=0 hardlinks=0\n", dest, len(files), total)
	return nil
}
