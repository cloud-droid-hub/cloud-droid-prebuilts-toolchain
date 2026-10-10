package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

type rustPart struct {
	size int64
	sum  string
}

var rustParts = []rustPart{
	{94371840, "fc8e519b876bb680c1f7655dea8f8645007230da6ef0a5c977dc43a4a1e18e4e"},
	{45117112, "f08ce2311eb195faaef59e89cd7726410ca76b4dfd5ba6d0c6ea965d01ad6b3e"},
}

const rustSum = "bab434183ef3e5d5151bafe99e204caee4b28084362ac4e6922d95dc943b23d3"
const rustLicense = "rust-1.97.1/share/doc/rust/COPYRIGHT-library.html"

func rustData(src, dest, work string) (returnErr error) {
	var files []*os.File
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()
	whole := sha256.New()
	var readers []io.Reader
	for n, part := range rustParts {
		name := fmt.Sprintf("rust-1.97.1-linux-arm64.tar.xz.part-%03d", n)
		file, err := openData(filepath.Join(src, "payload", "rust197", name))
		if err != nil {
			return err
		}
		files = append(files, file)
		hash := sha256.New()
		size, err := io.Copy(io.MultiWriter(whole, hash), file)
		if err != nil {
			return err
		}
		if size != part.size || fmt.Sprintf("%x", hash.Sum(nil)) != part.sum {
			return fmt.Errorf("registered Rust part identity mismatch: %s", name)
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		readers = append(readers, file)
		fmt.Printf("input=%s bytes=%d sha256=%s\n", name, size, part.sum)
	}
	if fmt.Sprintf("%x", whole.Sum(nil)) != rustSum {
		return fmt.Errorf("registered Rust archive SHA-256 mismatch")
	}
	reader, err := xz.NewReader(io.MultiReader(readers...))
	if err != nil {
		return err
	}
	tree, err := newTree(work)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, tree.close()) }()
	if err := tree.readTar(reader, func(name string) bool { return rustKeep(name) || name == rustLicense }); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	for _, stem := range []string{"libcore-", "libstd-", "libcompiler_builtins-"} {
		found := false
		for name, item := range tree.items {
			if strings.HasPrefix(name, rustPrefix+"lib/"+stem) && strings.HasSuffix(name, ".rlib") && item.kind == '0' {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("GNU Rust standard library is incomplete: %s", stem)
		}
	}
	license, ok := tree.items[rustLicense]
	if !ok || license.kind != '0' {
		return fmt.Errorf("Rust library copyright data is missing")
	}
	tree.items[rustPrefix+"COPYRIGHT-library.html"] = license
	delete(tree.items, rustLicense)
	fmt.Printf("rust=1.97.1 target=aarch64-unknown-linux-gnu archive_sha256=%s license_sha256=%x\n", rustSum, license.sum)
	return tree.emit(dest, rustPrefix)
}
