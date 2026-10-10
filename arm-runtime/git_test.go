package main

import (
	"archive/tar"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitSample(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "source.txt"), []byte("tracked source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "source-test"},
		{"config", "user.email", "source@example.invalid"},
		{"add", "source.txt"},
		{"commit", "-q", "-m", "source input"},
		{"tag", "test-source"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", src}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("Git preparation failed: %v %s", err, out)
		}
	}
	return src
}

func TestGitData(t *testing.T) {
	src := gitSample(t)
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	if err := gitData(src, dest, work); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
	if err != nil || string(got) != "tracked source\n" {
		t.Fatalf("Git copy content: %q error=%v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(dest, ".git")); !os.IsNotExist(err) {
		t.Fatal("Git metadata was copied")
	}
	a, err := os.Stat(filepath.Join(src, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(filepath.Join(dest, "source.txt"))
	if err != nil || regular(b) != nil || os.SameFile(a, b) {
		t.Fatal("Git copy is not independent")
	}
}

func TestGitDirty(t *testing.T) {
	src := gitSample(t)
	if err := os.WriteFile(filepath.Join(src, "source.txt"), []byte("changed source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	if err := gitData(src, dest, work); err != nil {
		t.Fatalf("dirty source was rejected: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
	if err != nil || string(got) != "tracked source\n" {
		t.Fatalf("dirty source copy content: %q error=%v", got, err)
	}
}

func TestPlainTree(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "source.txt"), []byte("plain source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "tool.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if workTree(src) {
		t.Fatal("plain copy was reported as a Git work tree")
	}
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	if err := gitData(src, dest, work); err != nil {
		t.Fatalf("plain copy was rejected: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
	if err != nil || string(got) != "plain source\n" {
		t.Fatalf("plain copy content: %q error=%v", got, err)
	}
	info, err := os.Lstat(filepath.Join(dest, "sub", "tool.sh"))
	if err != nil || regular(info) != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("plain copy mode: %v error=%v", info, err)
	}
	origin, err := os.Stat(filepath.Join(src, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.Stat(filepath.Join(dest, "source.txt"))
	if err != nil || os.SameFile(origin, copied) {
		t.Fatalf("plain copy is not independent: error=%v", err)
	}
}

// 纯复制源码中指向源码根内的已跟踪符号链接，应得到与Git来源相同的输出。
func TestPlainAlias(t *testing.T) {
	src := gitSample(t)
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../source.txt", filepath.Join(src, "sub", "LICENSE")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "sub/LICENSE"}, {"commit", "-q", "-m", "alias input"}} {
		if out, err := exec.Command("git", append([]string{"-C", src}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("Git preparation failed: %v %s", err, out)
		}
	}
	work := t.TempDir()
	want := filepath.Join(work, "git")
	if err := gitData(src, want, work); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(src, ".git")); err != nil {
		t.Fatal(err)
	}
	got := filepath.Join(work, "plain")
	if err := gitData(src, got, work); err != nil {
		t.Fatalf("plain source alias was rejected: %v", err)
	}
	for _, name := range []string{"source.txt", "sub/LICENSE"} {
		base, err := os.ReadFile(filepath.Join(want, name))
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(filepath.Join(got, name))
		if err != nil || regular(info) != nil {
			t.Fatalf("plain alias output is not a regular file: %s error=%v", name, err)
		}
		data, err := os.ReadFile(filepath.Join(got, name))
		if err != nil || !bytes.Equal(data, base) {
			t.Fatalf("plain alias output differs from Git source: %s %q error=%v", name, data, err)
		}
	}
}

// 纯复制源码中越出源码根的符号链接必须拒绝，且不产生输出。
func TestPlainEscape(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "outside.txt"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "source.txt"), []byte("plain source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../outside.txt", filepath.Join(src, "leak")); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	err := gitData(src, dest, work)
	if err == nil {
		t.Fatal("plain source alias escaping the root was accepted")
	}
	t.Logf("rejected: %v", err)
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Fatal("rejected plain source created its output")
	}
}

func TestGitHidden(t *testing.T) {
	src := gitSample(t)
	if out, err := exec.Command("git", "-C", src, "update-index", "--skip-worktree", "source.txt").CombinedOutput(); err != nil {
		t.Fatalf("Git preparation failed: %v %s", err, out)
	}
	work := t.TempDir()
	if err := gitData(src, filepath.Join(work, "copy"), work); err == nil {
		t.Fatal("hidden source index flag was accepted")
	}
}

func TestRawAlias(t *testing.T) {
	tree := testTree(t)
	tree.raw = true
	data := sampleTar(t, []sample{
		{"src/header.h", tar.TypeReg, "", "header", 0o644},
		{"include", tar.TypeSymlink, "src", "", 0o777},
	})
	if err := tree.readTar(bytes.NewReader(data), func(string) bool { return true }); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "copy")
	if err := tree.emit(dest, ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "include", "header.h"))
	if err != nil || string(got) != "header" {
		t.Fatalf("raw directory alias content: %q error=%v", got, err)
	}
	bad := sampleTar(t, []sample{{"src/header.h", tar.TypeReg, "", "alter!", 0o644}})
	if err := tree.readTar(bytes.NewReader(bad), func(string) bool { return true }); err == nil {
		t.Fatal("conflicting raw archive bytes were accepted")
	}
}

func TestGitSelect(t *testing.T) {
	src := gitSample(t)
	if err := os.WriteFile(filepath.Join(src, "unused.txt"), []byte("unused\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "unused.txt"}, {"commit", "-q", "-m", "additional input"}} {
		if out, err := exec.Command("git", append([]string{"-C", src}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("Git preparation failed: %v %s", err, out)
		}
	}
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	if err := run([]string{"git", "--src", src, "--dest", dest, "--work", work, "--path", "source.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dest, "unused.txt")); !os.IsNotExist(err) {
		t.Fatal("unselected source was copied")
	}
	got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
	if err != nil || string(got) != "tracked source\n" {
		t.Fatalf("selected source content: %q error=%v", got, err)
	}
}

func TestGitCheck(t *testing.T) {
	src := gitSample(t)
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	if err := gitData(src, dest, work); err != nil {
		t.Fatal(err)
	}
	args := []string{"git-check", "--src", src, "--dest", dest, "--work", work}
	if err := run(args); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dest, "source.txt")
	if err := os.WriteFile(file, []byte("changed source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(args); err == nil {
		t.Fatal("modified cached source was accepted")
	}
	got, err := os.ReadFile(file)
	if err != nil || string(got) != "changed source\n" {
		t.Fatal("verification modified the cached source")
	}
}

func TestGitPath(t *testing.T) {
	src := gitSample(t)
	work := t.TempDir()
	for _, name := range []string{"", ".", "..", "../source.txt", "/source.txt", "src/../source.txt", "*.txt"} {
		dest := filepath.Join(work, "copy")
		if err := gitSelect(src, dest, work, false, []string{name}); err == nil {
			t.Fatalf("invalid source path was accepted: %q", name)
		}
		if _, err := os.Lstat(dest); !os.IsNotExist(err) {
			t.Fatal("invalid selection created its output")
		}
	}
}

func TestGitEnv(t *testing.T) {
	src := gitSample(t)
	other := gitSample(t)
	if err := os.WriteFile(filepath.Join(other, "source.txt"), []byte("different source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	work := t.TempDir()
	dest := filepath.Join(work, "copy")
	if err := gitData(src, dest, work); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
	if err != nil || string(got) != "tracked source\n" {
		t.Fatalf("Git environment redirected source selection: %q, %v", got, err)
	}
}
