// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyRefresh(t *testing.T) {
	for _, kind := range []string{"git", "plain", "set"} {
		for _, bad := range []string{"mode", "bytes", "missing", "extra", "link"} {
			t.Run(kind+"/"+bad, func(t *testing.T) {
				src := gitSample(t)
				if kind == "plain" {
					if err := os.RemoveAll(filepath.Join(src, ".git")); err != nil {
						t.Fatal(err)
					}
				}
				work := t.TempDir()
				dest := filepath.Join(work, "source-local")
				args := []string{"git", "--src", src, "--dest", dest}
				file := filepath.Join(dest, "source.txt")
				if kind == "set" {
					args = []string{"git-set", "--repo", "external/test=" + src, "--dest", dest}
					file = filepath.Join(dest, "external", "test", "source.txt")
				}
				if err := run(args); err != nil {
					t.Fatal(err)
				}
				old, err := os.Lstat(dest)
				if err != nil {
					t.Fatal(err)
				}
				switch bad {
				case "mode":
					err = os.Chmod(file, 0o664)
				case "bytes":
					err = os.WriteFile(file, []byte("changed source\n"), 0o644)
				case "missing":
					err = os.Remove(file)
				case "extra":
					err = os.WriteFile(filepath.Join(dest, "keep.txt"), []byte("original"), 0o644)
				case "link":
					err = os.Link(file, filepath.Join(work, "shared.txt"))
				}
				if err != nil {
					t.Fatal(err)
				}
				before, statErr := os.Lstat(file)
				data, readErr := os.ReadFile(file)
				args[0] += "-check"
				if err := run(args); err == nil {
					t.Fatal("invalid source cache was accepted")
				}
				args[0] = "git-refresh"
				if kind == "set" {
					args[0] = "git-set-refresh"
				}
				if err := run(args); err != nil {
					t.Fatalf("source cache refresh failed: %v", err)
				}
				current, err := os.Lstat(dest)
				if err != nil || os.SameFile(old, current) {
					t.Fatalf("source cache root was not replaced: %v", err)
				}
				info, err := os.Lstat(file)
				if err != nil || regular(info) != nil || info.Mode().Perm() != 0o644 ||
					(statErr == nil && os.SameFile(before, info)) {
					t.Fatalf("new cache is not an independent 0644 file: %v", err)
				}
				got, err := os.ReadFile(file)
				if err != nil || string(got) != "tracked source\n" {
					t.Fatalf("new cache bytes differ: %q %v", got, err)
				}
				kept, err := filepath.Glob(filepath.Join(work, "arm-replaced-*", "source"))
				if err != nil || len(kept) != 1 {
					t.Fatalf("old cache was not retained: %v %v", kept, err)
				}
				info, err = os.Lstat(kept[0])
				if err != nil || !os.SameFile(old, info) {
					t.Fatalf("old cache root identity changed: %v", err)
				}
				rel, err := filepath.Rel(dest, file)
				if err != nil {
					t.Fatal(err)
				}
				name := filepath.Join(kept[0], rel)
				info, err = os.Lstat(name)
				if statErr == nil && (err != nil || !os.SameFile(before, info) || before.Mode() != info.Mode()) {
					t.Fatalf("old cache file changed: %v", err)
				}
				got, err = os.ReadFile(name)
				if readErr == nil && (err != nil || !bytes.Equal(got, data)) {
					t.Fatalf("old cache bytes changed: %v", err)
				}
				if bad == "missing" && !os.IsNotExist(err) {
					t.Fatal("missing old file was added")
				}
				if bad == "extra" {
					got, err = os.ReadFile(filepath.Join(kept[0], "keep.txt"))
					if err != nil || string(got) != "original" {
						t.Fatalf("unselected old file was removed: %v", err)
					}
				}
				args[0] = "git-check"
				if kind == "set" {
					args[0] = "git-set-check"
				}
				if err := run(args); err != nil {
					t.Fatalf("fresh cache check failed: %v", err)
				}
			})
		}
	}
}

func TestRefreshStop(t *testing.T) {
	for _, kind := range []string{"valid", "alias", "locked", "source"} {
		t.Run(kind, func(t *testing.T) {
			src := gitSample(t)
			work := t.TempDir()
			dest := filepath.Join(work, "source-local")
			args := []string{"git", "--src", src, "--dest", dest}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
			old, err := os.Lstat(dest)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "alias":
				moved := filepath.Join(work, "original")
				if err := os.Rename(dest, moved); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(moved, dest); err != nil {
					t.Fatal(err)
				}
			case "locked":
				if err := os.Mkdir(dest+".copy.lock", 0o755); err != nil {
					t.Fatal(err)
				}
			case "source":
				args = append(args, "--path", "absent.txt")
			}
			args[0] = "git-refresh"
			err = run(args)
			if kind == "valid" && err != nil {
				t.Fatal(err)
			}
			if kind != "valid" && err == nil {
				t.Fatal("unsafe cache refresh was accepted")
			}
			info, err := os.Stat(dest)
			if err != nil || !os.SameFile(old, info) {
				t.Fatalf("existing cache root changed: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dest, "source.txt"))
			if err != nil || string(got) != "tracked source\n" {
				t.Fatalf("existing cache bytes changed: %v", err)
			}
			kept, err := filepath.Glob(filepath.Join(work, "arm-replaced-*"))
			if err != nil || len(kept) != 0 {
				t.Fatalf("unchanged cache created replacement directories: %v %v", kept, err)
			}
		})
	}
}
