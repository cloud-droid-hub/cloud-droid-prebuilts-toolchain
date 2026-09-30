// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package tests

import (
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

// 从本仓分片安装源码App构建用SDK，核验各组件的包描述、原生程序架构与版本输出。
func TestAppSDK(t *testing.T) {
	spec := catalog.Current()
	if spec.Host != runtime.GOOS+"-"+runtime.GOARCH {
		t.Skip("App SDK tools need their native host")
	}
	var found []catalog.Archive
	for _, item := range spec.Archives {
		if item.Dir == "sdkapp37" {
			found = append(found, item)
		}
	}
	if len(found) != 1 {
		t.Fatalf("源码App构建用SDK归档数=%d", len(found))
	}
	spec.Archives = found
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chmod(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "tools")
	if err := install.Run(context.Background(), filepath.Dir(work), dest, spec); err != nil {
		t.Fatal(err)
	}
	if err := install.Check(context.Background(), dest, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dest, "sdkapp37")
	want := map[string]string{
		"build-tools/34.0.0":     "Pkg.Revision=34.0.0",
		"build-tools/35.0.0":     "Pkg.Revision=35.0.0",
		"build-tools/35.0.1":     "Pkg.Revision=35.0.1",
		"build-tools/36.0.0":     "Pkg.Revision=36.0.0",
		"build-tools/37.0.0":     "Pkg.Revision=37.0.0",
		"platforms/android-34":   "AndroidVersion.ApiLevel=34",
		"platforms/android-35":   "AndroidVersion.ApiLevel=35",
		"platforms/android-36":   "AndroidVersion.ApiLevel=36",
		"platforms/android-37.0": "AndroidVersion.ApiLevel=37.0",
		"cmake/3.22.1":           "Pkg.Revision = 3.22.1",
		"cmake/4.1.2":            "Pkg.Revision = 4.1.2",
	}
	for dir, line := range want {
		data, err := os.ReadFile(filepath.Join(root, dir, "source.properties"))
		if err != nil || !strings.Contains(string(data), line) {
			t.Fatalf("%s包描述不符:%v\n%s", dir, err, data)
		}
		if _, err := os.Stat(filepath.Join(root, dir, "package.xml")); err != nil {
			t.Fatalf("%s缺少package.xml:%v", dir, err)
		}
	}
	for _, kind := range []string{"build-tools", "platforms", "cmake"} {
		entries, err := os.ReadDir(filepath.Join(root, kind))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if _, ok := want[kind+"/"+entry.Name()]; !ok {
				t.Fatalf("存在未登记的组件:%s/%s", kind, entry.Name())
			}
		}
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		file, err := elf.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		if file.Class == elf.ELFCLASS64 && file.Machine != elf.EM_X86_64 {
			t.Errorf("原生文件不是x86-64:%s:%v", path, file.Machine)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(filepath.Join(root, "cmake", "3.22.1", "bin", "ninja"), "--version").Output()
	if err != nil || strings.TrimSpace(string(out)) != "1.10.2" {
		t.Fatalf("ninja 1.10.2输出=%q err=%v", out, err)
	}
}
