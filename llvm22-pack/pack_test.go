// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func writeXZ(t *testing.T, name string, data []byte) {
	t.Helper()
	f, err := newFile(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w, err := xz.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
}

func TestPackFiles(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.tar.xz")
	heads := []tar.Header{
		{Name: "llvm", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/bin/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: 8},
		{Name: "llvm/bin/alias", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "tool"},
		{Name: "llvm/LICENSE.TXT", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4},
	}
	writeXZ(t, input, testTar(t, heads))
	dest := filepath.Join(dir, "candidate")
	if err := pack(input, "llvm", dest); err != nil {
		t.Fatal(err)
	}
	var seen []os.FileInfo
	for _, name := range []string{input, filepath.Join(dest, "content.bin"), filepath.Join(dest, "llvm-cdh.tar"), filepath.Join(dest, "entries.tsv")} {
		info, err := os.Lstat(name)
		if err != nil || plain(info) != nil {
			t.Fatalf("input or output is not a regular one-link file: %s: %v", name, err)
		}
		for _, prior := range seen {
			if os.SameFile(prior, info) {
				t.Fatalf("files share an inode: %s", name)
			}
		}
		seen = append(seen, info)
	}
	data, err := os.ReadFile(filepath.Join(dest, "llvm-cdh.tar"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "output.tar.xz")
	writeXZ(t, output, data)
	if err := verify(input, output, "llvm"); err != nil {
		t.Fatal(err)
	}
	index := bytes.Index(data, []byte{0, 1, 0xfe, 0xff})
	if index < 0 {
		t.Fatal("generated archive has no test payload")
	}
	data[index] ^= 0x80
	changed := filepath.Join(dir, "changed.tar.xz")
	writeXZ(t, changed, data)
	if err := verify(input, changed, "llvm"); err == nil || !strings.Contains(err.Error(), "target bytes differ") {
		t.Fatalf("changed regular file bytes were not rejected: %v", err)
	}
}

func TestPackKeeps(t *testing.T) {
	dir := t.TempDir()
	mark := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(mark, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pack(filepath.Join(dir, "missing.tar.xz"), "llvm", dir); !os.IsExist(err) {
		t.Fatalf("existing candidate directory was not rejected: %v", err)
	}
	data, err := os.ReadFile(mark)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing output changed: %q: %v", data, err)
	}
	items, err := os.ReadDir(dir)
	if err != nil || len(items) != 1 {
		t.Fatalf("existing output directory gained files: %v: %v", items, err)
	}
}

func TestStageBytes(t *testing.T) {
	heads := []tar.Header{{Name: "llvm/bin/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: 8}}
	var data bytes.Buffer
	b, err := scan(bytes.NewReader(testTar(t, heads)), "llvm", &data)
	if err != nil {
		t.Fatal(err)
	}
	data.Bytes()[0] ^= 0x80
	err = writeTar(b, bytes.NewReader(data.Bytes()), io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "staged data mismatch") {
		t.Fatalf("changed staged bytes were not rejected: %v", err)
	}
}
