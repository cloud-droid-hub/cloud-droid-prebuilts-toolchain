package install

import (
	"archive/tar"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
)

func TestJDKCases(t *testing.T) {
	for _, name := range []string{"legacy", "backup", "marker", "archive"} {
		t.Run(name, func(t *testing.T) {
			body := "profile=jdk21-build-v1\nversion=21.0.12.1+1\n"
			spec, repo := makeSpec(t, []tarItem{{header: tar.Header{
				Name: "payload/cdh-jdk-trim.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body)),
			}, data: body}})
			arc := &spec.Archives[0]
			arc.Name, arc.Dir, arc.Version = "JDK", "jdk21", "21.0.12.1+1"
			arc.DataOnly, arc.Probes = true, nil
			arc.Files = []catalog.FileCheck{{Path: "cdh-jdk-trim.txt", Size: int64(len(body)), SHA256: sumHex([]byte(body)), Want: []string{arc.Version}}}
			dest := filepath.Join(t.TempDir(), "tools")
			old, backup := filepath.Join(dest, "jdk21"), filepath.Join(dest, "jdk21-original")
			if err := os.MkdirAll(old, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(old, "legacy.txt"), []byte("full JDK\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			prior, err := os.Stat(old)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "backup":
				if err := os.Mkdir(backup, 0o755); err != nil {
					t.Fatal(err)
				}
			case "marker":
				if err := os.WriteFile(filepath.Join(old, "cdh-jdk-trim.txt"), []byte("invalid\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "archive":
				if err := os.WriteFile(filepath.Join(repo, arc.Parts[0].Path), []byte("broken\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err = Run(context.Background(), repo, dest, spec)
			if (err == nil) != (name == "legacy") {
				t.Fatalf("迁移结果: %v", err)
			}
			kept := old
			if name == "legacy" {
				kept = backup
				dropParts(t, repo, *arc)
				if err := Run(context.Background(), repo, dest, spec); err != nil {
					t.Fatalf("重复初始化失败: %v", err)
				}
				if err := Check(context.Background(), dest, spec); err != nil {
					t.Fatal(err)
				}
			}
			got, err := os.Stat(kept)
			if err != nil || !os.SameFile(prior, got) {
				t.Fatalf("原目录身份改变: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(kept, "legacy.txt"))
			if err != nil || string(data) != "full JDK\n" {
				t.Fatalf("原目录内容改变: %v %q", err, data)
			}
		})
	}
}
