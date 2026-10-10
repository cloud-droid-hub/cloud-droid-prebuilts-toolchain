package main

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDriverArgs(t *testing.T) {
	root := t.TempDir()
	llvm := filepath.Join(root, "native llvm")
	if err := safeDir(filepath.Join(llvm, "bin")); err != nil {
		t.Fatal(err)
	}
	// A native script records the actual argument vector; no target program runs.
	if err := writeText(filepath.Join(llvm, "bin", "clang"), "#!/bin/sh\nprintf '%s\\n' \"$@\"\n", 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "cross")
	sysroot := filepath.Join(root, "target sysroot")
	if err := config(dest, llvm, sysroot); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"clang", "clang++"} {
		for _, phase := range []string{"", "-c", "-E", "-fsyntax-only"} {
			args := []string{"file name.cc", "-o", "test app"}
			if phase != "" {
				args = append(args, phase)
			}
			got, err := exec.Command(filepath.Join(dest, "bin", tool), args...).Output()
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"--target=aarch64-linux-gnu", "--sysroot=" + sysroot, "--gcc-toolchain=" + sysroot + "/usr"}
			if tool == "clang++" {
				want = append(want, "--driver-mode=g++", "--stdlib=libstdc++")
			}
			if phase == "" {
				want = append(want, "--rtlib=libgcc", "--unwindlib=libgcc", "--ld-path="+llvm+"/bin/ld.lld")
			}
			want = append(want, args...)
			if string(got) != strings.Join(want, "\n")+"\n" {
				t.Fatalf("unexpected driver arguments: %q", got)
			}
		}
	}
}

func TestNoReplace(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "existing")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := config(dest, filepath.Join(root, "llvm"), filepath.Join(root, "sysroot")); err == nil {
		t.Fatal("existing destination was replaced")
	}
	after, err := os.Stat(dest)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("existing directory identity changed")
	}
}

func TestBadCopy(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data")
	if err := writeData(file, bytes.NewBufferString("bad"), 4, 0o644, sha256.Sum256([]byte("good"))); err == nil {
		t.Fatal("mismatched content was accepted")
	}
}
