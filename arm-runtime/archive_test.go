package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

type sample struct {
	name string
	kind byte
	link string
	data string
	mode int64
}

func sampleTar(t *testing.T, list []sample) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, item := range list {
		h := tar.Header{Name: item.name, Typeflag: item.kind, Linkname: item.link, Mode: item.mode}
		if item.kind == tar.TypeReg {
			h.Size = int64(len(item.data))
		}
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if item.kind == tar.TypeReg {
			if _, err := tw.Write([]byte(item.data)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testTree(t *testing.T) *tarTree {
	t.Helper()
	tree, err := newTree(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tree.close(); err != nil {
			t.Error(err)
		}
	})
	return tree
}

func TestRustOnly(t *testing.T) {
	tree := testTree(t)
	data := sampleTar(t, []sample{
		{rustPrefix + "lib/libcore-a.rlib", tar.TypeReg, "", "core", 0o644},
		{rustPrefix + "lib/libstd-b.rlib", tar.TypeReg, "", "std", 0o644},
		{"rust-1.97.1/bin/rustc", tar.TypeReg, "", "never execute", 0o755},
		{"rust-1.97.1/lib/rustlib/aarch64-linux-android/lib/libcore-c.rlib", tar.TypeReg, "", "android", 0o644},
	})
	if err := tree.readTar(bytes.NewReader(data), rustKeep); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "std")
	if err := tree.emit(dest, rustPrefix); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"lib/libcore-a.rlib": "core", "lib/libstd-b.rlib": "std"} {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil || string(got) != want {
			t.Fatalf("%s: got %q, error=%v", name, got, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(dest, "bin")); !os.IsNotExist(err) {
		t.Fatal("compiler directory was extracted")
	}
	if len(tree.items) != 2 {
		t.Fatalf("selected %d entries, want 2", len(tree.items))
	}
}

func TestFlatLib(t *testing.T) {
	tree := testTree(t)
	base := "usr/lib/aarch64-linux-gnu/"
	data := sampleTar(t, []sample{
		{base + "real.so.1", tar.TypeReg, "", "\x7fELFdata", 0o755},
		{base + "lib.so.1", tar.TypeSymlink, "real.so.1", "", 0o777},
		{base + "lib.so", tar.TypeSymlink, "/" + base + "lib.so.1", "", 0o777},
	})
	if err := tree.readTar(bytes.NewReader(data), debKeep); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "root")
	if err := tree.emit(dest, ""); err != nil {
		t.Fatal(err)
	}
	var prior []os.FileInfo
	for _, name := range []string{"real.so.1", "lib.so.1", "lib.so"} {
		file := filepath.Join(dest, base, name)
		info, err := os.Lstat(file)
		if err != nil || regular(info) != nil || info.Mode().Perm() != 0o755 {
			t.Fatalf("not an independent executable regular file: %s", file)
		}
		for _, other := range prior {
			if os.SameFile(info, other) {
				t.Fatal("library aliases share file identity")
			}
		}
		prior = append(prior, info)
		got, err := os.ReadFile(file)
		if err != nil || string(got) != "\x7fELFdata" {
			t.Fatalf("alias bytes differ: %s", name)
		}
	}
}

func TestBadArchive(t *testing.T) {
	for _, item := range []sample{
		{"usr/bin/ignored", tar.TypeLink, "usr/bin/source", "", 0o644},
		{"../escape", tar.TypeReg, "", "bad", 0o644},
		{"/usr/include/bad", tar.TypeReg, "", "bad", 0o644},
	} {
		t.Run(item.name, func(t *testing.T) {
			tree := testTree(t)
			if err := tree.readTar(bytes.NewReader(sampleTar(t, []sample{item})), debKeep); err == nil {
				t.Fatal("invalid archive was accepted")
			}
		})
	}
}

func TestBadTargets(t *testing.T) {
	for _, link := range []string{"../../../outside", "missing", "lib.so", "lib.so/child"} {
		t.Run(link, func(t *testing.T) {
			tree := testTree(t)
			data := sampleTar(t, []sample{{"usr/lib/aarch64-linux-gnu/lib.so", tar.TypeSymlink, link, "", 0o777}})
			if err := tree.readTar(bytes.NewReader(data), debKeep); err != nil {
				t.Fatal(err)
			}
			if err := tree.emit(filepath.Join(t.TempDir(), "root"), ""); err == nil {
				t.Fatal("invalid alias target was accepted")
			}
		})
	}
}

func TestDirAlias(t *testing.T) {
	tree := testTree(t)
	data := sampleTar(t, []sample{
		{"usr/include/api/header.h", tar.TypeReg, "", "header", 0o644},
		{"usr/include/compat", tar.TypeSymlink, "api", "", 0o777},
	})
	if err := tree.readTar(bytes.NewReader(data), debKeep); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "root")
	if err := tree.emit(dest, ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "usr/include/compat/header.h"))
	if err != nil || string(got) != "header" {
		t.Fatalf("directory alias bytes: %q, error=%v", got, err)
	}
}

type linkInfo struct {
	os.FileInfo
}

func (info linkInfo) Sys() any { return &syscall.Stat_t{Nlink: 2} }

func TestMultiLink(t *testing.T) {
	name := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(name, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := regular(linkInfo{info}); err == nil {
		t.Fatal("injected multiple-link metadata was accepted")
	}
}
