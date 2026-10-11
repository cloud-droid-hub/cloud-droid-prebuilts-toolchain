// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

func TestTreeCWD(t *testing.T) {
	base, tmp := newTestRoot(t)
	old := filepath.Join(base, "old-work")
	if err := os.Mkdir(old, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "keep.txt"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			cwd := filepath.Join(base, name)
			if err := os.Mkdir(cwd, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(cwd)
			t.Setenv("TMPDIR", old)
			work := old
			if name == "second" {
				work = filepath.Join(base, "missing-work")
			}
			tree, err := newTree(work)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := tree.close(); err != nil {
					t.Error(err)
				}
				if _, err := os.Lstat(filepath.Dir(tree.data.Name())); !os.IsNotExist(err) {
					t.Errorf("归档内容工作目录未清理:err=%v", err)
				}
			}()
			file := tree.data.Name()
			want := tmp + string(os.PathSeparator)
			if !strings.HasPrefix(file, want) {
				t.Errorf("归档内容文件未采用统一目标根:file=%s want=%s", file, want)
			}
			srcWorkGone(t, cwd)
			if info, err := os.Lstat(file); err != nil || regular(info) != nil || info.Mode().Perm() != 0o644 {
				t.Errorf("归档内容文件身份或权限错误:info=%v err=%v", info, err)
			}
		})
	}
	if got, err := os.ReadFile(filepath.Join(old, "keep.txt")); err != nil || string(got) != "original" {
		t.Fatalf("旧工作目录标记改变:err=%v", err)
	}
}

func TestRunCWD(t *testing.T) {
	base, tmp := newTestRoot(t)
	old := filepath.Join(base, "old-work")
	if err := os.Mkdir(old, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "keep.txt"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := sampleTar(t, []sample{{name: "hello.txt", kind: tar.TypeReg, data: "content", mode: 0o600}})
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(base, "input.tar.gz")
	if err := os.WriteFile(input, zipped.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(zipped.Bytes())
	for _, name := range []string{"no-flag", "legacy-flag"} {
		t.Run(name, func(t *testing.T) {
			cwd := filepath.Join(base, name)
			if err := os.Mkdir(cwd, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(cwd)
			t.Setenv("TMPDIR", old)
			output := filepath.Join(base, name+"-output")
			args := []string{"source", "--src", input, "--dest", output, "--sha256", fmt.Sprintf("%x", sum)}
			missing := filepath.Join(base, "missing-work")
			if name == "legacy-flag" {
				args = append(args, "--work", missing)
			}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(output, "hello.txt")); err != nil || string(got) != "content" {
				t.Errorf("正式输出内容错误:err=%v", err)
			}
			if _, err := os.Lstat(missing); !os.IsNotExist(err) {
				t.Errorf("旧工作目录被创建:err=%v", err)
			}
			srcWorkGone(t, cwd)
			if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
				t.Errorf("当前目录工作路径未清理:entries=%v err=%v", entries, err)
			}
		})
	}
	if got, err := os.ReadFile(filepath.Join(old, "keep.txt")); err != nil || string(got) != "original" {
		t.Fatalf("继承临时目录标记改变:err=%v", err)
	}
}

func TestTreeSwap(t *testing.T) {
	base, _ := newTestRoot(t)
	cwd := filepath.Join(base, "execution")
	if err := os.Mkdir(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	tree, err := newTree(filepath.Join(base, "missing-work"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tree.data.Write([]byte("original")); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Dir(tree.data.Name())
	moved := filepath.Join(base, "moved")
	if err := os.Rename(directory, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(directory, "keep.txt")
	if err := os.WriteFile(marker, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := tree.close(); err == nil {
		t.Fatal("替换目录清理没有报告身份变化")
	}
	for _, item := range []struct {
		path string
		want string
	}{
		{filepath.Join(moved, "content.bin"), "original"},
		{marker, "replacement"},
	} {
		if got, err := os.ReadFile(item.path); err != nil || string(got) != item.want {
			t.Errorf("目录替换后对象未保留:path=%s err=%v", item.path, err)
		}
	}
}

func TestSourceFail(t *testing.T) {
	base, tmp := newTestRoot(t)
	cwd := filepath.Join(base, "execution")
	if err := os.Mkdir(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	data := sampleTar(t, []sample{{name: "../escape", kind: tar.TypeReg, data: "input", mode: 0o600}})
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(base, "input.tar.gz")
	if err := os.WriteFile(input, zipped.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(zipped.Bytes())
	output := filepath.Join(base, "source-out")
	if err := sourceData(input, output, filepath.Join(base, "missing-work"), fmt.Sprintf("%x", sum)); err == nil {
		t.Fatal("无效归档未被拒绝")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Errorf("失败后出现正式输出:err=%v", err)
	}
	if got, err := os.ReadFile(input); err != nil || !bytes.Equal(got, zipped.Bytes()) {
		t.Errorf("失败后原输入改变:err=%v", err)
	}
	srcWorkGone(t, cwd)
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
		t.Errorf("失败后内容工作目录未清理:entries=%v err=%v", entries, err)
	}
}

func newTestRoot(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	if err := os.Chmod(base, 0o755); err != nil {
		t.Fatal(err)
	}
	mod := []byte("module github.com/cloud-droid-hub/cloud-droid-hub\n\ngo 1.26.0\n")
	if err := os.WriteFile(filepath.Join(base, "go.mod"), mod, 0o644); err != nil {
		t.Fatal(err)
	}
	scope, err := workpath.Scope(base)
	if err != nil {
		t.Fatal(err)
	}
	return base, filepath.Join(scope, "tmp")
}

func srcWorkGone(t *testing.T, cwd string) {
	t.Helper()
	for _, name := range []string{"build", "output", "dist"} {
		path := filepath.Join(cwd, name)
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("源码子目录出现生成内容:path=%s err=%v", path, err)
		}
	}
}
