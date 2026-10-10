// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package version_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func tcPathSetup(t *testing.T, host string) (string, string, string) {
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
	return root, repo, dest
}

func TestToolPaths(t *testing.T) {
	for _, host := range []string{"darwin-arm64", "linux-arm64", "linux-amd64"} {
		t.Run(host, func(t *testing.T) {
			root, repo, dest := tcPathSetup(t, host)
			vars := tcEnvVars(dest, host)
			paths := map[string]string{
				"bash":    vars["CDH_BASH_BIN"],
				"python":  vars["CDH_PYTHON_BIN"],
				"go":      vars["GO_BIN"],
				"java":    filepath.Join(vars["JAVA_HOME"], "bin", "java"),
				"javac":   filepath.Join(vars["JAVA_HOME"], "bin", "javac"),
				"bazel":   vars["BAZEL_BIN"],
				"cargo":   vars["CARGO_BIN"],
				"rustc":   vars["RUSTC"],
				"rustdoc": vars["RUSTDOC"],
				"clang":   vars["CDH_ANDROID_CLANG"],
				"clang++": vars["CDH_ANDROID_CXX"],
				"ld":      vars["CDH_HOST_LD"],
				"lld":     vars["CDH_ANDROID_LLD"],
				"ar":      vars["CDH_ANDROID_AR"],
				"readelf": vars["CDH_ANDROID_READELF"],
			}
			for name, want := range paths {
				t.Run(name, func(t *testing.T) {
					cmd := exec.Command("bash", filepath.Join(root, "tools", "toolchains", "use-prebuilt.sh"),
						"path", "--tool", name, "--host", host, "--repo-root", repo, "--dest", dest)
					var errs bytes.Buffer
					cmd.Stderr = &errs
					got, err := cmd.Output()
					if err != nil || errs.Len() != 0 || string(got) != want+"\n" {
						t.Fatalf("tool %s: %v stdout=%q stderr=%q want=%q", name, err, got, errs.String(), want)
					}
				})
			}
		})
	}
}

func TestPathErrors(t *testing.T) {
	root, repo, dest := tcPathSetup(t, "darwin-arm64")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"path"}, "path需要--tool"},
		{[]string{"path", "--tool"}, "usage:"},
		{[]string{"path", "--tool", "missing"}, "不支持的工具"},
		{[]string{"path", "--tool", "go", "--export"}, "--export只适用于env"},
		{[]string{"env", "--tool", "go"}, "--tool只适用于path"},
		{[]string{"check", "--tool", "go"}, "--tool只适用于path"},
		{[]string{"env", "--tool", ""}, "--tool只适用于path"},
	} {
		args := []string{filepath.Join(root, "tools", "toolchains", "use-prebuilt.sh"), tc.args[0],
			"--host", "darwin-arm64", "--repo-root", repo, "--dest", dest}
		args = append(args, tc.args[1:]...)
		cmd := exec.Command("bash", args...)
		var out, errs bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errs
		err := cmd.Run()
		fail, ok := err.(*exec.ExitError)
		if !ok || fail.ExitCode() != 2 || out.Len() != 0 || !strings.Contains(errs.String(), tc.want) {
			t.Fatalf("args=%q: %v stdout=%q stderr=%q", tc.args, err, out.String(), errs.String())
		}
	}
}

func TestPathPlan(t *testing.T) {
	root, repo, dest := tcPathSetup(t, "darwin-arm64")
	cmd := exec.Command("bash", filepath.Join(root, "tools", "toolchains", "use-prebuilt.sh"),
		"path", "--tool", "go", "--host", "darwin-arm64", "--repo-root", repo, "--dest", dest, "--print-plan")
	got, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	fields := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(got)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("invalid plan field: %q", line)
		}
		fields[key] = value
	}
	if fields["command"] != "path" || fields["tool"] != "go" || fields["host"] != "darwin-arm64" || fields["dest"] != dest {
		t.Fatalf("plan fields: %v", fields)
	}
}
