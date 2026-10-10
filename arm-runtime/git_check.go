// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type copyFault struct{ err error }

func (fault *copyFault) Error() string { return fault.err.Error() }
func (fault *copyFault) Unwrap() error { return fault.err }

func (tree *tarTree) check(dest string) (returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = &copyFault{err: returnErr}
		}
	}()
	files, err := tree.outputFiles("")
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(dest)
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
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("source copy directory is invalid: %s", base)
		}
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return err
	}
	defer root.Close()
	seen := 0
	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		item, ok := files[name]
		if !ok {
			return fmt.Errorf("unselected file in source copy: %s", name)
		}
		info, err := root.Lstat(name)
		if err != nil {
			return fmt.Errorf("source copy file is invalid: %s: %w", name, err)
		}
		if err := regular(info); err != nil {
			return fmt.Errorf("source copy file is invalid: %s: %w", name, err)
		}
		if info.Size() != item.size || info.Mode().Perm() != hostMode(item.mode) {
			return fmt.Errorf("source copy file is invalid: %s: size=%d want=%d mode=%#o want=%#o",
				name, info.Size(), item.size, info.Mode().Perm(), hostMode(item.mode))
		}
		file, err := root.Open(name)
		if err != nil {
			return err
		}
		defer file.Close()
		now, err := file.Stat()
		if err != nil || regular(now) != nil || !os.SameFile(info, now) {
			return fmt.Errorf("source copy identity changed: %s", name)
		}
		equal, err := sameBytes(io.NewSectionReader(tree.data, item.off, item.size), file)
		if err != nil || !equal {
			return fmt.Errorf("source copy bytes differ: %s", name)
		}
		after, err := root.Lstat(name)
		if err != nil || regular(after) != nil || !os.SameFile(info, after) {
			return fmt.Errorf("source copy identity changed: %s", name)
		}
		seen++
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(files) {
		return fmt.Errorf("source copy file count mismatch: got=%d want=%d", seen, len(files))
	}
	fmt.Printf("source_copy=%s files=%d bytes_equal=true\n", dest, seen)
	return nil
}
