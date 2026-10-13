// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestCacheParent(t *testing.T) {
	testCacheModes(t, true)
}

func TestCacheMode(t *testing.T) {
	testCacheModes(t, false)
}

func testCacheModes(t *testing.T, parentMode bool) {
	t.Helper()
	modes := []os.FileMode{0o755, 0o775, 0o700, 0o777, 0o775 | os.ModeSetgid, 0o700 | os.ModeSetgid, 0o777 | os.ModeSticky}
	if !parentMode {
		modes = append(modes, 0o555)
	}
	for _, kind := range []string{"git", "plain", "set"} {
		for _, mode := range modes {
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
					changed := dest
					if parentMode {
						changed = parent
					}
					if err := os.Chmod(changed, mode); err != nil {
						t.Fatal(err)
					}
					first, err := os.Lstat(changed)
					if err != nil || first.Mode()&(os.ModePerm|os.ModeSetgid|os.ModeSticky) != mode {
						t.Fatalf("mode setup: %v %v", first, err)
					}
					old, err := os.Lstat(dest)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						names, err := filepath.Glob(filepath.Join(parent, "arm-replaced-*", "source"))
						if err != nil {
							t.Error(err)
							return
						}
						for _, name := range append(names, dest) {
							info, err := os.Lstat(name)
							if err == nil && os.SameFile(old, info) && info.Mode().Perm()&0o200 == 0 {
								if err := os.Chmod(name, 0o755); err != nil {
									t.Error(err)
								}
							}
						}
					})
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
					probe := append([]string(nil), args...)
					probe[0] += "-check"
					if err := run(probe); (!bad && err != nil) || (bad && err == nil) {
						t.Fatalf("cache setup did not match expected validity: invalid=%t error=%v", bad, err)
					}
					args[0] += "-refresh"
					warning, callErr := stderrRun(t, args)
					blocked := !parentMode && mode == 0o555 && bad && errors.Is(callErr, syscall.EACCES)
					if parentMode {
						after, err := os.Lstat(parent)
						if err != nil {
							t.Fatal(err)
						}
						t.Logf("parent=%s mode=%s->%s inode=%d->%d refresh_error=%v", parent, first.Mode(), after.Mode(),
							first.Sys().(*syscall.Stat_t).Ino, after.Sys().(*syscall.Stat_t).Ino, callErr)
						if !os.SameFile(first, after) || first.Mode() != after.Mode() {
							t.Errorf("parent mode or identity changed: before=%s after=%s", first.Mode(), after.Mode())
						}
					}
					if callErr != nil && !blocked {
						t.Fatalf("owned directory rejected: %v", callErr)
					}
					if mode != 0o755 {
						if !strings.Contains(warning, changed) || !strings.Contains(warning, "实际模式="+first.Mode().String()) ||
							!strings.Contains(warning, "建议模式=0755") {
							t.Errorf("mode warning lacks path, actual or suggested mode: %q", warning)
						}
						t.Logf("mode_warning=%s", strings.SplitN(warning, "\n", 2)[0])
					} else if strings.Contains(warning, "警告:") {
						t.Errorf("normal mode reported a warning: %q", warning)
					}
					after, err := os.Lstat(note)
					got, readErr := os.ReadFile(note)
					if err != nil || readErr != nil || !os.SameFile(mark, after) || mark.Mode() != after.Mode() ||
						!mark.ModTime().Equal(after.ModTime()) || string(got) != "keep parent file" {
						t.Fatalf("unrelated parent file changed: %v %v", err, readErr)
					}
					kept, err := filepath.Glob(filepath.Join(parent, "arm-replaced-*", "source"))
					if err != nil || (bad && len(kept) != 1) || (!bad && len(kept) != 0) {
						t.Fatalf("retained cache count differs: %v %v", kept, err)
					}
					name := dest
					if bad && !blocked {
						name = kept[0]
					}
					after, err = os.Lstat(name)
					if err != nil || !os.SameFile(old, after) || old.Mode() != after.Mode() || !old.ModTime().Equal(after.ModTime()) {
						t.Fatalf("old cache root changed: %v", err)
					}
					if !parentMode {
						t.Logf("cache=%s mode=%s->%s inode=%d->%d", name, old.Mode(), after.Mode(),
							old.Sys().(*syscall.Stat_t).Ino, after.Sys().(*syscall.Stat_t).Ino)
					}
					name = filepath.Join(name, rel)
					after, err = os.Lstat(name)
					got, readErr = os.ReadFile(name)
					if err != nil || readErr != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() ||
						!before.ModTime().Equal(after.ModTime()) || !bytes.Equal(data, got) {
						t.Fatalf("old cache file changed: %v %v", err, readErr)
					}
					args[0] = "git-check"
					if kind == "set" {
						args[0] = "git-set-check"
					}
					if blocked {
						t.Logf("read-only cache exchange refused: %v; original=%s candidate=%s", callErr, dest, kept[0])
						fresh, err := os.Lstat(kept[0])
						if err != nil || !ownDir(fresh) || os.SameFile(old, fresh) {
							t.Fatalf("failed exchange did not retain an independent candidate: %v", err)
						}
						for index := 1; index < len(args)-1; index++ {
							if args[index] == "--dest" {
								args[index+1] = kept[0]
								break
							}
						}
					}
					if err := run(args); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestSwapRO(t *testing.T) {
	for _, nested := range []bool{false, true} {
		kind := "same-parent"
		if nested {
			kind = "cross-parent"
		}
		t.Run(kind, func(t *testing.T) {
			nativeSwap(t, nested)
		})
	}
}

func nativeSwap(t *testing.T, nested bool) {
	t.Helper()
	work := t.TempDir()
	newName := "new"
	if nested {
		if err := os.Mkdir(filepath.Join(work, "candidate"), 0o755); err != nil {
			t.Fatal(err)
		}
		newName = filepath.Join("candidate", "new")
	}
	paths := map[string]string{"old": "old", "new": newName}
	dirs := make(map[string]os.FileInfo)
	files := make(map[string]os.FileInfo)
	for _, name := range []string{"old", "new"} {
		path := filepath.Join(work, paths[name])
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(path, "data.txt")
		if err := os.WriteFile(file, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(file)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = info
		if name == "old" {
			if err := os.Chmod(path, 0o555); err != nil {
				t.Fatal(err)
			}
		}
		info, err = os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		dirs[name] = info
	}
	t.Cleanup(func() {
		for _, name := range []string{"old", "new"} {
			path := filepath.Join(work, paths[name])
			info, err := os.Lstat(path)
			if err == nil && os.SameFile(dirs["old"], info) {
				if err := os.Chmod(path, 0o755); err != nil {
					t.Error(err)
				}
			}
		}
	})
	file, err := os.Open(work)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	swapErr := swapCopy(int(file.Fd()), newName, "old")
	t.Logf("actual directory exchange: parent=%s source=%s old_mode=%s old_inode=%d error=%v", work, newName, dirs["old"].Mode(),
		dirs["old"].Sys().(*syscall.Stat_t).Ino, swapErr)
	if swapErr != nil && !errors.Is(swapErr, syscall.EACCES) {
		t.Fatalf("unexpected directory exchange error: %v", swapErr)
	}
	for _, name := range []string{"old", "new"} {
		want := name
		if swapErr == nil {
			if name == "old" {
				want = "new"
			} else {
				want = "old"
			}
		}
		path := filepath.Join(work, paths[name])
		info, err := os.Lstat(path)
		before := dirs[want]
		if err != nil || !os.SameFile(before, info) || before.Mode() != info.Mode() || !before.ModTime().Equal(info.ModTime()) {
			t.Fatalf("directory exchange changed the input: %s %v", path, err)
		}
		path = filepath.Join(path, "data.txt")
		info, err = os.Lstat(path)
		before = files[want]
		data, readErr := os.ReadFile(path)
		if err != nil || readErr != nil || !os.SameFile(before, info) || before.Mode() != info.Mode() ||
			!before.ModTime().Equal(info.ModTime()) || string(data) != want {
			t.Fatalf("directory exchange changed an input file: %s %v %v", path, err, readErr)
		}
	}
}

func TestParentRO(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root身份不能用0555模式建立真实写入拒绝")
	}
	for _, kind := range []string{"git", "plain", "set"} {
		for _, bad := range []bool{false, true} {
			state := "valid"
			if bad {
				state = "rebuild"
			}
			t.Run(kind+"/"+state, func(t *testing.T) {
				parentReadOnly(t, kind, bad)
			})
		}
	}
}

func parentReadOnly(t *testing.T, kind string, bad bool) {
	t.Helper()
	src := gitSample(t)
	if kind == "plain" {
		if err := os.RemoveAll(filepath.Join(src, ".git")); err != nil {
			t.Fatal(err)
		}
	}
	parent := t.TempDir()
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
	name := filepath.Join(dest, rel)
	if bad {
		if err := os.Chmod(name, 0o664); err != nil {
			t.Fatal(err)
		}
	}
	old, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(parent)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		after, err := os.Lstat(parent)
		if err != nil || !os.SameFile(before, after) {
			t.Errorf("cleanup parent identity differs: %v", err)
			return
		}
		if err := os.Chmod(parent, 0o755); err != nil {
			t.Error(err)
		}
	})
	args[0] += "-refresh"
	warning, callErr := stderrRun(t, args)
	if !errors.Is(callErr, syscall.EACCES) || !strings.Contains(callErr.Error(), "mkdir") ||
		!strings.Contains(callErr.Error(), filepath.Base(dest)+".copy.lock") {
		t.Errorf("real parent write failure not preserved: %v", callErr)
	}
	if !strings.Contains(warning, parent) || !strings.Contains(warning, "实际模式="+before.Mode().String()) ||
		!strings.Contains(warning, "建议模式=0755") {
		t.Errorf("read-only mode warning lacks context: %q", warning)
	}
	after, err := os.Lstat(parent)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("parent=%s mode=%s->%s inode=%d->%d write_error=%v", parent, before.Mode(), after.Mode(),
		before.Sys().(*syscall.Stat_t).Ino, after.Sys().(*syscall.Stat_t).Ino, callErr)
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("rejected parent changed: %v", err)
	}
	after, err = os.Lstat(dest)
	if err != nil || !os.SameFile(old, after) || old.Mode() != after.Mode() || !old.ModTime().Equal(after.ModTime()) {
		t.Fatalf("rejected cache root changed: %v", err)
	}
	after, err = os.Lstat(name)
	got, readErr := os.ReadFile(name)
	if err != nil || readErr != nil || !os.SameFile(file, after) || file.Mode() != after.Mode() ||
		!file.ModTime().Equal(after.ModTime()) || string(got) != "tracked source\n" {
		t.Fatalf("rejected cache changed: %v", err)
	}
	items, err := os.ReadDir(parent)
	if err != nil || len(items) != 1 || items[0].Name() != filepath.Base(dest) {
		t.Fatalf("failed write left output in parent: %v %v", items, err)
	}
}

func stderrRun(t *testing.T, args []string) (string, error) {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(t.TempDir(), "stderr.txt"), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := file.Chmod(0o644); err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = file
	defer func() { os.Stderr = saved }()
	callErr := run(args)
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data), callErr
}
