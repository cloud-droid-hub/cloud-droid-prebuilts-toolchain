// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/selectcfg"
)

func TestGo126(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("需要Linux arm64宿主执行Go安装与编译验证")
	}
	spec := catalog.Current()
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
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
	if strings.TrimSpace(string(output)) != "go version go1.26.8 linux/arm64" {
		t.Fatalf("go version=%q", output)
	}
	info, err := os.Stat(goBin)
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
	if machine != elf.EM_AARCH64 {
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
	if machine != elf.EM_AARCH64 {
		t.Fatalf("probe machine=%v", machine)
	}
}

func TestJDK21(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("需要Linux arm64宿主执行JDK安装与编译验证")
	}
	spec := catalog.Current()
	archive := spec.Archives[1]
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
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
		if machine != elf.EM_AARCH64 {
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

func TestLLVM22(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("需要Linux arm64宿主执行LLVM安装与链接验证")
	}
	spec := catalog.Current()
	archive := spec.Archives[3]
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	spec.Archives = []catalog.Archive{archive}
	root := os.Getenv("CDH_LLVM22_TEST_ROOT")
	if root == "" {
		dest := filepath.Join(t.TempDir(), "tools")
		if err := install.Run(context.Background(), repo, dest, spec); err != nil {
			t.Fatal(err)
		}
		root = filepath.Join(dest, "llvm22")
	}
	if !filepath.IsAbs(root) || filepath.Base(root) != "llvm22" {
		t.Fatalf("invalid LLVM 22 root: %q", root)
	}
	if err := install.Check(context.Background(), filepath.Dir(root), spec); err != nil {
		t.Fatal(err)
	}
	hostLibs := map[string]bool{
		"ld-linux-aarch64.so.1": true,
		"libbsd.so.0":           true,
		"libc.so.6":             true,
		"libcom_err.so.2":       true,
		"libcrypto.so.3":        true,
		"libgcc_s.so.1":         true,
		"libgmp.so.10":          true,
		"libm.so.6":             true,
		"libmd.so.0":            true,
		"libresolv.so.2":        true,
		"libssl.so.3":           true,
		"libstdc++.so.6":        true,
		"libtinfo.so.6":         true,
		"libz.so.1":             true,
		"libzstd.so.1":          true,
	}
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
		if !mode.IsRegular() {
			return nil
		}
		binary, openErr := elf.Open(pathName)
		if openErr != nil {
			return nil
		}
		libs, libErr := binary.ImportedLibraries()
		closeErr := binary.Close()
		if closeErr != nil {
			return closeErr
		}
		if libErr != nil || len(libs) == 0 {
			return nil
		}
		output, runErr := exec.Command("ldd", pathName).CombinedOutput()
		if runErr != nil {
			return fmt.Errorf("ldd %s: %w: %s", pathName, runErr, output)
		}
		if strings.Contains(string(output), "not found") {
			return fmt.Errorf("missing dependency for %s: %s", pathName, output)
		}
		for _, line := range strings.Split(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[1] == "=>" && strings.HasPrefix(fields[2], "/") &&
				!strings.HasPrefix(fields[2], root+string(filepath.Separator)) && !hostLibs[fields[0]] {
				return fmt.Errorf("unexpected host dependency for %s: %s", pathName, fields[0])
			}
			if len(fields) >= 2 && strings.HasPrefix(fields[0], "/") &&
				!strings.HasPrefix(fields[0], root+string(filepath.Separator)) && !hostLibs[filepath.Base(fields[0])] {
				return fmt.Errorf("unexpected host dependency for %s: %s", pathName, fields[0])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "clang", want: "Ubuntu clang version 22.1.2"},
		{name: "ld.lld", want: "Ubuntu LLD 22.1.2"},
		{name: "llvm-ar", want: "Ubuntu LLVM version 22.1.2"},
	} {
		program := filepath.Join(root, "bin", item.name)
		output, err := exec.Command(program, "--version").CombinedOutput()
		if err != nil {
			t.Fatalf("%s version: %v: %s", item.name, err, output)
		}
		if !strings.Contains(string(output), item.want) {
			t.Fatalf("%s version=%q", item.name, output)
		}
		binary, err := elf.Open(program)
		if err != nil {
			t.Fatal(err)
		}
		machine := binary.Machine
		if err := binary.Close(); err != nil {
			t.Fatal(err)
		}
		if machine != elf.EM_AARCH64 {
			t.Fatalf("%s machine=%v", item.name, machine)
		}
		output, err = exec.Command("ldd", program).CombinedOutput()
		if err != nil {
			t.Fatalf("%s ldd: %v: %s", item.name, err, output)
		}
		for _, want := range []string{"/lib/cdh/libLLVM.so.22.1", "/lib/cdh/libffi.so.8", "/lib/cdh/libxml2.so.16"} {
			if !strings.Contains(string(output), want) {
				t.Fatalf("%s dependencies miss %q: %s", item.name, want, output)
			}
		}
		if item.name == "clang" && !strings.Contains(string(output), "/lib/cdh/libclang-cpp.so.22.1") {
			t.Fatalf("clang dependencies=%s", output)
		}
	}
	probe := t.TempDir()
	object := filepath.Join(probe, "probe.o")
	source := filepath.Join(probe, "probe.c")
	if err := os.WriteFile(source, []byte("int probe(void) { return 7; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clang := filepath.Join(root, "bin", "clang")
	output, err := exec.Command(clang, "-c", "-nostdlib", source, "-o", object).CombinedOutput()
	if err != nil {
		t.Fatalf("C compile: %v: %s", err, output)
	}
	library := filepath.Join(probe, "libprobe.a")
	ar := filepath.Join(root, "bin", "llvm-ar")
	output, err = exec.Command(ar, "rcs", library, object).CombinedOutput()
	if err != nil {
		t.Fatalf("archive: %v: %s", err, output)
	}
	output, err = exec.Command(ar, "t", library).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "probe.o" {
		t.Fatalf("archive members: %v: %s", err, output)
	}
	for _, item := range []struct {
		name string
		ext  string
		body string
		tool string
		more []string
	}{
		{name: "c", ext: ".c", body: "int probe(void); void _start(void) { (void)probe(); }\n", tool: "clang", more: []string{library}},
		{name: "cc", ext: ".cc", body: "extern \"C\" void _start() {}\n", tool: "clang++"},
	} {
		source := filepath.Join(probe, item.name+item.ext)
		if err := os.WriteFile(source, []byte(item.body), 0o644); err != nil {
			t.Fatal(err)
		}
		program := filepath.Join(probe, item.name)
		args := []string{"-fuse-ld=lld", "-nostdlib", "-no-pie", "-Wl,-e,_start", source}
		args = append(args, item.more...)
		args = append(args, "-o", program)
		output, err := exec.Command(filepath.Join(root, "bin", item.tool), args...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s link: %v: %s", item.name, err, output)
		}
		binary, err := elf.Open(program)
		if err != nil {
			t.Fatal(err)
		}
		machine := binary.Machine
		kind := binary.Type
		if err := binary.Close(); err != nil {
			t.Fatal(err)
		}
		if machine != elf.EM_AARCH64 || kind != elf.ET_EXEC {
			t.Fatalf("%s machine=%v type=%v", item.name, machine, kind)
		}
	}
}

func TestNDK29(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("需要Linux arm64宿主执行NDK安装与链接验证")
	}
	spec := catalog.Current()
	ndk := spec.Archives[4]
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	spec.Archives = []catalog.Archive{spec.Archives[3], ndk}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	root, err := selectcfg.Resolve(dest, "ndk", spec)
	if err != nil {
		t.Fatal(err)
	}
	sysroot, err := selectcfg.Resolve(dest, "ndk-sysroot", spec)
	if err != nil {
		t.Fatal(err)
	}
	resource, err := selectcfg.Resolve(dest, "ndk-resource", spec)
	if err != nil {
		t.Fatal(err)
	}
	clang, err := selectcfg.Resolve(dest, "llvm22-clang", spec)
	if err != nil {
		t.Fatal(err)
	}
	clangxx, err := selectcfg.Resolve(dest, "llvm22-clang++", spec)
	if err != nil {
		t.Fatal(err)
	}
	lld, err := selectcfg.Resolve(dest, "llvm22-lld", spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "NOTICE", want: "faf0dc52cf6ce9ab09a04b0c074a378dd2565ee1ed9bc1179d83793bcdba012e"},
		{name: "NOTICE.toolchain", want: "789714a9076571c580cf531bdcebec776a4d889c12c3c1108adafc6e4338710a"},
		{name: "source.properties", want: "716f3518a923198cfab037abb32dc3f1b1f7e9a9dcdcda6b66be5906215d2658"},
	} {
		pathName := filepath.Join(root, item.name)
		data, err := os.ReadFile(pathName)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != item.want {
			t.Fatalf("installed %s SHA-256=%s", item.name, got)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "source.properties"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Pkg.Revision = 29.0.14206865") {
		t.Fatalf("source.properties=%q", data)
	}
	for _, pathName := range []string{
		filepath.Join(root, "build"),
		filepath.Join(root, "meta"),
		filepath.Join(root, "ndk-build"),
		filepath.Join(root, "toolchains", "llvm", "prebuilt", "linux-x86_64", "bin"),
	} {
		if _, err := os.Lstat(pathName); !os.IsNotExist(err) {
			t.Fatalf("excluded path=%s err=%v", pathName, err)
		}
	}
	for _, triple := range []string{
		"arm-linux-androideabi",
		"i686-linux-android",
		"riscv64-linux-android",
		"x86_64-linux-android",
	} {
		for _, parent := range []string{
			filepath.Join(sysroot, "usr", "include"),
			filepath.Join(sysroot, "usr", "lib"),
		} {
			pathName := filepath.Join(parent, triple)
			if _, err := os.Lstat(pathName); !os.IsNotExist(err) {
				t.Fatalf("excluded target=%s err=%v", pathName, err)
			}
		}
	}
	elfCount := 0
	err = filepath.Walk(root, func(pathName string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		mode := info.Mode()
		if mode.IsDir() && mode.Perm() != 0o755 {
			return fmt.Errorf("directory mode=%#o path=%s", mode.Perm(), pathName)
		}
		if !mode.IsRegular() {
			return nil
		}
		if mode.Perm() != 0o644 {
			return fmt.Errorf("file mode=%#o path=%s", mode.Perm(), pathName)
		}
		binary, openErr := elf.Open(pathName)
		if openErr != nil {
			return nil
		}
		elfCount++
		machine := binary.Machine
		if closeErr := binary.Close(); closeErr != nil {
			return closeErr
		}
		if machine != elf.EM_AARCH64 {
			return fmt.Errorf("ELF machine=%v path=%s", machine, pathName)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if elfCount == 0 {
		t.Fatal("NDK has no target ELF input")
	}
	probe := t.TempDir()
	link := func(tool, source, program string, more ...string) {
		t.Helper()
		args := []string{
			"--target=aarch64-linux-android35",
			"--sysroot=" + sysroot,
			"-resource-dir=" + resource,
			"--rtlib=compiler-rt",
			"--ld-path=" + lld,
		}
		args = append(args, more...)
		args = append(args, source, "-o", program)
		output, err := exec.Command(tool, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("Android link: %v: %s", err, output)
		}
		binary, err := elf.Open(program)
		if err != nil {
			t.Fatal(err)
		}
		machine := binary.Machine
		kind := binary.Type
		section := binary.Section(".interp")
		if section == nil {
			binary.Close()
			t.Fatalf("%s has no interpreter", program)
		}
		interp, err := section.Data()
		if closeErr := binary.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatal(err)
		}
		if machine != elf.EM_AARCH64 || kind != elf.ET_DYN ||
			strings.TrimRight(string(interp), "\x00") != "/system/bin/linker64" {
			t.Fatalf("%s machine=%v type=%v interp=%q", program, machine, kind, interp)
		}
	}
	cSource := filepath.Join(probe, "probe.c")
	cBody := "unsigned long long divide(unsigned __int128 v) { return v / 7; }\nint main(int n, char **v) { return divide((unsigned)n + (unsigned)v[0][0]) == 0; }\n"
	if err := os.WriteFile(cSource, []byte(cBody), 0o644); err != nil {
		t.Fatal(err)
	}
	link(clang, cSource, filepath.Join(probe, "probe-c"))
	ccSource := filepath.Join(probe, "probe.cc")
	ccBody := "#include <vector>\nint main() { std::vector<int> v{1, 2, 3}; return v.size() == 3 ? 0 : 1; }\n"
	if err := os.WriteFile(ccSource, []byte(ccBody), 0o644); err != nil {
		t.Fatal(err)
	}
	link(clangxx, ccSource, filepath.Join(probe, "probe-cc"), "--unwindlib=libunwind", "-stdlib=libc++")
	if mainRoot := os.Getenv("CDH_MAIN_ROOT"); mainRoot != "" {
		source := filepath.Join(mainRoot, "tools", "android", "getaddrinfo-probe.c")
		if _, err := os.Stat(source); err != nil {
			t.Fatal(err)
		}
		link(clang, source, filepath.Join(probe, "main-probe"))
	}
}

func TestBazel(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("需要Linux arm64宿主执行Bazel安装与运行验证")
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	spec := catalog.Current()
	bzl := spec.Archives[5]
	spec.Archives = []catalog.Archive{bzl}
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
	bin, err := elf.Open(bazel)
	if err != nil {
		t.Fatal(err)
	}
	if bin.Machine != elf.EM_AARCH64 {
		t.Errorf("bazel machine=%v", bin.Machine)
	}
	if err := bin.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(bazel)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("bazel mode=%#o", info.Mode().Perm())
	}
}

func TestRust197(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("需要Linux arm64宿主执行Rust安装与编译验证")
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	spec := catalog.Current()
	rust := spec.Archives[6]
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
