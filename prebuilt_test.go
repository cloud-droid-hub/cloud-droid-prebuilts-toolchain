package version_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func tcCopy(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

// tcTree构造只含revision所需入口的项目目录，组件仓默认没有Git元数据。
func tcTree(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	tcCopy(t, "use-prebuilt.sh", filepath.Join(root, "toolchains", "use-prebuilt.sh"))
	tcCopy(t, filepath.Join("..", "tools", "build", "git-worktree.sh"),
		filepath.Join(root, "tools", "build", "git-worktree.sh"))

	repo := filepath.Join(root, "toolchains")
	bin := filepath.Join(repo, "darwin-arm64", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "cdh-toolchain"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package version\n\n// Value是三个宿主共用的仓库版本。\nconst Value = \"9.9.9\"\n"
	if err := os.WriteFile(filepath.Join(repo, "version.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dest"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, repo
}

func tcRun(t *testing.T, root, repo string) string {
	t.Helper()
	tool := filepath.Join(root, "toolchains", "use-prebuilt.sh")
	cmd := exec.Command("bash", tool, "revision",
		"--host", "darwin-arm64",
		"--repo-root", repo,
		"--dest", filepath.Join(root, "dest"))
	out, err := cmd.Output()
	if err != nil {
		note := ""
		if fail, ok := err.(*exec.ExitError); ok {
			note = string(fail.Stderr)
		}
		t.Fatalf("revision失败: %v\n%s", err, note)
	}
	return strings.TrimSpace(string(out))
}

func TestTcRevPlain(t *testing.T) {
	t.Parallel()
	root, repo := tcTree(t)
	if got := tcRun(t, root, repo); got != "nogit-9.9.9" {
		t.Fatalf("纯复制组件仓没有按登记版本标识: %q", got)
	}
}

func TestTcRoot(t *testing.T) {
	root, _ := tcTree(t)
	tool := filepath.Join(root, "toolchains", "use-prebuilt.sh")
	cmd := exec.Command("bash", tool, "revision", "--host", "darwin-arm64",
		"--dest", filepath.Join(root, "dest"))
	var errs strings.Builder
	cmd.Stderr = &errs
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "nogit-9.9.9" {
		t.Fatalf("默认组件根没有采用工具链目录: %v\n%s\n%s", err, out, &errs)
	}
}

func TestTcRevGit(t *testing.T) {
	t.Parallel()
	root, repo := tcTree(t)
	runTcGit(t, repo, "init", "-q", "-b", "main", "--template=")
	runTcGit(t, repo, "config", "user.name", "tc-test")
	runTcGit(t, repo, "config", "user.email", "tc@example.invalid")
	runTcGit(t, repo, "add", "-A")
	runTcGit(t, repo, "commit", "-q", "-m", "prebuilts")
	bare := filepath.Join(t.TempDir(), "origin.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	runTcGit(t, bare, "init", "-q", "--bare", "-b", "main", "--template=")
	runTcGit(t, repo, "remote", "add", "origin", bare)
	runTcGit(t, repo, "push", "-q", "-u", "origin", "main")
	got := tcRun(t, root, repo)
	if strings.HasPrefix(got, "nogit-") {
		t.Fatalf("有Git元数据的组件仓退回登记版本: %q", got)
	}
	if len(got) != 40 {
		t.Fatalf("组件仓版本不是提交对象: %q", got)
	}
}

// TestTcRevOuter覆盖打包源码被放进其它仓库的情况：判定不能落到上级仓。
func TestTcRevOuter(t *testing.T) {
	t.Parallel()
	root, repo := tcTree(t)
	runTcGit(t, root, "init", "-q", "-b", "main", "--template=")
	runTcGit(t, root, "config", "user.name", "tc-test")
	runTcGit(t, root, "config", "user.email", "tc@example.invalid")
	runTcGit(t, root, "add", "-A")
	runTcGit(t, root, "commit", "-q", "-m", "outer")
	if got := tcRun(t, root, repo); got != "nogit-9.9.9" {
		t.Fatalf("组件仓版本落到上级仓: %q", got)
	}
}

func runTcGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
