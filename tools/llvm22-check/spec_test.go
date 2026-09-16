// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
	"github.com/ulikunitz/xz"
)

func TestPartSets(t *testing.T) {
	for _, test := range []struct {
		name  string
		files []string
		valid bool
	}{
		{"ordered", []string{"llvm.tar.xz.part-001", "llvm.tar.xz.part-000"}, true},
		{"gap", []string{"llvm.tar.xz.part-000", "llvm.tar.xz.part-002"}, false},
		{"mixed", []string{"one.tar.xz.part-000", "two.tar.xz.part-001"}, false},
		{"extra", []string{"llvm.tar.xz.part-000", "extra.txt"}, false},
		{"suffix", []string{"llvm.tar.xz.part-next"}, false},
		{"empty", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range test.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("part"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := partPaths(dir)
			if (err == nil) != test.valid {
				t.Fatalf("part set accepted=%t want=%t: %v", err == nil, test.valid, err)
			}
			if test.valid && (len(got) != 2 || filepath.Base(got[0]) != "llvm.tar.xz.part-000") {
				t.Fatalf("parts are not in sequence: %v", got)
			}
		})
	}
}

func tarBytes(t *testing.T, heads []tar.Header, body string) []byte {
	t.Helper()
	var data bytes.Buffer
	w := tar.NewWriter(&data)
	for _, head := range heads {
		if head.Typeflag == tar.TypeReg {
			head.Size = int64(len(body))
		}
		if err := w.WriteHeader(&head); err != nil {
			t.Fatal(err)
		}
		if head.Typeflag == tar.TypeReg {
			if _, err := io.WriteString(w, body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestTarTypes(t *testing.T) {
	for _, test := range []struct {
		name  string
		heads []tar.Header
		valid bool
	}{
		{"plain", []tar.Header{{Name: "llvm/LICENSE", Typeflag: tar.TypeReg, Mode: 0o644}}, true},
		{"clang", []tar.Header{{Name: "llvm/bin/clang-22", Typeflag: tar.TypeReg, Mode: 0o755}, {Name: "llvm/bin/clang", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "clang-22"}}, true},
		{"lld", []tar.Header{{Name: "llvm/bin/lld", Typeflag: tar.TypeReg, Mode: 0o755}, {Name: "llvm/bin/ld.lld", Typeflag: tar.TypeSymlink, Mode: 0o777, Linkname: "lld"}}, true},
		{"alias", []tar.Header{{Name: "llvm/bin/clang++", Typeflag: tar.TypeSymlink, Linkname: "clang"}}, false},
		{"hard", []tar.Header{{Name: "llvm/bin/copy", Typeflag: tar.TypeLink, Linkname: "llvm/bin/clang"}}, false},
		{"missing", []tar.Header{{Name: "llvm/bin/clang", Typeflag: tar.TypeSymlink, Linkname: "clang-22"}}, false},
		{"directory", []tar.Header{{Name: "llvm/bin/lld", Typeflag: tar.TypeDir}, {Name: "llvm/bin/ld.lld", Typeflag: tar.TypeSymlink, Linkname: "lld"}}, false},
		{"escape", []tar.Header{{Name: "llvm/../outside", Typeflag: tar.TypeReg}}, false},
		{"parent", []tar.Header{{Name: "llvm/bin", Typeflag: tar.TypeReg}, {Name: "llvm/bin/tool", Typeflag: tar.TypeReg}}, false},
		{"duplicate", []tar.Header{{Name: "llvm/bin/tool", Typeflag: tar.TypeReg}, {Name: "llvm/bin/tool", Typeflag: tar.TypeReg}}, false},
		{"special", []tar.Header{{Name: "llvm/pipe", Typeflag: tar.TypeFifo}}, false},
		{"mode", []tar.Header{{Name: "llvm/tool", Typeflag: tar.TypeReg, Mode: 0o4755}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			need, err := scanTar(bytes.NewReader(tarBytes(t, test.heads, "data")), "llvm")
			if (err == nil) != test.valid {
				t.Fatalf("archive accepted=%t want=%t: %v", err == nil, test.valid, err)
			}
			if test.valid && need < 4096 {
				t.Fatalf("archive size was not reserved: %d", need)
			}
		})
	}
}

func TestInstaller(t *testing.T) {
	cfg, err := hostConfig(runtime.GOOS + "-" + runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	payload := filepath.Join(dir, "payload")
	if err := os.Mkdir(payload, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\nprintf '%s\\n' '" + cfg.version + "'\n"
	data := tarBytes(t, []tar.Header{{Name: "llvm/bin/check", Typeflag: tar.TypeReg, Mode: 0o755}}, body)
	var packed bytes.Buffer
	w, err := xz.NewWriter(&packed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	part := filepath.Join(payload, "llvm.tar.xz.part-000")
	if err := os.WriteFile(part, packed.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	pack, err := makeSpec(cfg, payload, "llvm", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(packed.Bytes())
	a := pack.spec.Archives[0]
	if len(pack.spec.Archives) != 1 || len(a.Parts) != 1 || a.SHA256 != hex.EncodeToString(sum[:]) || a.Parts[0].SHA256 != a.SHA256 || a.Parts[0].Size != int64(packed.Len()) {
		t.Fatalf("catalog does not describe actual part bytes: %+v", a)
	}
	// This small package exercises the real installer without a compiler or links.
	a.DataOnly = true
	a.Probes = []catalog.Probe{{Path: "bin/check", Want: []string{cfg.version}, Machine: catalog.Script}}
	pack.spec.Archives[0] = a
	dest := filepath.Join(dir, "install")
	if err := install.RunTo(context.Background(), payload, dest, pack.spec, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := install.Check(context.Background(), dest, pack.spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dest, "llvm22")
	if err := audit(root, false, pack.ids); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "bin", "check"))
	if err != nil || string(got) != body {
		t.Fatalf("installed bytes differ: %q: %v", got, err)
	}
	if err := os.WriteFile(part, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := install.RunTo(context.Background(), payload, filepath.Join(dir, "changed"), pack.spec, io.Discard); err == nil {
		t.Fatal("installer accepted a changed part")
	}
}
