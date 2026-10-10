// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func testTar(t *testing.T, heads []tar.Header) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, h := range heads {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write(bytes.Repeat([]byte{0, 1, 0xfe, 0xff}, int(h.Size)/4)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRoundTrip(t *testing.T) {
	heads := []tar.Header{
		{Name: "llvm", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/bin/clang-22", Typeflag: tar.TypeReg, Mode: 0o755, Size: 12},
		{Name: "llvm/bin/clang", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "clang-22"},
		{Name: "llvm/bin/clang++", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "clang"},
		{Name: "llvm/bin/lld", Typeflag: tar.TypeReg, Mode: 0o755, Size: 8},
		{Name: "llvm/bin/ld.lld", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "lld"},
		{Name: "llvm/bin/ld64.lld", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "lld"},
		{Name: "llvm/lib/real.so", Typeflag: tar.TypeReg, Mode: 0o644, Size: 16},
		{Name: "llvm/lib/alias.so", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "real.so"},
		{Name: "llvm/lib/second.so", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "alias.so"},
		{Name: "llvm/lib/hard.so", Typeflag: tar.TypeLink, Mode: 0o777, Linkname: "llvm/lib/second.so"},
		{Name: "llvm/lib/clang", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "real.so"},
		{Name: "llvm/LICENSE.TXT", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4},
	}
	var data bytes.Buffer
	b, err := scan(bytes.NewReader(testTar(t, heads)), "llvm", &data)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeTar(b, bytes.NewReader(data.Bytes()), &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	var after bytes.Buffer
	got, err := scan(bytes.NewReader(out.Bytes()), "llvm", &after)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.names) != len(heads) {
		t.Fatalf("entry count=%d, want %d", len(got.names), len(heads))
	}
	for _, h := range heads {
		e := got.items[h.Name]
		if h.Name == "llvm/bin/clang" || h.Name == "llvm/bin/ld.lld" {
			if e.head.Typeflag != tar.TypeSymlink || e.head.Linkname != h.Linkname {
				t.Errorf("registered link changed: %s", h.Name)
			}
			continue
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		want, err := b.resolve(h.Name)
		if err != nil {
			t.Fatal(err)
		}
		if e.head.Typeflag != tar.TypeReg || e.head.Mode != want.head.Mode || e.head.Size != want.head.Size {
			t.Errorf("wrong regular file metadata: %s", h.Name)
		}
		start, end := int(e.off), int(e.off+e.head.Size)
		if sha256.Sum256(after.Bytes()[start:end]) != want.sum {
			t.Errorf("target bytes changed: %s", h.Name)
		}
	}
}

func TestLinkReject(t *testing.T) {
	tests := []struct {
		name  string
		heads []tar.Header
	}{
		{"escape", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeSymlink, Linkname: "../../x"}}},
		{"absolute", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}}},
		{"missing", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeSymlink, Linkname: "missing"}}},
		{"cycle", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeSymlink, Linkname: "y"}, {Name: "llvm/y", Typeflag: tar.TypeSymlink, Linkname: "x"}}},
		{"directory", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeSymlink, Linkname: "dir"}, {Name: "llvm/dir", Typeflag: tar.TypeDir}}},
		{"parent", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeReg}, {Name: "llvm/x/y", Typeflag: tar.TypeReg}}},
		{"traversal", []tar.Header{{Name: "llvm/../x", Typeflag: tar.TypeReg}}},
		{"duplicate", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeReg}, {Name: "llvm/x", Typeflag: tar.TypeReg}}},
		{"special", []tar.Header{{Name: "llvm/x", Typeflag: tar.TypeFifo}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := scan(bytes.NewReader(testTar(t, tt.heads)), "llvm", nil); err == nil {
				t.Fatal("invalid archive was accepted")
			}
		})
	}
}

type testInfo struct {
	mode  os.FileMode
	links uint16
}

func (i testInfo) Name() string       { return "sample" }
func (i testInfo) Size() int64        { return 1 }
func (i testInfo) Mode() os.FileMode  { return i.mode }
func (i testInfo) ModTime() time.Time { return time.Time{} }
func (i testInfo) IsDir() bool        { return i.mode.IsDir() }
func (i testInfo) Sys() any {
	st := &syscall.Stat_t{}
	// 递增兼容Darwin的uint16字段与Linux的uint64字段，无需反射。
	for n := uint16(0); n < i.links; n++ {
		st.Nlink++
	}
	return st
}

func TestFileKinds(t *testing.T) {
	if err := plain(testInfo{mode: 0o644, links: 1}); err != nil {
		t.Fatal(err)
	}
	for _, info := range []testInfo{{mode: 0o644, links: 2}, {mode: os.ModeSymlink, links: 1}, {mode: os.ModeDir, links: 1}} {
		if plain(info) == nil {
			t.Fatal("invalid file metadata was accepted")
		}
	}
}

func TestNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "data")
	f, err := newFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("keep"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if f, err := newFile(name); err == nil {
		f.Close()
		t.Fatal("existing output was reopened")
	}
	got, err := os.ReadFile(name)
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing output changed: %q, %v", got, err)
	}
	info, err := os.Lstat(name)
	if err != nil || plain(info) != nil {
		t.Fatal("output is not an independent regular file")
	}
}
