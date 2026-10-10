// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitSet(t *testing.T) {
	left := gitSample(t)
	right := gitSample(t)
	for _, args := range [][]string{{"config", "core.symlinks", "false"}} {
		if out, err := exec.Command("git", append([]string{"-C", left}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("Git configuration: %v %s", err, out)
		}
	}
	file := filepath.Join(left, "alias.h")
	if err := os.WriteFile(file, []byte("../right/source.txt"), 0o644); err != nil {
		t.Fatal(err)
	}
	obj, err := gitRead(left, "hash-object", "-w", "alias.h")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"update-index", "--add", "--cacheinfo", "120000," + obj + ",alias.h"},
		{"commit", "-q", "-m", "registered cross-repository alias"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", left}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("Git alias metadata: %v %s", err, out)
		}
	}
	work := t.TempDir()
	dest := filepath.Join(work, "sources")
	args := []string{"git-set", "--dest", dest, "--work", work,
		"--repo", "left=" + left, "--repo", "right=" + right,
		"--path", "left=alias.h", "--path", "right=source.txt"}
	if err := run(args); err != nil {
		t.Fatal(err)
	}
	first, err := os.Lstat(filepath.Join(dest, "left", "alias.h"))
	if err != nil || regular(first) != nil {
		t.Fatalf("alias output: %v", err)
	}
	second, err := os.Lstat(filepath.Join(dest, "right", "source.txt"))
	if err != nil || regular(second) != nil || os.SameFile(first, second) {
		t.Fatal("cross-repository alias output is not independent")
	}
	data, err := os.ReadFile(filepath.Join(dest, "left", "alias.h"))
	if err != nil || string(data) != "tracked source\n" {
		t.Fatalf("alias bytes: %q, %v", data, err)
	}
	args[0] = "git-set-check"
	if err := run(args); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "right", "source.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(args); err == nil {
		t.Fatal("changed set copy was accepted")
	}
	args[0] = "git-set"
	args[2] = filepath.Join(work, "missing")
	if err := run(args[:7]); err == nil {
		t.Fatal("cross-repository alias without its registered target was accepted")
	}
}

func TestSetPaths(t *testing.T) {
	src := gitSample(t)
	work := t.TempDir()
	dest := filepath.Join(work, "sources")
	for _, args := range [][]string{
		{"--repo", "../outside=" + src},
		{"--repo", "left=" + src, "--repo", "left=" + src},
		{"--repo", "left=" + src, "--repo", "left/sub=" + src},
		{"--repo", "left=" + src, "--path", "right=source.txt"},
		{"--repo", "left=" + src, "--path", "left=../source.txt"},
	} {
		call := append([]string{"git-set", "--work", work, "--dest", dest}, args...)
		if err := run(call); err == nil {
			t.Fatalf("invalid source selection accepted: %v", args)
		}
		if _, err := os.Lstat(dest); !os.IsNotExist(err) {
			t.Fatal("invalid source selection created an output")
		}
	}
}
