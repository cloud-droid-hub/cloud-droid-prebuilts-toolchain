// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyCLI(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.tar.xz")
	heads := []tar.Header{
		{Name: "llvm", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/lib", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/lib/cdh", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/lib/cdh/libdep.so", Typeflag: tar.TypeReg, Mode: 0o644, Size: 8},
		{Name: "llvm/LICENSE.TXT", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4},
	}
	writeXZ(t, input, testTar(t, heads))
	bin := filepath.Join(dir, "llvm22-pack")
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	if out, err := exec.Command(goBin, "build", "-buildvcs=false", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build command: %v: %s", err, out)
	}
	dest := filepath.Join(dir, "candidate")
	mapping := "lib/libdep.so=lib/cdh/libdep.so"
	if out, err := exec.Command(bin, "pack", input, "llvm", dest, mapping).CombinedOutput(); err != nil {
		t.Fatalf("pack declared copy: %v: %s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(dest, "llvm-cdh.tar"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "output.tar.xz")
	writeXZ(t, output, data)
	if out, err := exec.Command(bin, "verify", input, output, "llvm", mapping).CombinedOutput(); err != nil {
		t.Fatalf("verify declared copy: %v: %s", err, out)
	}
	if out, err := exec.Command(bin, "verify", input, output, "llvm").CombinedOutput(); err == nil {
		t.Fatalf("accepted undeclared archive entry: %s", out)
	}
}

func copyHeads() []tar.Header {
	return []tar.Header{
		{Name: "llvm", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/lib", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/lib/cdh", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "llvm/lib/cdh/dep.so", Typeflag: tar.TypeReg, Mode: 0o644, Size: 8},
		{Name: "llvm/lib/cdh/alias.so", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "dep.so"},
	}
}

func TestCopyData(t *testing.T) {
	var data bytes.Buffer
	b, err := scan(bytes.NewReader(testTar(t, copyHeads())), "llvm", &data)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.addCopies([]string{"lib/dep.so=lib/cdh/alias.so"}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeTar(b, bytes.NewReader(data.Bytes()), &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(bytes.NewReader(out.Bytes()))
	found := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Name != "llvm/lib/dep.so" {
			continue
		}
		found = true
		if h.Typeflag != tar.TypeReg || h.Mode != 0o644 || h.Size != 8 || h.Linkname != "" {
			t.Fatalf("copy is not a regular source-mode file: %+v", h)
		}
		got, err := io.ReadAll(tr)
		want := []byte{0, 1, 0xfe, 0xff, 0, 1, 0xfe, 0xff}
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("copy bytes=%x: %v", got, err)
		}
	}
	if !found {
		t.Fatal("copy entry is absent")
	}
}

func TestCopyReject(t *testing.T) {
	cases := [][]string{
		{"lib/new.so"},
		{"=lib/cdh/dep.so"},
		{"lib/new.so="},
		{"../new.so=lib/cdh/dep.so"},
		{"/lib/new.so=lib/cdh/dep.so"},
		{"lib//new.so=lib/cdh/dep.so"},
		{"lib/new.so/=lib/cdh/dep.so"},
		{"lib/new\n.so=lib/cdh/dep.so"},
		{"lib/new.so=../outside.so"},
		{"lib/new.so=lib/missing.so"},
		{"lib/new.so=lib/cdh"},
		{"lib/cdh/dep.so=lib/cdh/alias.so"},
		{"missing/new.so=lib/cdh/dep.so"},
		{"lib/cdh/dep.so/new.so=lib/cdh/dep.so"},
		{"lib/new.so=lib/cdh/dep.so", "lib/new.so=lib/cdh/dep.so"},
		{"lib/new.so=lib/cdh/dep.so", "lib/next.so=lib/new.so"},
	}
	for _, specs := range cases {
		t.Run(specs[0], func(t *testing.T) {
			b, err := scan(bytes.NewReader(testTar(t, copyHeads())), "llvm", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := b.addCopies(specs); err == nil {
				t.Fatalf("accepted invalid copies: %q", specs)
			}
			if len(b.names) != 5 || len(b.items) != 5 {
				t.Fatal("failed copy request changed archive entries")
			}
		})
	}
}
