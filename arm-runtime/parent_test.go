// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheParent(t *testing.T) {
	for _, kind := range []string{"git", "plain", "set"} {
		for _, mode := range []os.FileMode{0o775, 0o700, 0o775 | os.ModeSetgid} {
			for _, bad := range []bool{false, true} {
				state := "valid"
				if bad {
					state = "rebuild"
				}
				t.Run(kind+"/"+mode.String()+"/"+state, func(t *testing.T) {
					src := gitSample(t)
					if kind == "plain" {
						if err := os.RemoveAll(filepath.Join(src, ".git")); err != nil {
							t.Fatal(err)
						}
					}
					work := t.TempDir()
					parent := filepath.Join(work, "codec2-android15")
					dest := filepath.Join(parent, "source-local")
					args := []string{"git", "--src", src, "--dest", dest}
					rel := "source.txt"
					if kind == "set" {
						args = []string{"git-set", "--repo", "external/test=" + src, "--dest", dest}
						rel = filepath.Join("external", "test", "source.txt")
					}
					if err := run(args); err != nil {
						t.Fatal(err)
					}
					file := filepath.Join(dest, rel)
					if bad {
						if err := os.Chmod(file, 0o664); err != nil {
							t.Fatal(err)
						}
					}
					old, err := os.Lstat(dest)
					if err != nil {
						t.Fatal(err)
					}
					before, err := os.Lstat(file)
					if err != nil {
						t.Fatal(err)
					}
					data, err := os.ReadFile(file)
					if err != nil {
						t.Fatal(err)
					}
					note := filepath.Join(parent, "keep.txt")
					if err := os.WriteFile(note, []byte("keep parent file"), 0o644); err != nil {
						t.Fatal(err)
					}
					mark, err := os.Lstat(note)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(parent, mode); err != nil {
						t.Fatal(err)
					}
					first, err := os.Lstat(parent)
					if err != nil || first.Mode()&(os.ModePerm|os.ModeSetgid) != mode {
						t.Fatalf("parent setup: %v %v", first, err)
					}
					args[0] += "-refresh"
					if err := run(args); err != nil {
						t.Fatalf("owned parent rejected: %v", err)
					}
					after, err := os.Lstat(parent)
					if err != nil || !os.SameFile(first, after) || after.Mode() != os.ModeDir|0o755 {
						t.Fatalf("parent identity or 0755 mode differs: %v %v", after, err)
					}
					after, err = os.Lstat(note)
					got, readErr := os.ReadFile(note)
					if err != nil || readErr != nil || !os.SameFile(mark, after) || string(got) != "keep parent file" {
						t.Fatalf("unrelated parent file changed: %v %v", err, readErr)
					}
					kept, err := filepath.Glob(filepath.Join(parent, "arm-replaced-*", "source"))
					if err != nil || (bad && len(kept) != 1) || (!bad && len(kept) != 0) {
						t.Fatalf("retained cache count differs: %v %v", kept, err)
					}
					name := dest
					if bad {
						name = kept[0]
					}
					after, err = os.Lstat(name)
					if err != nil || !os.SameFile(old, after) || old.Mode() != after.Mode() {
						t.Fatalf("old cache root changed: %v", err)
					}
					name = filepath.Join(name, rel)
					after, err = os.Lstat(name)
					got, readErr = os.ReadFile(name)
					if err != nil || readErr != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || !bytes.Equal(data, got) {
						t.Fatalf("old cache file changed: %v %v", err, readErr)
					}
					args[0] = "git-check"
					if kind == "set" {
						args[0] = "git-set-check"
					}
					if err := run(args); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestParentReject(t *testing.T) {
	src := gitSample(t)
	parent := t.TempDir()
	dest := filepath.Join(parent, "source-local")
	if err := run([]string{"git", "--src", src, "--dest", dest}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"git-refresh", "--src", src, "--dest", dest}); err == nil {
		t.Fatal("parent writable by others accepted")
	}
	after, err := os.Lstat(parent)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
		t.Fatalf("rejected parent changed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
	if err != nil || string(got) != "tracked source\n" {
		t.Fatalf("rejected cache changed: %v", err)
	}
}
