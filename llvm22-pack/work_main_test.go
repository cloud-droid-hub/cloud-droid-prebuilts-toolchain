// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

func TestMain(m *testing.M) { os.Exit(workpath.Run(m)) }

func TestTempCWD(t *testing.T) {
	base, err := workpath.Base()
	if err != nil {
		t.Fatal(err)
	}
	scope, err := workpath.Scope(base)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("TMPDIR")
	want := filepath.Join(scope, "tmp")
	if filepath.Dir(root) != want || !strings.HasPrefix(filepath.Base(root), "go-test-") {
		t.Fatalf("测试临时根未采用工作根:root=%s want=%s", root, want)
	}
	if os.Getenv("GOTMPDIR") != root || os.Getenv("CDH_GO_TEST_ROOT") != root {
		t.Fatal("测试临时变量未绑定本轮私有根")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o755 {
		t.Fatalf("测试根不是0755真实目录:info=%v err=%v", info, err)
	}
	meta, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(meta.Uid) != os.Geteuid() {
		t.Fatalf("测试根不属于本进程:info=%v", info)
	}
	var dir string
	// TempDir内部随后注册的清理先执行，本检查核对数据目录已删除且父根未变。
	t.Cleanup(func() {
		if dir != "" {
			if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("测试数据目录未清理:dir=%s err=%v", dir, err)
			}
		}
		now, err := os.Lstat(root)
		if err != nil || !os.SameFile(info, now) {
			t.Errorf("测试数据清理改变了私有根身份:err=%v", err)
		}
	})
	dir = t.TempDir()
	if !strings.HasPrefix(dir, root+string(os.PathSeparator)) {
		t.Fatalf("测试数据未进入本轮私有根:dir=%s root=%s", dir, root)
	}
	before, err := os.Lstat(dir)
	if err != nil || !before.IsDir() {
		t.Fatalf("测试数据目录不是普通实际目录:info=%v err=%v", before, err)
	}
	path := filepath.Join(dir, "probe")
	if err := os.WriteFile(path, []byte("llvm-temp"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "llvm-temp" {
		t.Fatalf("测试私有数据读写错误:got=%q err=%v", got, err)
	}
	now, err := os.Lstat(dir)
	if err != nil || !os.SameFile(before, now) {
		t.Fatalf("测试数据读写改变了目录身份:err=%v", err)
	}
	t.Logf("私有临时根=%s 数据目录=%s", root, dir)
}
