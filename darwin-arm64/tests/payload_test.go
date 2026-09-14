// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"context"
	"crypto/sha256"
	"debug/elf"
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

func TestNDK29(t *testing.T) {
	spec := catalog.Current()
	if len(spec.Archives) != 7 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	ndk := spec.Archives[4]
	if ndk.Name != "Android NDK" || ndk.Version != "29.0.14206865" || ndk.Dir != "ndk29" {
		t.Fatalf("archive=%+v", ndk)
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	for _, notice := range []struct {
		name string
		want string
	}{
		{name: "NOTICE", want: "faf0dc52cf6ce9ab09a04b0c074a378dd2565ee1ed9bc1179d83793bcdba012e"},
		{name: "NOTICE.toolchain", want: "789714a9076571c580cf531bdcebec776a4d889c12c3c1108adafc6e4338710a"},
	} {
		pathName := filepath.Join(repo, "licenses", "ndk29", notice.name)
		data, err := os.ReadFile(pathName)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != notice.want {
			t.Fatalf("%s SHA-256=%s", notice.name, got)
		}
		info, err := os.Stat(pathName)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("%s mode=%#o", notice.name, info.Mode().Perm())
		}
	}
	dest := filepath.Join(t.TempDir(), "tools")
	spec.Archives = []catalog.Archive{ndk}
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dest, "ndk29")
	properties, err := os.ReadFile(filepath.Join(root, "source.properties"))
	if err != nil {
		t.Fatal(err)
	}
	wantProps := "Pkg.Desc = Android NDK\nPkg.Revision = 29.0.14206865\nPkg.BaseRevision = 29.0.14206865\nPkg.ReleaseName = r29\n"
	if string(properties) != wantProps {
		t.Fatalf("source.properties=%q", properties)
	}
	host := filepath.Join(root, "toolchains", "llvm", "prebuilt", "darwin-x86_64")
	for _, rel := range []string{
		"sysroot/usr/include/aarch64-linux-android",
		"sysroot/usr/lib/aarch64-linux-android",
		"lib/clang/21/lib/linux",
	} {
		if info, err := os.Stat(filepath.Join(host, rel)); err != nil || !info.IsDir() {
			t.Fatalf("required directory %s: %v", rel, err)
		}
	}
	for _, triple := range []string{
		"arm-linux-androideabi",
		"i686-linux-android",
		"x86_64-linux-android",
		"riscv64-linux-android",
	} {
		for _, parent := range []string{"sysroot/usr/include", "sysroot/usr/lib"} {
			pathName := filepath.Join(host, parent, triple)
			if _, err := os.Lstat(pathName); !os.IsNotExist(err) {
				t.Fatalf("non-arm64 target path %s: %v", pathName, err)
			}
		}
	}
	binDir := filepath.Join(host, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.Contains(name, "linux-android") && !strings.HasPrefix(name, "aarch64-linux-android") {
			t.Fatalf("non-arm64 target tool=%s", name)
		}
		if name == "yasm" || name == "vsyasm" || name == "ytasm" {
			t.Fatalf("x86 assembler=%s", name)
		}
	}
	resourceLib := filepath.Join(host, "lib", "clang", "21", "lib", "linux")
	err = filepath.Walk(resourceLib, func(pathName string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(resourceLib, pathName)
		if relErr != nil {
			return relErr
		}
		inArmDir := strings.HasPrefix(rel, "aarch64"+string(filepath.Separator))
		if info.Mode().IsRegular() && !inArmDir && !strings.Contains(info.Name(), "aarch64") {
			t.Errorf("non-arm64 runtime=%s", pathName)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	machCount := 0
	elfCount := 0
	err = filepath.Walk(root, func(pathName string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() {
			if info.Mode().Perm() != 0o755 {
				t.Errorf("directory mode %s=%#o", pathName, info.Mode().Perm())
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			t.Errorf("unsupported file %s=%s", pathName, info.Mode())
			return nil
		}
		if mode := info.Mode().Perm(); mode != 0o644 && mode != 0o755 {
			t.Errorf("file mode %s=%#o", pathName, mode)
		}
		if fat, openErr := macho.OpenFat(pathName); openErr == nil {
			fat.Close()
			t.Errorf("fat Mach-O=%s", pathName)
			return nil
		}
		if image, openErr := macho.Open(pathName); openErr == nil {
			machCount++
			if image.Cpu != macho.CpuArm64 {
				t.Errorf("Mach-O cpu %s=%v", pathName, image.Cpu)
			}
			image.Close()
		}
		if image, openErr := elf.Open(pathName); openErr == nil {
			elfCount++
			if image.Machine != elf.EM_AARCH64 {
				t.Errorf("ELF machine %s=%v", pathName, image.Machine)
			}
			image.Close()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if machCount < 3 || elfCount == 0 {
		t.Fatalf("Mach-O=%d ELF=%d", machCount, elfCount)
	}
	probeDir := t.TempDir()
	for _, probe := range []struct {
		name string
		ext  string
		body string
	}{
		{name: "c", ext: ".c", body: "int main(void) { return 0; }\n"},
		{name: "cc", ext: ".cc", body: "#include <vector>\nint main() { std::vector<int> v{0}; return v[0]; }\n"},
	} {
		source := filepath.Join(probeDir, probe.name+probe.ext)
		program := filepath.Join(probeDir, probe.name)
		if err := os.WriteFile(source, []byte(probe.body), 0o644); err != nil {
			t.Fatal(err)
		}
		tool := filepath.Join(binDir, "aarch64-linux-android35-clang")
		if probe.ext == ".cc" {
			tool += "++"
		}
		output, err := exec.Command(tool, "-fuse-ld=lld", source, "-o", program).CombinedOutput()
		if err != nil {
			t.Fatalf("%s link: %v: %s", probe.name, err, output)
		}
		image, err := elf.Open(program)
		if err != nil {
			t.Fatalf("%s ELF: %v", probe.name, err)
		}
		if image.Machine != elf.EM_AARCH64 || image.Type != elf.ET_DYN {
			t.Errorf("%s ELF machine=%v type=%v", probe.name, image.Machine, image.Type)
		}
		interp := image.Section(".interp")
		if interp == nil {
			t.Errorf("%s missing .interp", probe.name)
		} else if data, readErr := interp.Data(); readErr != nil {
			t.Errorf("%s .interp: %v", probe.name, readErr)
		} else if got := strings.TrimRight(string(data), "\x00"); got != "/system/bin/linker64" {
			t.Errorf("%s interpreter=%q", probe.name, got)
		}
		image.Close()
	}
	project := filepath.Join(probeDir, "ndk-project")
	jni := filepath.Join(project, "jni")
	if err := os.MkdirAll(jni, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"Android.mk":     "LOCAL_PATH := $(call my-dir)\ninclude $(CLEAR_VARS)\nLOCAL_MODULE := ndkprobe\nLOCAL_SRC_FILES := ndkprobe.c\ninclude $(BUILD_EXECUTABLE)\n",
		"Application.mk": "APP_ABI := arm64-v8a\nAPP_PLATFORM := android-35\n",
		"ndkprobe.c":     "int main(void) { return 0; }\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(jni, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(filepath.Join(root, "ndk-build"), "V=1")
	cmd.Dir = project
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ndk-build: %v: %s", err, output)
	}
	program := filepath.Join(project, "libs", "arm64-v8a", "ndkprobe")
	image, err := elf.Open(program)
	if err != nil {
		t.Fatalf("ndk-build ELF: %v", err)
	}
	if image.Machine != elf.EM_AARCH64 || image.Type != elf.ET_DYN {
		t.Errorf("ndk-build ELF machine=%v type=%v", image.Machine, image.Type)
	}
	image.Close()
}

func TestBazel(t *testing.T) {
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	lic := filepath.Join(repo, "licenses", "bazel", "LICENSE")
	data, err := os.ReadFile(lic)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30" {
		t.Fatalf("Bazel license SHA-256=%s", got)
	}
	info, err := os.Stat(lic)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("Bazel license mode=%#o", info.Mode().Perm())
	}
	spec := catalog.Current()
	if len(spec.Archives) != 7 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	spec.Archives = spec.Archives[5:]
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	bazel := filepath.Join(dest, "bazel851", "bin", "bazel")
	out, err := exec.Command(bazel, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("bazel version: %v: %s", err, out)
	}
	if strings.TrimSpace(string(out)) != "bazel 8.5.1" {
		t.Fatalf("bazel version=%q", out)
	}
	bin, err := macho.Open(bazel)
	if err != nil {
		t.Fatal(err)
	}
	if bin.Cpu != macho.CpuArm64 {
		t.Errorf("bazel cpu=%v", bin.Cpu)
	}
	if err := bin.Close(); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(bazel)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("bazel mode=%#o", info.Mode().Perm())
	}
}

func TestRust197(t *testing.T) {
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	licenses := []struct {
		name string
		sum  string
	}{
		{name: "COPYRIGHT", sum: "172020dbfd5b53a226dfde77616190a48dcff519b0bc0e6deb91a8450782c4af"},
		{name: "LICENSE-APACHE", sum: "62c7a1e35f56406896d7aa7ca52d0cc0d272ac022b5d2796e7d6905db8a3636a"},
		{name: "LICENSE-MIT", sum: "b71bd43a069ca0641a9ecfe585ca7b3c53b5cc1608f8b68321168698e28b5ea1"},
	}
	for _, item := range licenses {
		name := filepath.Join(repo, "licenses", "rust", item.name)
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != item.sum {
			t.Fatalf("%s SHA-256=%s", item.name, got)
		}
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("%s mode=%#o", item.name, info.Mode().Perm())
		}
	}
	spec := catalog.Current()
	if len(spec.Archives) != 7 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	rust := spec.Archives[6]
	if rust.Name != "Rust" || rust.Version != "1.97.1" || rust.Dir != "rust197" {
		t.Fatalf("archive=%+v", rust)
	}
	spec.Archives = []catalog.Archive{rust}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	rustc := filepath.Join(dest, "rust197", "bin", "rustc")
	output, err := exec.Command(rustc, "-vV").CombinedOutput()
	if err != nil {
		t.Fatalf("rustc version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "release: 1.97.1") ||
		!strings.Contains(string(output), "LLVM version: 22.1.6") {
		t.Fatalf("rustc version=%q", output)
	}
	probe := t.TempDir()
	source := filepath.Join(probe, "probe.rs")
	body := "fn main() { println!(\"42\"); }\n"
	if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(probe, "probe")
	output, err = exec.Command(rustc, source, "-o", program).CombinedOutput()
	if err != nil {
		t.Fatalf("rustc host compile: %v: %s", err, output)
	}
	output, err = exec.Command(program).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "42" {
		t.Fatalf("rust host run: %v: %q", err, output)
	}
	object := filepath.Join(probe, "android.o")
	output, err = exec.Command(rustc, "--target", "aarch64-linux-android", "--crate-type=lib", "--emit=obj", source, "-o", object).CombinedOutput()
	if err != nil {
		t.Fatalf("rustc Android object: %v: %s", err, output)
	}
	image, err := elf.Open(object)
	if err != nil {
		t.Fatal(err)
	}
	if image.Machine != elf.EM_AARCH64 {
		t.Errorf("Android object machine=%v", image.Machine)
	}
	if err := image.Close(); err != nil {
		t.Fatal(err)
	}
}
