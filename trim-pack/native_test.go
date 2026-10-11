// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"debug/elf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// nativeTree只展开本轮已生成的规范归档，不读取已发布组件。
func nativeTree(t *testing.T, dir, comp, dest string) string {
	t.Helper()
	plan, err := lookup("darwin-arm64", comp)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, comp+"-darwin-arm64")
	body, err := os.ReadFile(filepath.Join(base, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rep report
	if err := json.Unmarshal(body, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Host != plan.Host || rep.Profile != plan.Comp+"-v1" {
		t.Fatalf("消费者报告宿主或策略不符：%s：%s/%s", comp, rep.Host, rep.Profile)
	}
	file, err := os.Open(filepath.Join(base, "payload.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != rep.Bytes {
		t.Fatalf("消费者归档身份不符：%s：%v", comp, err)
	}
	hash := sha256.New()
	gr, err := gzip.NewReader(io.TeeReader(file, hash))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	links := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name, err := cleanName(h.Name)
		if err != nil || name != plan.Prefix && !strings.HasPrefix(name, plan.Prefix+"/") {
			t.Fatalf("消费者归档路径无效：%q：%v", h.Name, err)
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if h.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeSymlink {
			rel := strings.TrimPrefix(name, plan.Prefix+"/")
			if plan.Links[rel] != h.Linkname || h.Linkname == "" {
				t.Fatalf("消费者归档含未登记链接：%s", name)
			}
			links[target] = h.Linkname
			continue
		}
		if h.Typeflag != tar.TypeReg {
			t.Fatalf("消费者归档类型无效：%s type=%d", name, h.Typeflag)
		}
		out, err := newFile(target)
		if err != nil {
			t.Fatal(err)
		}
		n, err := io.Copy(out, tr)
		if err = errors.Join(err, out.Close()); err != nil || n != h.Size {
			t.Fatalf("消费者展开失败：%s：%v", name, err)
		}
		if err := os.Chmod(target, os.FileMode(h.Mode)&0o755); err != nil {
			t.Logf("警告：消费者副本设权：%s：%v", name, err)
		}
	}
	if _, err := io.Copy(io.Discard, gr); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != rep.SHA256 {
		t.Fatalf("消费者归档SHA-256不符：%s got=%s want=%s", comp, got, rep.SHA256)
	}
	for name, link := range links {
		if err := os.Symlink(link, name); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dest, plan.Prefix)
}

func nativeCmd(t *testing.T, work string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), name, args...)
	cmd.Dir = work
	cmd.Env = append(append([]string{}, env...), "TMPDIR="+work, "TMP="+work, "TEMP="+work)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v退出失败：%v\n%s", name, args, err, out)
	}
	t.Logf("%s %v退出0", filepath.Base(name), args)
	return string(out)
}

func nativeWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	file := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := newFile(file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.WriteString(out, body)
	if err = errors.Join(err, out.Close()); err != nil {
		t.Fatal(err)
	}
}

