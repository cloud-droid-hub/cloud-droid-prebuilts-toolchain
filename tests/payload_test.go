// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"context"
	"crypto/sha256"
	"debug/macho"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func TestPayloads(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "go1.27.0")
	t.Setenv("JAVA_TOOL_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("JDK_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, catalog.Current()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOTOOLCHAIN", "local")
	program := filepath.Join(dest, "go126", "bin", "go")
	output, err := exec.Command(program, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("go version: %v: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "go version go1.26.8 darwin/arm64" {
		t.Fatalf("go version=%q", output)
	}
	info, err := os.Stat(program)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("go mode=%#o", info.Mode().Perm())
	}
	t.Setenv("JAVA_TOOL_OPTIONS", "")
	t.Setenv("_JAVA_OPTIONS", "")
	t.Setenv("JDK_JAVA_OPTIONS", "")
	java := filepath.Join(dest, "jdk21", "bin", "java")
	output, err = exec.Command(java, "-version").CombinedOutput()
	if err != nil {
		t.Fatalf("java version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "Temurin-21.0.12.1+1") {
		t.Fatalf("java version=%q", output)
	}
	info, err = os.Stat(java)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("java mode=%#o", info.Mode().Perm())
	}
	for _, name := range []string{"CodeResources", "libjli.dylib", "Info.plist"} {
		if _, err := os.Lstat(filepath.Join(dest, "jdk21", name)); !os.IsNotExist(err) {
			t.Fatalf("JDK根含包装文件%s: %v", name, err)
		}
	}
	clang := filepath.Join(dest, "llvm19", "bin", "clang")
	output, err = exec.Command(clang, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("clang version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "clang version 19.1.7") {
		t.Fatalf("clang version=%q", output)
	}
	lld := filepath.Join(dest, "llvm19", "bin", "ld64.lld")
	output, err = exec.Command(lld, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("lld version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "LLD 19.1.7") {
		t.Fatalf("lld version=%q", output)
	}
	llvmAr := filepath.Join(dest, "llvm19", "bin", "llvm-ar")
	output, err = exec.Command(llvmAr, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("llvm-ar version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "LLVM version 19.1.7") {
		t.Fatalf("llvm-ar version=%q", output)
	}
}

func TestLLVM22(t *testing.T) {
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	license := filepath.Join(repo, "licenses", "llvm22", "LICENSE.TXT")
	licenseData, err := os.ReadFile(license)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(licenseData)
	if got := hex.EncodeToString(sum[:]); got != "8d85c1057d742e597985c7d4e6320b015a9139385cff4cbae06ffc0ebe89afee" {
		t.Fatalf("LLVM 22 license SHA-256=%s", got)
	}
	licenseInfo, err := os.Stat(license)
	if err != nil {
		t.Fatal(err)
	}
	if licenseInfo.Mode().Perm() != 0o644 {
		t.Fatalf("LLVM 22 license mode=%#o", licenseInfo.Mode().Perm())
	}
	dest := filepath.Join(t.TempDir(), "tools")
	spec := catalog.Current()
	spec.Archives = spec.Archives[3:]
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	clang := filepath.Join(dest, "llvm22", "bin", "clang")
	output, err := exec.Command(clang, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("clang 22 version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "clang version 22.1.8") {
		t.Fatalf("clang 22 version=%q", output)
	}
	lld := filepath.Join(dest, "llvm22", "bin", "ld64.lld")
	output, err = exec.Command(lld, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("lld 22 version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "LLD 22.1.8") {
		t.Fatalf("lld 22 version=%q", output)
	}
	llvmAr := filepath.Join(dest, "llvm22", "bin", "llvm-ar")
	output, err = exec.Command(llvmAr, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("llvm-ar 22 version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "LLVM version 22.1.8") {
		t.Fatalf("llvm-ar 22 version=%q", output)
	}
	sdkOut, err := exec.Command("xcrun", "--sdk", "macosx", "--show-sdk-path").CombinedOutput()
	if err != nil {
		t.Fatalf("macOS SDK: %v: %s", err, sdkOut)
	}
	sdk := strings.TrimSpace(string(sdkOut))
	if !filepath.IsAbs(sdk) {
		t.Fatalf("macOS SDK path=%q", sdk)
	}
	probeDir := t.TempDir()
	for _, probe := range []struct {
		name string
		ext  string
		tool string
		body string
	}{
		{name: "c", ext: ".c", tool: clang, body: "int main(void) { return 0; }\n"},
		{name: "cc", ext: ".cc", tool: filepath.Join(dest, "llvm22", "bin", "clang++"), body: "int main() { return 0; }\n"},
	} {
		source := filepath.Join(probeDir, probe.name+probe.ext)
		program := filepath.Join(probeDir, probe.name)
		if err := os.WriteFile(source, []byte(probe.body), 0o644); err != nil {
			t.Fatal(err)
		}
		output, err = exec.Command(probe.tool, "-isysroot", sdk, "-fuse-ld=lld", source, "-o", program).CombinedOutput()
		if err != nil {
			t.Fatalf("%s link: %v: %s", probe.name, err, output)
		}
		binary, err := macho.Open(program)
		if err != nil {
			t.Fatalf("%s Mach-O: %v", probe.name, err)
		}
		cpu := binary.Cpu
		if err := binary.Close(); err != nil {
			t.Fatal(err)
		}
		if cpu != macho.CpuArm64 {
			t.Fatalf("%s cpu=%v", probe.name, cpu)
		}
	}
}
