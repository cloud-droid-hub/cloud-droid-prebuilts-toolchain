// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package version_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBashCLI(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "toolchains", "bash.sh")
	cmd := exec.Command(script, "-c", `printf '%s\n%s\n%s\n' "$BASH_VERSION" "$BASH" "$1"; exit 37`, "probe", "空格 $value `literal`")
	out, err := cmd.CombinedOutput()
	var code int
	if fail, ok := err.(*exec.ExitError); ok {
		code = fail.ExitCode()
	}
	if code != 37 {
		t.Fatalf("Bash退出码：%v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 || lines[0] != "5.3.15(1)-release" || !strings.HasSuffix(lines[1], "/bash53/bin/bash") || lines[2] != "空格 $value `literal`" {
		t.Fatalf("Bash身份或参数不符：%s", out)
	}
}

func TestBashScript(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	script := filepath.Join(work, "probe.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\nset -euo pipefail\ndeclare -A values=([name]=ok)\nprintf '%s|%s|%s' \"${values[name]}\" \"$BASH_VERSION\" \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(root, "toolchains", "bash.sh"), script, "with spaces")
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "ok|5.3.15(1)-release|with spaces" {
		t.Fatalf("脚本执行：%v\n%s", err, out)
	}
}

func TestBashCold(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	host := runtime.GOOS + "-" + runtime.GOARCH
	work := t.TempDir()
	repo := "toolchains/" + host
	for _, name := range []string{"toolchains/bash.sh", repo + "/bin/cdh-toolchain",
		repo + "/payload/bash53/bash-5.3.15-" + host + ".tar.gz.part-000"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(work, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(name, "/bash.sh") || strings.HasSuffix(name, "/cdh-toolchain") {
			mode = 0o755
		}
		if err := os.WriteFile(path, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	dest := filepath.Join(work, "output", "target", "common", "toolchains", host)
	cmd := exec.Command(filepath.Join(work, "toolchains/bash.sh"), "-c",
		`declare -A vals=([ok]=yes); printf '%s|%s|%s' "${vals[ok]}" "$BASH_VERSION" "$BASH"`)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "CDH_TOOLCHAIN_ROOT" && key != "CDH_BASH_BIN" && key != "CDH_BASH_RUN" {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "CDH_TOOLCHAIN_ROOT="+dest)
	var log bytes.Buffer
	cmd.Stderr = &log
	out, err := cmd.Output()
	want := "yes|5.3.15(1)-release|" + filepath.Join(dest, "bash53/bin/bash")
	if err != nil || string(out) != want {
		t.Fatalf("普通源码冷启动:%v:%s:%s", err, out, log.String())
	}
	if _, err := os.Stat(filepath.Join(dest, "go126")); !os.IsNotExist(err) {
		t.Fatalf("Bash冷启动展开了无关Go组件:%v", err)
	}
	cmd = exec.Command(filepath.Join(work, repo, "bin/cdh-toolchain"), "check", "--dest", dest, "--component", "unknown")
	err = cmd.Run()
	fail, ok := err.(*exec.ExitError)
	if !ok || fail.ExitCode() != 2 {
		t.Fatalf("未知组件未返回2:%v", err)
	}
}
