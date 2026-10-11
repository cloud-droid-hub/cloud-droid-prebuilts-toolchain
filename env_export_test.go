// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package version_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func tcEnvVars(dest, host string) map[string]string {
	android := filepath.Join(dest, "llvm22", "bin")
	for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(marker); err == nil {
			android = filepath.Join(dest, "selected", "bin")
		}
	}
	ld := filepath.Join(dest, "llvm22", "bin", "ld.lld")
	if host == "darwin-arm64" {
		ld = filepath.Join(dest, "llvm22", "bin", "ld64.lld")
	}
	return map[string]string{
		"CDH_TOOLCHAIN_ROOT":  dest,
		"GO_BIN":              filepath.Join(dest, "go126", "bin", "go"),
		"CDH_BASH_BIN":        filepath.Join(dest, "bash53", "bin", "bash"),
		"CDH_PYTHON_BIN":      filepath.Join(dest, "python314", "bin", "python3.14"),
		"GOTOOLCHAIN":         "local",
		"JAVA_HOME":           filepath.Join(dest, "jdk21"),
		"BAZEL_BIN":           filepath.Join(dest, "bazel851", "bin", "bazel"),
		"CDH_RUST_ROOT":       filepath.Join(dest, "rust197"),
		"CARGO_BIN":           filepath.Join(dest, "rust197", "bin", "cargo"),
		"RUSTC":               filepath.Join(dest, "rust197", "bin", "rustc"),
		"RUSTDOC":             filepath.Join(dest, "rust197", "bin", "rustdoc"),
		"ANDROID_NDK_HOME":    filepath.Join(dest, "ndk29"),
		"ANDROID_NDK_ROOT":    filepath.Join(dest, "ndk29"),
		"NDK_HOME":            filepath.Join(dest, "ndk29"),
		"ANDROID_HOME":        filepath.Join(dest, "sdkapp37"),
		"ANDROID_SDK_ROOT":    filepath.Join(dest, "sdkapp37"),
		"CDH_ANDROID_CLANG":   filepath.Join(android, "clang"),
		"CDH_ANDROID_CXX":     filepath.Join(android, "clang++"),
		"CDH_ANDROID_LLD":     filepath.Join(android, "ld.lld"),
		"CDH_ANDROID_READELF": filepath.Join(dest, "llvm22", "bin", "llvm-readelf"),
		"CDH_ANDROID_AR":      filepath.Join(dest, "llvm22", "bin", "llvm-ar"),
		"CDH_HOST_LD":         ld,
	}
}

func tcEnvOutput(t *testing.T, host string, export bool) (string, map[string]string) {
	t.Helper()
	root, repo := tcTree(t)
	bin := filepath.Join(repo, host, "bin", "cdh-toolchain")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "tools with $value `name` space")
	args := []string{filepath.Join(root, "toolchains", "use-prebuilt.sh"),
		"env", "--host", host, "--repo-root", repo, "--dest", dest}
	if export {
		args = append(args, "--export")
	}
	cmd := exec.Command("bash", args...)
	var errs bytes.Buffer
	cmd.Stderr = &errs
	out, err := cmd.Output()
	if err != nil || errs.Len() != 0 {
		t.Fatalf("env export=%t: %v\n%s", export, err, errs.String())
	}
	return string(out), tcEnvVars(dest, host)
}

func tcEnvProbe(t *testing.T, text string, vars map[string]string, export bool) {
	t.Helper()
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	slices.Sort(names)
	probe := `for name in "$@"; do printf "%s=%s\n" "$name" "${!name}"; done`
	script := text + "\nexec bash -c '" + probe + "' env-probe \"$@\""
	args := append([]string{"-c", script, "env-parent"}, names...)
	cmd := exec.Command("bash", args...)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, ok := vars[name]; !ok {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	var want strings.Builder
	for _, name := range names {
		value := ""
		if export {
			value = vars[name]
		}
		want.WriteString(name + "=" + value + "\n")
	}
	got, err := cmd.Output()
	if err != nil {
		t.Fatalf("child environment: %v", err)
	}
	if string(got) != want.String() {
		t.Fatalf("child environment export=%t:\n%s\nwant:\n%s", export, got, want.String())
	}
}

func TestEnvExports(t *testing.T) {
	for _, host := range []string{"darwin-arm64", "linux-arm64", "linux-amd64"} {
		t.Run(host, func(t *testing.T) {
			text, vars := tcEnvOutput(t, host, true)
			tcEnvProbe(t, text, vars, true)
		})
	}
}

func TestEnvPlain(t *testing.T) {
	text, vars := tcEnvOutput(t, "darwin-arm64", false)
	tcEnvProbe(t, text, vars, false)
}

func TestEnvExportBad(t *testing.T) {
	root, repo := tcTree(t)
	tool := filepath.Join(root, "toolchains", "use-prebuilt.sh")
	for _, mode := range []string{"check", "require", "require-env", "install", "revision"} {
		cmd := exec.Command("bash", tool, mode, "--export", "--host", "darwin-arm64",
			"--repo-root", repo, "--dest", filepath.Join(root, "dest"))
		var out, errs bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errs
		err := cmd.Run()
		fail, ok := err.(*exec.ExitError)
		if !ok || fail.ExitCode() != 2 || out.Len() != 0 || !strings.Contains(errs.String(), "只适用于env") {
			t.Fatalf("%s export: %v stdout=%q stderr=%q", mode, err, out.String(), errs.String())
		}
	}
}
