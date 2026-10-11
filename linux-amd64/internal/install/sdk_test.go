// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package install

import (
	"archive/tar"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
)

func sdkSample(t *testing.T) (catalog.Spec, string, string) {
	t.Helper()
	body := "version=37.0.0\nplatforms=34,35,36,37.0\ncmake=3.22.1,4.1.2\n"
	spec, repo := makeSpec(t, []tarItem{{header: tar.Header{
		Name: "payload/full-sdk.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body)),
	}, data: body}})
	arc := &spec.Archives[0]
	arc.Name, arc.Dir, arc.Version = "Android SDK", "sdkapp37", "37.0.0"
	arc.DataOnly, arc.Probes = true, nil
	arc.Files = []catalog.FileCheck{{Path: "full-sdk.txt", Size: int64(len(body)), SHA256: sumHex([]byte(body)), Want: []string{arc.Version}}}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	arc.Dir = "sdk37"
	return spec, repo, dest
}

func TestSDKRename(t *testing.T) {
	spec, repo, dest := sdkSample(t)
	prior, err := os.Stat(filepath.Join(dest, "sdkapp37"))
	if err != nil {
		t.Fatal(err)
	}
	dropParts(t, repo, spec.Archives[0])
	for range 2 {
		if err := Run(context.Background(), repo, dest, spec); err != nil {
			t.Fatal(err)
		}
		got, err := os.Stat(filepath.Join(dest, "sdk37"))
		if err != nil || !os.SameFile(prior, got) {
			t.Fatalf("完整SDK目录身份改变: %v", err)
		}
		if err := Check(context.Background(), dest, spec); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Lstat(filepath.Join(dest, "sdkapp37")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("旧名称残留: %v", err)
	}
}

func TestSDKLegacy(t *testing.T) {
	spec, repo, dest := sdkSample(t)
	full, err := os.Stat(filepath.Join(dest, "sdkapp37"))
	if err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dest, "sdk37")
	if err := os.Mkdir(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "legacy.txt"), []byte("prior SDK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prior, err := os.Stat(old)
	if err != nil {
		t.Fatal(err)
	}
	dropParts(t, repo, spec.Archives[0])
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]os.FileInfo{"sdk37": full, "sdk37-original": prior} {
		got, err := os.Stat(filepath.Join(dest, name))
		if err != nil || !os.SameFile(want, got) {
			t.Fatalf("SDK目录身份改变:%s %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(dest, "sdk37-original", "legacy.txt"))
	if err != nil || string(data) != "prior SDK\n" {
		t.Fatalf("原目录内容改变:%v %q", err, data)
	}
	if _, err := os.Lstat(filepath.Join(dest, "sdkapp37")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("旧名称残留:%v", err)
	}
}

func TestSDKBlocked(t *testing.T) {
	spec, repo, dest := sdkSample(t)
	want := map[string]os.FileInfo{}
	for _, name := range []string{"sdk37", "sdk37-original", "sdkapp37"} {
		path := filepath.Join(dest, name)
		if name != "sdkapp37" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want[name] = info
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("已有SDK备份被覆盖")
	}
	for name, prior := range want {
		got, err := os.Stat(filepath.Join(dest, name))
		if err != nil || !os.SameFile(prior, got) {
			t.Fatalf("拒绝后目录身份改变:%s %v", name, err)
		}
	}
}

func TestSDKBadAlias(t *testing.T) {
	spec, repo, dest := sdkSample(t)
	path := filepath.Join(dest, "sdkapp37", "full-sdk.txt")
	if err := os.WriteFile(path, []byte("changed SDK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("损坏的完整SDK被迁移")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "changed SDK\n" {
		t.Fatalf("原文件改变:%v %q", err, data)
	}
	if _, err := os.Lstat(filepath.Join(dest, "sdk37")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("失败后产生新SDK目录:%v", err)
	}
}
