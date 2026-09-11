// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func TestGo126(t *testing.T) {
	spec := catalog.Current()
	if spec.Host != "linux-amd64" || !spec.Ready || len(spec.Archives) < 1 {
		t.Fatalf("host=%q ready=%v archives=%d", spec.Host, spec.Ready, len(spec.Archives))
	}
	archive := spec.Archives[0]
	if archive.Name != "Go" || archive.Version != "1.26.8" || archive.Dir != "go126" {
		t.Fatalf("archive=%+v", archive)
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	license := filepath.Join(repo, "licenses", "go", "LICENSE")
	data, err := os.ReadFile(license)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != "911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad" {
		t.Fatalf("license SHA-256=%s", got)
	}
	info, err := os.Stat(license)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("license mode=%#o", info.Mode().Perm())
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return
	}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	err = filepath.Walk(filepath.Join(dest, "go126"), func(pathName string, item os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		mode := item.Mode()
		if mode.IsDir() && mode.Perm() != 0o755 {
			t.Errorf("directory mode=%#o path=%s", mode.Perm(), pathName)
		}
		if mode.IsRegular() && mode.Perm() != 0o644 && mode.Perm() != 0o755 {
			t.Errorf("file mode=%#o path=%s", mode.Perm(), pathName)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	goBin := filepath.Join(dest, "go126", "bin", "go")
	cmd := exec.Command(goBin, "version")
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go version: %v: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "go version go1.26.8 linux/amd64" {
		t.Fatalf("go version=%q", output)
	}
	info, err = os.Stat(goBin)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("go mode=%#o", info.Mode().Perm())
	}
	binary, err := elf.Open(goBin)
	if err != nil {
		t.Fatal(err)
	}
	machine := binary.Machine
	if err := binary.Close(); err != nil {
		t.Fatal(err)
	}
	if machine != elf.EM_X86_64 {
		t.Fatalf("go machine=%v", machine)
	}
	probe := t.TempDir()
	source := filepath.Join(probe, "main.go")
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(probe, "probe")
	cmd = exec.Command(goBin, "build", "-trimpath", "-o", program, source)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v: %s", err, output)
	}
	binary, err = elf.Open(program)
	if err != nil {
		t.Fatal(err)
	}
	machine = binary.Machine
	if err := binary.Close(); err != nil {
		t.Fatal(err)
	}
	if machine != elf.EM_X86_64 {
		t.Fatalf("probe machine=%v", machine)
	}
}

func TestJDK21(t *testing.T) {
	spec := catalog.Current()
	if len(spec.Archives) != 2 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	archive := spec.Archives[1]
	if archive.Name != "JDK" || archive.Version != "21.0.12.1+1" || archive.Dir != "jdk21" {
		t.Fatalf("archive=%+v", archive)
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "NOTICE", want: "c02756bcd9fa8191bf0fda4451bc018414dd44ee35bf09922c24377a475e4b5a"},
		{name: "java.base/ADDITIONAL_LICENSE_INFO", want: "a69bce275ba7a3570af6579cb0f55682cd75fedfcd49e0e8e9022270c447c916"},
		{name: "java.base/ASSEMBLY_EXCEPTION", want: "75292f03bf23d3db7c985aecc191029b93883200721ed23ed34a2e601463df33"},
		{name: "java.base/LICENSE", want: "4b9abebc4338048a7c2dc184e9f800deb349366bdf28eb23c2677a77b4c87726"},
	} {
		pathName := filepath.Join(repo, "licenses", "jdk", filepath.FromSlash(item.name))
		data, err := os.ReadFile(pathName)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != item.want {
			t.Fatalf("%s SHA-256=%s", item.name, got)
		}
		info, err := os.Stat(pathName)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("%s mode=%#o", item.name, info.Mode().Perm())
		}
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return
	}
	t.Setenv("JAVA_TOOL_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("JDK_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	spec.Archives = []catalog.Archive{archive}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dest, "jdk21")
	err = filepath.Walk(root, func(pathName string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		mode := info.Mode()
		if mode.IsDir() && mode.Perm() != 0o755 {
			t.Errorf("directory mode=%#o path=%s", mode.Perm(), pathName)
		}
		if mode.IsRegular() && mode.Perm() != 0o644 && mode.Perm() != 0o755 {
			t.Errorf("file mode=%#o path=%s", mode.Perm(), pathName)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	java := filepath.Join(root, "bin", "java")
	cmd := exec.Command(java, "-version")
	cmd.Env = javaEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("java version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "Temurin-21.0.12.1+1") {
		t.Fatalf("java version=%q", output)
	}
	for _, program := range []string{java, filepath.Join(root, "bin", "javac")} {
		binary, err := elf.Open(program)
		if err != nil {
			t.Fatal(err)
		}
		machine := binary.Machine
		if err := binary.Close(); err != nil {
			t.Fatal(err)
		}
		if machine != elf.EM_X86_64 {
			t.Fatalf("JDK machine=%v", machine)
		}
	}
	probe := t.TempDir()
	source := filepath.Join(probe, "Probe.java")
	body := "public class Probe { public static void main(String[] args) { System.out.println(\"jdk-ok\"); } }\n"
	if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(filepath.Join(root, "bin", "javac"), "-d", probe, source)
	cmd.Env = javaEnv()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("javac: %v: %s", err, output)
	}
	cmd = exec.Command(java, "-cp", probe, "Probe")
	cmd.Env = javaEnv()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("java probe: %v: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "jdk-ok" {
		t.Fatalf("java probe=%q", output)
	}
}

func javaEnv() []string {
	result := make([]string, 0, len(os.Environ()))
	for _, setting := range os.Environ() {
		name, _, _ := strings.Cut(setting, "=")
		if name != "JAVA_TOOL_OPTIONS" && name != "_JAVA_OPTIONS" && name != "JDK_JAVA_OPTIONS" {
			result = append(result, setting)
		}
	}
	return result
}
