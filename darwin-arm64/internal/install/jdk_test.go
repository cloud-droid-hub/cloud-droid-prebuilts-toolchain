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

func jdkSample(t *testing.T) (catalog.Spec, string, string) {
	t.Helper()
	body := "profile=jdk21-build-v1\nversion=21.0.12.1+1\n"
	spec, repo := makeSpec(t, []tarItem{{header: tar.Header{
		Name: "payload/cdh-jdk-trim.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body)),
	}, data: body}})
	arc := &spec.Archives[0]
	arc.Name, arc.Dir, arc.Version = "JDK", "jdk21", "21.0.12.1+1"
	arc.DataOnly, arc.Probes = true, nil
	arc.Files = []catalog.FileCheck{{Path: "cdh-jdk-trim.txt", Size: int64(len(body)), SHA256: sumHex([]byte(body)), Want: []string{arc.Version}}}
	dest := filepath.Join(t.TempDir(), "tools")
	old := filepath.Join(dest, "jdk21")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "legacy.txt"), []byte("full upstream JDK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return spec, repo, dest
}

func TestJDKMigrate(t *testing.T) {
	spec, repo, dest := jdkSample(t)
	old, err := os.Stat(filepath.Join(dest, "jdk21"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(dest, "jdk21-original")
	kept, err := os.Stat(backup)
	if err != nil || !os.SameFile(old, kept) {
		t.Fatalf("旧JDK目录身份没有保留: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(backup, "legacy.txt"))
	if err != nil || string(body) != "full upstream JDK\n" {
		t.Fatalf("旧JDK内容改变: %v %q", err, body)
	}
	dropParts(t, repo, spec.Archives[0])
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatalf("第二轮初始化未复用已验证JDK: %v", err)
	}
	if err := Check(context.Background(), dest, spec); err != nil {
		t.Fatal(err)
	}
}

func TestJDKBackup(t *testing.T) {
	spec, repo, dest := jdkSample(t)
	backup := filepath.Join(dest, "jdk21-original")
	if err := os.Mkdir(backup, 0o755); err != nil {
		t.Fatal(err)
	}
	prior, err := os.Stat(filepath.Join(dest, "jdk21"))
	if err != nil {
		t.Fatal(err)
	}
	kept, err := os.Stat(backup)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("已有备份被覆盖")
	}
	for path, want := range map[string]os.FileInfo{"jdk21": prior, "jdk21-original": kept} {
		got, err := os.Stat(filepath.Join(dest, path))
		if err != nil || !os.SameFile(want, got) {
			t.Fatalf("已有目录身份改变: %s %v", path, err)
		}
	}
}

func TestJDKBadMark(t *testing.T) {
	spec, repo, dest := jdkSample(t)
	marker := filepath.Join(dest, "jdk21", "cdh-jdk-trim.txt")
	if err := os.WriteFile(marker, []byte("invalid marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("损坏的裁剪组件被自动替换")
	}
	body, err := os.ReadFile(marker)
	if err != nil || string(body) != "invalid marker\n" {
		t.Fatalf("原标记改变: %v %q", err, body)
	}
	if _, err := os.Lstat(filepath.Join(dest, "jdk21-original")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("损坏组件产生备份目录: %v", err)
	}
}

func TestJDKBadPart(t *testing.T) {
	spec, repo, dest := jdkSample(t)
	part := filepath.Join(repo, spec.Archives[0].Parts[0].Path)
	if err := os.WriteFile(part, []byte("broken archive\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prior, err := os.Stat(filepath.Join(dest, "jdk21"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("损坏归档通过迁移")
	}
	got, err := os.Stat(filepath.Join(dest, "jdk21"))
	if err != nil || !os.SameFile(prior, got) {
		t.Fatalf("失败后原JDK目录改变: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dest, "jdk21-original")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("失败后出现备份目录: %v", err)
	}
}
