// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

func TestMain(m *testing.M) { os.Exit(workpath.Run(m)) }

func TestTempCWD(t *testing.T) {
	base, err := workpath.Base()
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("TMPDIR")
	want := filepath.Join(base, "build", "tmp") + string(os.PathSeparator)
	if !strings.HasPrefix(root, want) {
		t.Fatalf("测试临时根未采用工作根:root=%s want=%s", root, want)
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o755 {
		t.Fatalf("测试根不是0755真实目录:info=%v err=%v", info, err)
	}
	dir := t.TempDir()
	if !strings.HasPrefix(dir, root+string(os.PathSeparator)) {
		t.Fatalf("测试数据未进入本轮私有根:dir=%s root=%s", dir, root)
	}
}