func zipDex(t *testing.T, apk, dex string) {
	t.Helper()
	reader, err := zip.OpenReader(apk)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	out, err := newFile(apk + ".next")
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(out)
	for _, item := range reader.File {
		if err := writer.Copy(item); err != nil {
			t.Fatal(err)
		}
	}
	entry, err := writer.Create("classes.dex")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.Open(dex)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(entry, data)
	if err = errors.Join(err, data.Close(), writer.Close(), out.Close()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(apk+".next", apk); err != nil {
		t.Fatal(err)
	}
}

func nativeAPK(t *testing.T, root, jdk string) {
	t.Helper()
	dir := testDir(t)
	env := append(os.Environ(), "JAVA_HOME="+jdk, "PATH="+filepath.Join(jdk, "bin")+":"+os.Getenv("PATH"),
		"JAVA_TOOL_OPTIONS=-Djava.io.tmpdir="+dir)
	tools := filepath.Join(root, "build-tools", "37.0.0")
	jar := filepath.Join(root, "platforms", "android-36", "android.jar")
	copy := "// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)\n"
	xml := "<!-- Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course) -->\n"
	nativeWrite(t, dir, "AndroidManifest.xml", xml+`<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="sample.trim"><uses-sdk android:minSdkVersion="26" android:targetSdkVersion="36"/><application android:label="@string/name"/></manifest>`)
	nativeWrite(t, dir, "res/values/strings.xml", xml+`<resources><string name="name">Trim</string></resources>`)
	nativeWrite(t, dir, "aidl/sample/trim/IValue.aidl", copy+"package sample.trim; interface IValue { int value(); }\n")
	nativeWrite(t, dir, "java/sample/trim/Main.java", copy+"package sample.trim; public final class Main { public static int twice(int v) { return v * 2; } }\n")
	gen := filepath.Join(dir, "gen")
	nativeCmd(t, dir, env, filepath.Join(tools, "aidl"), "-I"+filepath.Join(dir, "aidl"), "-o", gen, filepath.Join(dir, "aidl/sample/trim/IValue.aidl"))
	nativeCmd(t, dir, env, filepath.Join(tools, "aapt2"), "compile", "--dir", filepath.Join(dir, "res"), "-o", filepath.Join(dir, "res.zip"))
	unsigned := filepath.Join(dir, "unsigned.apk")
	nativeCmd(t, dir, env, filepath.Join(tools, "aapt2"), "link", "-I", jar, "--manifest", filepath.Join(dir, "AndroidManifest.xml"), "--java", gen, "-o", unsigned, filepath.Join(dir, "res.zip"))
	classes := filepath.Join(dir, "classes")
	nativeCmd(t, dir, env, filepath.Join(jdk, "bin", "javac"), "--release", "8", "-classpath", jar, "-d", classes,
		filepath.Join(gen, "sample/trim/IValue.java"), filepath.Join(gen, "sample/trim/R.java"), filepath.Join(dir, "java/sample/trim/Main.java"))
	var inputs []string
	if err := filepath.WalkDir(classes, func(name string, item os.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(name, ".class") {
			inputs = append(inputs, name)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	dex := filepath.Join(dir, "dex")
	if err := os.Mkdir(dex, 0o755); err != nil {
		t.Fatal(err)
	}
	nativeCmd(t, dir, env, filepath.Join(tools, "d8"), append([]string{"--lib", jar, "--min-api", "26", "--output", dex}, inputs...)...)
	zipDex(t, unsigned, filepath.Join(dex, "classes.dex"))
	aligned := filepath.Join(dir, "aligned.apk")
	nativeCmd(t, dir, env, filepath.Join(tools, "zipalign"), "-f", "4", unsigned, aligned)
	nativeCmd(t, dir, env, filepath.Join(tools, "zipalign"), "-c", "4", aligned)
	key := filepath.Join(dir, "debug.keystore")
	nativeCmd(t, dir, env, filepath.Join(jdk, "bin", "keytool"), "-genkeypair", "-keystore", key, "-storepass", "android", "-keypass", "android", "-alias", "sample", "-keyalg", "RSA", "-keysize", "2048", "-validity", "1", "-dname", "CN=sample")
	signed := filepath.Join(dir, "signed.apk")
	nativeCmd(t, dir, env, filepath.Join(tools, "apksigner"), "sign", "--ks", key, "--ks-pass", "pass:android", "--out", signed, aligned)
	nativeCmd(t, dir, env, filepath.Join(tools, "apksigner"), "verify", signed)
	if out := nativeCmd(t, dir, env, filepath.Join(tools, "aapt2"), "dump", "badging", signed); !strings.Contains(out, "package: name='sample.trim'") {
		t.Fatalf("签名APK包名不符：%s", out)
	}
}

// TestNative使用生成结果，检查本机消费者；未提供制品时明确报告跳过。
func TestNative(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("该验收为macOS arm64原生消费者；Linux原生运行须单独执行")
	}
	dir := os.Getenv("CDH_TRIM_NATIVE")
	if dir == "" {
		t.Skip("未提供CDH_TRIM_NATIVE生成制品目录")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	work := testDir(t)
	if err := freeSpace(work); err != nil {
		t.Fatal(err)
	}
	roots := map[string]string{}
	for _, comp := range []string{"llvm22", "ndk29", "sdk37", "sdkapp37", "aosp15"} {
		roots[comp] = nativeTree(t, dir, comp, filepath.Join(work, comp))
	}
	env := os.Environ()
	copy := "// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)\n"
	nativeWrite(t, work, "main.c", copy+"#include <stdio.h>\nint main(void) { puts(\"trim-native\"); return 0; }\n")
	llvm := filepath.Join(roots["llvm22"], "bin")
	if out := nativeCmd(t, work, env, filepath.Join(llvm, "clang"), "--version"); !strings.Contains(out, "clang version 22.") {
		t.Fatal(out)
	}
	if out := nativeCmd(t, work, env, filepath.Join(llvm, "ld64.lld"), "--version"); !strings.Contains(out, "LLD 22.") {
		t.Fatal(out)
	}
	sdk := strings.TrimSpace(nativeCmd(t, work, env, "/usr/bin/xcrun", "--show-sdk-path"))
	nativeCmd(t, work, env, filepath.Join(llvm, "clang"), "--ld-path="+filepath.Join(llvm, "ld64.lld"), "-isysroot", sdk, filepath.Join(work, "main.c"), "-o", filepath.Join(work, "hello"))
	if out := nativeCmd(t, work, env, filepath.Join(work, "hello")); out != "trim-native\n" {
		t.Fatalf("LLVM宿主程序结果不符：%q", out)
	}
	ndk := filepath.Join(roots["ndk29"], "toolchains", "llvm", "prebuilt", "darwin-x86_64")
	if out := nativeCmd(t, work, env, filepath.Join(ndk, "bin", "clang"), "--version"); !strings.Contains(out, "clang version 21.") {
		t.Fatal(out)
	}
	if out := nativeCmd(t, work, env, filepath.Join(ndk, "bin", "ld.lld"), "--version"); !strings.Contains(out, "LLD 21.") {
		t.Fatal(out)
	}
	android := filepath.Join(work, "hello.android")
	nativeCmd(t, work, env, filepath.Join(ndk, "bin", "clang"), "--target=aarch64-linux-android35", "--sysroot="+filepath.Join(ndk, "sysroot"), "--ld-path="+filepath.Join(ndk, "bin", "ld.lld"), filepath.Join(work, "main.c"), "-o", android)
	file, err := elf.Open(android)
	if err != nil {
		t.Fatal(err)
	}
	if file.Machine != elf.EM_AARCH64 {
		t.Fatalf("NDK产物不是AArch64：%s", file.Machine)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("CDH_TOOLCHAIN_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("消费者验收需要绝对路径CDH_TOOLCHAIN_ROOT登记裁剪后的JDK")
	}
	jdk := filepath.Join(root, "jdk21")
	nativeAPK(t, roots["sdk37"], jdk)
	nativeAPK(t, roots["sdkapp37"], jdk)
	for _, version := range []string{"3.22.1", "4.1.2"} {
		bin := filepath.Join(roots["sdkapp37"], "cmake", version, "bin")
		if out := nativeCmd(t, work, env, filepath.Join(bin, "cmake"), "--version"); !strings.Contains(out, "cmake version "+version) {
			t.Fatal(out)
		}
		nativeCmd(t, work, env, filepath.Join(bin, "ninja"), "--version")
	}
	nativeWrite(t, work, "sample/trim/Value.aidl", copy+"package sample.trim; parcelable Value { int value; }\n")
	nativeWrite(t, work, "hidl/1.0/types.hal", copy+"package sample.trim@1.0; struct Value { uint32_t value; };\n")
	out := filepath.Join(work, "generated")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(roots["aosp15"], "bin")
	nativeCmd(t, work, env, filepath.Join(bin, "aidl"), "--lang=ndk", "--structured", "-I"+work, "--out="+out, "--header_out="+out, filepath.Join(work, "sample/trim/Value.aidl"))
	nativeCmd(t, work, env, filepath.Join(bin, "hidl-gen"), "-Lc++-headers", "-rsample.trim:"+filepath.Join(work, "hidl"), "-o", out, "sample.trim@1.0")
	for _, name := range []string{"aidl/sample/trim/Value.h", "sample/trim/1.0/types.h"} {
		info, err := os.Lstat(filepath.Join(out, name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			t.Fatalf("接口生成产物无效：%s：%v", name, err)
		}
	}
}
