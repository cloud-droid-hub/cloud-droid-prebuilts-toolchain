// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func TestAOSP15Arc(t *testing.T) {
	spec := catalog.Current()
	if spec.Host != runtime.GOOS+"-"+runtime.GOARCH {
		t.Skip("AOSP tools need their native host")
	}
	var found []catalog.Archive
	for _, item := range spec.Archives {
		if item.Dir == "aosp15" {
			found = append(found, item)
		}
	}
	if len(found) != 1 {
		t.Fatalf("AOSP tools archives=%d", len(found))
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
	input := t.TempDir()
	outDir := filepath.Join(input, "generated")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	aidl := filepath.Join(input, "sample", "tools", "Value.aidl")
	if err := os.MkdirAll(filepath.Dir(aidl), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aidl, []byte("package sample.tools; parcelable Value { int value; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(dest, "aosp15/bin/aidl"), "--lang=ndk", "--structured", "-I"+input, "--out="+outDir, "--header_out="+outDir, aidl)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("AIDL generation: %v\n%s", err, out)
	}
	hidlDir := filepath.Join(input, "hidl", "1.0")
	if err := os.MkdirAll(hidlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hidlDir, "types.hal"), []byte("package sample.tools@1.0; struct Value { uint32_t value; };\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(filepath.Join(dest, "aosp15/bin/hidl-gen"), "-Lc++-headers", "-rsample.tools:"+filepath.Dir(hidlDir), "-o", outDir, "sample.tools@1.0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("HIDL generation: %v\n%s", err, out)
	}
	for _, name := range []string{"aidl/sample/tools/Value.h", "sample/tools/1.0/types.h"} {
		info, err := os.Lstat(filepath.Join(outDir, name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			t.Fatalf("generated header %s: %v", name, err)
		}
		if data, ok := info.Sys().(*syscall.Stat_t); !ok || data.Nlink != 1 {
			t.Fatalf("generated header is not independent: %s", name)
		}
	}
}
