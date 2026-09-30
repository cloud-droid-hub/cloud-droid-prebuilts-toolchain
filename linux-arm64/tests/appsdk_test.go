// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package tests

import (
	"context"
	"debug/elf"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

// 从本仓分片安装源码App构建用SDK，核验包描述、原生程序架构与版本输出。
// 本宿主没有登记来源的Build Tools 34.0.0至36.0.0与CMake不在归档内。
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
		"build-tools/37.0.0":     "Pkg.Revision=37.0.0",
		"platforms/android-34":   "AndroidVersion.ApiLevel=34",
		"platforms/android-35":   "AndroidVersion.ApiLevel=35",
		"platforms/android-36":   "AndroidVersion.ApiLevel=36",
		"platforms/android-37.0": "AndroidVersion.ApiLevel=37.0",
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
	for _, kind := range []string{"build-tools", "platforms"} {
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
	if _, err := os.Stat(filepath.Join(root, "cmake")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("本宿主没有登记来源的CMake不应出现:%v", err)
	}
	elfCount := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		file, err := elf.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		elfCount++
		if file.Machine != elf.EM_AARCH64 {
			t.Errorf("原生文件不是arm64:%s:%v", path, file.Machine)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if elfCount != 3 {
		t.Fatalf("原生文件数=%d，应为aapt2、aidl、zipalign", elfCount)
	}
	out, err := exec.Command(filepath.Join(root, "build-tools", "37.0.0", "zipalign")).CombinedOutput()
	if !strings.Contains(string(out), "Zip alignment utility") {
		t.Fatalf("zipalign输出=%q err=%v", out, err)
	}
}
