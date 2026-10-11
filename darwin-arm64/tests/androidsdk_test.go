// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package tests

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

// 只用本仓Android SDK组件与JDK 21，从AIDL、资源和Java源码构建、对齐、签名并回读一个APK。
func TestAndroidSDK(t *testing.T) {
	spec := catalog.Current()
	if spec.Host != runtime.GOOS+"-"+runtime.GOARCH {
		t.Skip("Android SDK tools need their native host")
	}
	var found []catalog.Archive
	for _, item := range spec.Archives {
		if item.Dir == "sdk37" || item.Dir == "jdk21" {
			found = append(found, item)
		}
	}
	if len(found) != 2 {
		t.Fatalf("Android SDK与JDK归档数=%d", len(found))
	}
	spec.Archives = found
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chmod(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "tools")
	if err := install.Run(context.Background(), filepath.Dir(work), dest, spec); err != nil {
		t.Fatal(err)
	}
	if err := install.Check(context.Background(), dest, spec); err != nil {
		t.Fatal(err)
	}
	jdk := filepath.Join(dest, "jdk21")
	tools := filepath.Join(dest, "sdk37", "build-tools", "37.0.0")
	jar := filepath.Join(dest, "sdk37", "platforms", "android-36", "android.jar")
	env := append(os.Environ(), "JAVA_HOME="+jdk, "PATH="+filepath.Join(jdk, "bin")+":"+os.Getenv("PATH"))
	run := func(name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", filepath.Base(name), args, err, out)
		}
		return string(out)
	}
	src := t.TempDir()
	files := map[string]string{
		"AndroidManifest.xml": `<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="sample.sdk">` +
			`<uses-sdk android:minSdkVersion="26" android:targetSdkVersion="36"/><application android:label="@string/name"/></manifest>`,
		"res/values/strings.xml":      `<resources><string name="name">Sample</string></resources>`,
		"aidl/sample/sdk/IValue.aidl": "package sample.sdk; interface IValue { int value(); }\n",
		"java/sample/sdk/Main.java":   "package sample.sdk; public final class Main { public static int twice(int v) { return v * 2; } }\n",
	}
	for name, body := range files {
		path := filepath.Join(src, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gen := filepath.Join(src, "gen")
	run(filepath.Join(tools, "aidl"), "-I"+filepath.Join(src, "aidl"), "-o", gen, filepath.Join(src, "aidl/sample/sdk/IValue.aidl"))
	run(filepath.Join(tools, "aapt2"), "compile", "--dir", filepath.Join(src, "res"), "-o", filepath.Join(src, "res.zip"))
	unsigned := filepath.Join(src, "unsigned.apk")
	run(filepath.Join(tools, "aapt2"), "link", "-I", jar, "--manifest", filepath.Join(src, "AndroidManifest.xml"),
		"--java", gen, "-o", unsigned, filepath.Join(src, "res.zip"))
	classes := filepath.Join(src, "classes")
	run(filepath.Join(jdk, "bin", "javac"), "--release", "8", "-Xlint:-options", "-classpath", jar, "-d", classes,
		filepath.Join(gen, "sample/sdk/IValue.java"), filepath.Join(gen, "sample/sdk/R.java"),
		filepath.Join(src, "java/sample/sdk/Main.java"))
	var inputs []string
	if err := filepath.Walk(classes, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(path, ".class") {
			inputs = append(inputs, path)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	dex := filepath.Join(src, "dex")
	if err := os.MkdirAll(dex, 0o755); err != nil {
		t.Fatal(err)
	}
	run(filepath.Join(tools, "d8"), append([]string{"--lib", jar, "--min-api", "26", "--output", dex}, inputs...)...)
	addDex(t, unsigned, filepath.Join(dex, "classes.dex"))
	aligned := filepath.Join(src, "aligned.apk")
	run(filepath.Join(tools, "zipalign"), "-f", "4", unsigned, aligned)
	run(filepath.Join(tools, "zipalign"), "-c", "4", aligned)
	keystore := filepath.Join(src, "debug.keystore")
	run(filepath.Join(jdk, "bin", "keytool"), "-genkeypair", "-keystore", keystore, "-storepass", "android",
		"-keypass", "android", "-alias", "sample", "-keyalg", "RSA", "-keysize", "2048", "-validity", "1",
		"-dname", "CN=sample")
	signed := filepath.Join(src, "signed.apk")
	run(filepath.Join(tools, "apksigner"), "sign", "--ks", keystore, "--ks-pass", "pass:android", "--out", signed, aligned)
	run(filepath.Join(tools, "apksigner"), "verify", signed)
	badging := run(filepath.Join(tools, "aapt2"), "dump", "badging", signed)
	if !strings.Contains(badging, "package: name='sample.sdk'") {
		t.Fatalf("签名APK包名回读错误:\n%s", badging)
	}
	if out := run(filepath.Join(tools, "apksigner"), "--version"); strings.TrimSpace(out) == "" {
		t.Fatal("apksigner版本输出为空")
	}
}

// addDex把classes.dex写入APK副本并替换原文件，保留原有条目。
func addDex(t *testing.T, apk, dex string) {
	t.Helper()
	reader, err := zip.OpenReader(apk)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	next := apk + ".next"
	out, err := os.Create(next)
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
	if _, err := io.Copy(entry, data); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(data.Close(), writer.Close(), out.Close()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(next, apk); err != nil {
		t.Fatal(err)
	}
}
