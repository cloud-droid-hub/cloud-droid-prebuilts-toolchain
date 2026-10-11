// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyMove(t *testing.T) {
	spec, source := makeMany(t, 2)
	base := t.TempDir()
	repo := filepath.Join(base, "toolchains", spec.Host)
	if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, repo); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(base, "output", "toolchains", spec.Host)
	dest := filepath.Join(base, "output", "target", "common", "toolchains", spec.Host)
	if err := Run(context.Background(), repo, old, spec); err != nil {
		t.Fatal(err)
	}
	first := spec.Archives[0].Dir
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(old, first), filepath.Join(dest, first)); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(filepath.Join(old, spec.Archives[1].Dir))
	if err != nil {
		t.Fatal(err)
	}
	dropParts(t, repo, spec.Archives...)
	for round := range 2 {
		if err := Run(context.Background(), repo, dest, spec); err != nil {
			t.Fatalf("round=%d迁移失败：%v", round, err)
		}
		after, err := os.Stat(filepath.Join(dest, spec.Archives[1].Dir))
		if err != nil || !os.SameFile(before, after) {
			t.Fatalf("已核验组件没有保持身份：%v", err)
		}
		if err := Check(context.Background(), dest, spec); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(filepath.Join(base, "output", "toolchains")); !os.IsNotExist(err) {
			t.Fatalf("旧工具链根未清理：%v", err)
		}
	}
}

func TestLegacyDupe(t *testing.T) {
	spec, source := makeMany(t, 2)
	base := t.TempDir()
	repo := filepath.Join(base, "toolchains", spec.Host)
	if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, repo); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(base, "output", "toolchains", spec.Host)
	dest := filepath.Join(base, "output", "target", "common", "toolchains", spec.Host)
	for _, root := range []string{dest, old} {
		if err := Run(context.Background(), repo, root, spec); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("id.txt", filepath.Join(root, spec.Archives[0].Dir, "id-link")); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(dest, spec.Archives[0].Dir, "id.txt")
	before, err := os.Stat(keep)
	if err != nil {
		t.Fatal(err)
	}
	for round := range 2 {
		if err := Run(context.Background(), repo, dest, spec); err != nil {
			t.Fatalf("round=%d重复组件清理失败：%v", round, err)
		}
		after, err := os.Stat(keep)
		if err != nil || !os.SameFile(before, after) {
			t.Fatalf("新工具链被替换：%v", err)
		}
		if _, err := os.Lstat(old); !os.IsNotExist(err) {
			t.Fatalf("重复旧安装仍然存在：%v", err)
		}
		if got, err := os.Readlink(filepath.Join(dest, spec.Archives[0].Dir, "id-link")); err != nil || got != "id.txt" {
			t.Fatalf("新安装的正常链接被改变：%q %v", got, err)
		}
	}
}

func TestLegacyKeep(t *testing.T) {
	for _, name := range []string{"extra", "bytes", "root", "link"} {
		t.Run(name, func(t *testing.T) {
			spec, source := makeMany(t, 2)
			base := t.TempDir()
			repo := filepath.Join(base, "toolchains", spec.Host)
			if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(source, repo); err != nil {
				t.Fatal(err)
			}
			old := filepath.Join(base, "output", "toolchains", spec.Host)
			dest := filepath.Join(base, "output", "target", "common", "toolchains", spec.Host)
			for _, root := range []string{dest, old} {
				if err := Run(context.Background(), repo, root, spec); err != nil {
					t.Fatal(err)
				}
			}
			keep := filepath.Join(old, spec.Archives[0].Dir, "extra.txt")
			switch name {
			case "bytes":
				keep = filepath.Join(old, spec.Archives[0].Dir, "id.txt")
			case "root":
				keep = filepath.Join(old, "unknown.txt")
			case "link":
				keep = filepath.Join(base, "outside.txt")
			}
			if err := os.WriteFile(keep, []byte("keep original\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if name == "link" {
				if err := os.RemoveAll(filepath.Join(old, spec.Archives[0].Dir)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Dir(keep), filepath.Join(old, spec.Archives[0].Dir)); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Stat(keep)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				err := Run(context.Background(), repo, dest, spec)
				if name == "root" {
					if err != nil {
						t.Fatal(err)
					}
					if err := CheckLegacy(repo, dest, spec.Host); err == nil {
						t.Fatal("完整初始化没有报告未知旧文件")
					}
				} else if err == nil {
					t.Fatal("冲突或链接旧组件没有拒绝")
				}
				after, err := os.Stat(keep)
				data, readErr := os.ReadFile(keep)
				if err != nil || readErr != nil || !os.SameFile(before, after) || string(data) != "keep original\n" {
					t.Fatalf("旧文件没有保留：stat=%v read=%v data=%q", err, readErr, data)
				}
				if err := Check(context.Background(), dest, spec); err != nil {
					t.Fatalf("新安装受影响：%v", err)
				}
			}
		})
	}
}

func TestLegacyBusy(t *testing.T) {
	spec, source := makeMany(t, 2)
	base := t.TempDir()
	repo := filepath.Join(base, "toolchains", spec.Host)
	if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, repo); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(base, "output", "toolchains", spec.Host)
	dest := filepath.Join(base, "output", "target", "common", "toolchains", spec.Host)
	for _, root := range []string{dest, old} {
		if err := Run(context.Background(), repo, root, spec); err != nil {
			t.Fatal(err)
		}
	}
	bash := os.Getenv("CDH_BASH_BIN")
	if bash == "" {
		t.Fatal("登记Bash路径未绑定")
	}
	cmd := exec.Command(bash, "-c", `exec 3<"$1"; printf ready; read -r line`, "holder", filepath.Join(old, spec.Archives[0].Dir, "id.txt"))
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { input.Close(); cmd.Process.Kill(); cmd.Wait() })
	data := make([]byte, 5)
	if _, err := io.ReadFull(pipe, data); err != nil || string(data) != "ready" {
		t.Fatalf("工具使用进程未准备：%v", err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil || !strings.Contains(err.Error(), "in use") {
		t.Fatalf("仍在使用的旧目录未被拒绝：%v", err)
	}
	if err := Check(context.Background(), old, spec); err != nil {
		t.Fatalf("正在使用的旧安装被修改：%v", err)
	}
	input.Close()
	if err := cmd.Wait(); err != nil {
		if code, ok := err.(*exec.ExitError); !ok || code.ExitCode() != 1 {
			t.Fatal(err)
		}
	}
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatalf("进程结束后迁移失败：%v", err)
	}
}

func TestLegacyPart(t *testing.T) {
	spec, source := makeMany(t, 2)
	base := t.TempDir()
	repo := filepath.Join(base, "toolchains", spec.Host)
	if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, repo); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(base, "output", "toolchains", spec.Host)
	dest := filepath.Join(base, "output", "target", "common", "toolchains", spec.Host)
	for _, root := range []string{dest, old} {
		if err := Run(context.Background(), repo, root, spec); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(old, spec.Archives[1].Dir, "id.txt")
	before, err := os.Stat(keep)
	if err != nil {
		t.Fatal(err)
	}
	part := spec
	part.Archives = spec.Archives[:1]
	if err := Run(context.Background(), repo, dest, part); err != nil {
		t.Fatalf("单组件冷启动失败：%v", err)
	}
	after, err := os.Stat(keep)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("单组件启动改动其它旧组件：%v", err)
	}
	if err := CheckLegacy(repo, dest, spec.Host); err == nil {
		t.Fatal("单组件迁移被误认为完整清理")
	}
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	if err := CheckLegacy(repo, dest, spec.Host); err != nil {
		t.Fatal(err)
	}
}
