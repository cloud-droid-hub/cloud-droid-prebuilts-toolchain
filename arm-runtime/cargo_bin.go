package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// lstatFile is the file identity boundary; tests inject link metadata here.
var lstatFile = os.Lstat

// cargoBin publishes the Cargo product behind an uplifted binary as an
// independent regular file. Cargo hard-links release/<bin> to deps/<bin>-<hash>;
// the alias is removed so the real product keeps exactly one link.
func cargoBin(src, dest, work string) error {
	src, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	info, err := lstatFile(src)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cargo product is not a regular file: %s", src)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cargo product has no link metadata: %s", src)
	}
	product := src
	if stat.Nlink != 1 {
		product, err = depsTwin(src, stat)
		if err != nil {
			return err
		}
		if err := os.Remove(src); err != nil {
			return err
		}
		after, err := lstatFile(product)
		if err != nil {
			return err
		}
		if err := regular(after); err != nil {
			return fmt.Errorf("%s: %w", product, err)
		}
	}
	return copyPlain(product, dest)
}

// depsTwin finds the single deps/ file sharing identity with the uplifted alias.
func depsTwin(src string, stat *syscall.Stat_t) (string, error) {
	deps := filepath.Join(filepath.Dir(src), "deps")
	entries, err := os.ReadDir(deps)
	if err != nil {
		return "", err
	}
	var found []string
	for _, entry := range entries {
		name := filepath.Join(deps, entry.Name())
		info, err := lstatFile(name)
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		twin, ok := info.Sys().(*syscall.Stat_t)
		if ok && twin.Dev == stat.Dev && twin.Ino == stat.Ino {
			found = append(found, name)
		}
	}
	if len(found) != 1 {
		return "", fmt.Errorf("cargo product %s must share identity with exactly one deps file, found %d", src, len(found))
	}
	return found[0], nil
}

// copyPlain writes an independent copy of a single-link regular file.
func copyPlain(src, dest string) error {
	file, err := openData(src)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if err := safeDir(filepath.Dir(dest)); err != nil {
		return err
	}
	reader := io.NewSectionReader(file, 0, info.Size())
	return writePlain(dest, reader, info.Size(), info.Mode().Perm())
}
