package main

import (
	"archive/tar"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func debData(src, dest, work string) (returnErr error) {
	list, err := filepath.Glob(filepath.Join(src, "*.deb"))
	if err != nil || len(list) == 0 {
		return fmt.Errorf("no Debian packages found in %s", src)
	}
	sort.Strings(list)
	tree, err := newTree(work)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, tree.close()) }()
	for _, name := range list {
		if err := tree.readDeb(name); err != nil {
			return err
		}
	}
	// Ubuntu uses /usr/lib; GNU linker scripts also refer to /lib. Both
	// paths receive separate regular files instead of a merged-/usr link.
	for _, pair := range [][2]string{
		{"lib/aarch64-linux-gnu", "usr/lib/aarch64-linux-gnu"},
		{"lib/ld-linux-aarch64.so.1", "usr/lib/ld-linux-aarch64.so.1"},
	} {
		if _, ok := tree.items[pair[0]]; ok {
			continue
		}
		if _, _, err := tree.resolve(pair[0], make(map[string]bool)); err != nil {
			if _, _, err := tree.resolve(pair[1], make(map[string]bool)); err == nil {
				tree.items[pair[0]] = tarItem{kind: tar.TypeSymlink, link: "/" + pair[1]}
			}
		}
	}
	for _, name := range []string{"usr/include/stdio.h", "usr/lib/aarch64-linux-gnu/crt1.o", "usr/lib/aarch64-linux-gnu/libc.so", "usr/lib/aarch64-linux-gnu/pkgconfig/glib-2.0.pc"} {
		if _, _, err := tree.resolve(name, make(map[string]bool)); err != nil {
			return fmt.Errorf("target sysroot is incomplete: %w", err)
		}
	}
	return tree.emit(dest, "")
}

func (tree *tarTree) readDeb(name string) error {
	file, err := openData(name)
	if err != nil {
		return err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	meta, err := exec.Command("dpkg-deb", "--field", name, "Package", "Architecture", "Version").Output()
	if err != nil {
		return fmt.Errorf("Debian metadata %s: %w", name, err)
	}
	fields := make(map[string]string)
	for _, line := range strings.Split(string(meta), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			fields[key] = strings.TrimSpace(value)
		}
	}
	arch := fields["Architecture"]
	if arch != "arm64" && arch != "all" {
		return fmt.Errorf("unexpected Debian package architecture: %s (%s)", name, arch)
	}
	cmd := exec.Command("dpkg-deb", "--fsys-tarfile", name)
	cmd.Stderr = os.Stderr
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	err = tree.readTar(reader, debKeep)
	if err != nil {
		cmd.Process.Kill()
	} else {
		_, err = io.Copy(io.Discard, reader)
	}
	wait := cmd.Wait()
	if err != nil {
		return fmt.Errorf("Debian package %s: %w", name, err)
	}
	if wait != nil {
		return wait
	}
	after, err := os.Lstat(name)
	if err != nil || regular(after) != nil || !os.SameFile(before, after) || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return fmt.Errorf("Debian package identity changed: %s", name)
	}
	fmt.Printf("package=%s arch=%s version=%s sha256=%x\n", fields["Package"], arch, fields["Version"], hash.Sum(nil))
	return nil
}
