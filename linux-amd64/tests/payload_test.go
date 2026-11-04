// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"archive/tar"
	"context"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func payloadSpec(t *testing.T, dirs ...string) catalog.Spec {
	t.Helper()
	spec := catalog.Current()
	all := spec.Archives
	spec.Archives = nil
	for _, dir := range dirs {
		found := false
		for _, arc := range all {
			if arc.Dir == dir {
				spec.Archives = append(spec.Archives, arc)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("安装测试所需组件未登记:%s", dir)
		}
	}
	return spec
}

func TestGo126(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("需要Linux amd64宿主执行Go安装与编译验证")
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
	if strings.TrimSpace(string(output)) != "go version go1.26.8 linux/amd64" {
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("需要Linux amd64宿主执行JDK安装与编译验证")
	}
	spec := payloadSpec(t, "jdk21")
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	t.Setenv("JAVA_TOOL_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("JDK_JAVA_OPTIONS", "-XX:InvalidCDHOption")
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

func TestLLVM22GLIBC(t *testing.T) {
	repo, archive := llvm22Archive(t)
	stream := llvm22Input(t, repo, archive)
	versions := regexp.MustCompile(`GLIBC_([0-9]+)\.([0-9]+)`).FindAllSubmatch
	found := false
	tars := tar.NewReader(stream)
	for {
		header, err := tars.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tars)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range versions(data, -1) {
			found = true
			major, err := strconv.Atoi(string(match[1]))
			if err != nil {
				t.Fatal(err)
			}
			minor, err := strconv.Atoi(string(match[2]))
			if err != nil {
				t.Fatal(err)
			}
			if major > 2 || major == 2 && minor > 39 {
				t.Fatalf("%s requires GLIBC_%d.%d", header.Name, major, minor)
			}
		}
	}
	if !found {
		t.Fatal("LLVM 22 GLIBC requirements not found")
	}
	if _, err := io.Copy(io.Discard, stream); err != nil {
		t.Fatal(err)
	}
}

func TestLLVM22(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("需要Linux amd64宿主执行LLVM安装与链接验证")
	}
	spec := payloadSpec(t, "llvm22")
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	root := os.Getenv("CDH_LLVM22_TEST_ROOT")
	if root == "" {
		dest := filepath.Join(t.TempDir(), "tools")
		if err := install.Run(context.Background(), repo, dest, spec); err != nil {
			t.Fatal(err)
		}
		root = filepath.Join(dest, "llvm22")
	}
	if filepath.Base(root) != "llvm22" {
		t.Fatalf("invalid LLVM 22 root: %q", root)
	}
	llvm22Dir(t, root)
	if err := install.Check(context.Background(), filepath.Dir(root), spec); err != nil {
		t.Fatal(err)
	}
	hostLibs := map[string]bool{
		"ld-linux-x86-64.so.2": true,
		"libbsd.so.0":          true,
		"libc.so.6":            true,
		"libcom_err.so.2":      true,
		"libcrypto.so.3":       true,
		"libgcc_s.so.1":        true,
		"libgmp.so.10":         true,
		"libm.so.6":            true,
		"libmd.so.0":           true,
		"libresolv.so.2":       true,
		"libssl.so.3":          true,
		"libstdc++.so.6":       true,
		"libtinfo.so.6":        true,
		"libz.so.1":            true,
		"libzstd.so.1":         true,
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
		if machine != elf.EM_X86_64 {
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
		if machine != elf.EM_X86_64 || kind != elf.ET_EXEC {
			t.Fatalf("%s machine=%v type=%v", item.name, machine, kind)
		}
	}
}

func TestNDK29(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("需要Linux amd64宿主执行NDK安装与链接验证")
	}
	spec := payloadSpec(t, "ndk29")
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	dest := filepath.Join(t.TempDir(), "tools")
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
	host := filepath.Join(root, "toolchains", "llvm", "prebuilt", "linux-x86_64")
	for _, rel := range []string{
		"sysroot/usr/include/aarch64-linux-android",
		"sysroot/usr/lib/aarch64-linux-android",
		"lib/clang/21/lib/linux",
	} {
		if info, statErr := os.Stat(filepath.Join(host, rel)); statErr != nil || !info.IsDir() {
			t.Fatalf("required directory %s: %v", rel, statErr)
		}
	}
	for _, name := range []string{"xt_rateest.h", "xt_RATEEST.h"} {
		pathName := filepath.Join(host, "sysroot", "usr", "include", "linux", "netfilter", name)
		if info, statErr := os.Stat(pathName); statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("case-sensitive header %s: %v", name, statErr)
		}
	}
	for _, triple := range []string{
		"arm-linux-androideabi",
		"i686-linux-android",
		"riscv64-linux-android",
		"x86_64-linux-android",
	} {
		for _, parent := range []string{"sysroot/usr/include", "sysroot/usr/lib"} {
			pathName := filepath.Join(host, parent, triple)
			if _, statErr := os.Lstat(pathName); !os.IsNotExist(statErr) {
				t.Fatalf("non-arm64 target path %s: %v", pathName, statErr)
			}
		}
	}
	for _, arch := range []string{"arm", "i386", "riscv64", "x86_64"} {
		pathName := filepath.Join(host, "lib", "clang", "21", "lib", "linux", arch)
		if _, statErr := os.Lstat(pathName); !os.IsNotExist(statErr) {
			t.Fatalf("non-arm64 runtime directory %s: %v", pathName, statErr)
		}
	}
	wantBin := map[string]bool{}
	for _, name := range []string{
		"clang", "clang++", "clang-21", "ld", "ld.lld", "lld",
		"llvm-addr2line", "llvm-ar", "llvm-as", "llvm-cov", "llvm-cxxfilt",
		"llvm-dis", "llvm-dwp", "llvm-link", "llvm-nm", "llvm-objcopy",
		"llvm-objdump", "llvm-profdata", "llvm-ranlib", "llvm-readelf",
		"llvm-readobj", "llvm-size", "llvm-strings", "llvm-strip", "llvm-symbolizer",
	} {
		wantBin[name] = true
	}
	for api := 21; api <= 35; api++ {
		wantBin[fmt.Sprintf("aarch64-linux-android%d-clang", api)] = true
		wantBin[fmt.Sprintf("aarch64-linux-android%d-clang++", api)] = true
	}
	binDir := filepath.Join(host, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(wantBin) {
		t.Fatalf("NDK bin entries=%d want=%d", len(entries), len(wantBin))
	}
	for _, entry := range entries {
		if !wantBin[entry.Name()] {
			t.Errorf("unexpected NDK tool=%s", entry.Name())
		}
	}
	makeDir := filepath.Join(root, "prebuilt", "linux-x86_64", "bin")
	entries, err = os.ReadDir(makeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "make" {
		t.Fatalf("NDK prebuilt bin=%v", entries)
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
	hostLibs := map[string]bool{
		"ld-linux-x86-64.so.2": true,
		"libc.so.6":            true,
		"libdl.so.2":           true,
		"libgcc_s.so.1":        true,
		"libm.so.6":            true,
		"libpthread.so.0":      true,
		"librt.so.1":           true,
		"libz.so.1":            true,
	}
	hostCount := 0
	targetCount := 0
	err = filepath.Walk(root, func(pathName string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		mode := info.Mode()
		if mode.IsDir() && mode.Perm() != 0o755 {
			t.Errorf("directory mode %s=%#o", pathName, mode.Perm())
		}
		if mode.IsRegular() && mode.Perm() != 0o644 && mode.Perm() != 0o755 {
			t.Errorf("file mode %s=%#o", pathName, mode.Perm())
		}
		if !mode.IsRegular() {
			return nil
		}
		binary, openErr := elf.Open(pathName)
		if openErr != nil {
			return nil
		}
		machine := binary.Machine
		libs, libErr := binary.ImportedLibraries()
		closeErr := binary.Close()
		if closeErr != nil {
			return closeErr
		}
		switch machine {
		case elf.EM_X86_64:
			hostCount++
			hostPath := strings.HasPrefix(pathName, binDir+string(filepath.Separator)) ||
				strings.HasPrefix(pathName, makeDir+string(filepath.Separator)) ||
				pathName == filepath.Join(host, "lib", "libxml2.so.2")
			if !hostPath {
				return fmt.Errorf("unexpected host ELF: %s", pathName)
			}
		case elf.EM_AARCH64:
			targetCount++
			return nil
		default:
			return fmt.Errorf("unexpected ELF machine %s=%v", pathName, machine)
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
	if hostCount != 19 || targetCount == 0 {
		t.Fatalf("host ELF=%d target ELF=%d", hostCount, targetCount)
	}
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "clang", want: "clang version 21.0.0"},
		{name: "ld.lld", want: "LLD 21.0.0"},
		{name: "llvm-ar", want: "LLVM version 21.0.0"},
	} {
		program := filepath.Join(binDir, item.name)
		output, runErr := exec.Command(program, "--version").CombinedOutput()
		if runErr != nil || !strings.Contains(string(output), item.want) {
			t.Fatalf("%s version: %v: %s", item.name, runErr, output)
		}
	}
	output, err := exec.Command(filepath.Join(root, "ndk-build"), "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(output), "GNU Make 4.3") {
		t.Fatalf("ndk-build version: %v: %s", err, output)
	}
	probe := t.TempDir()
	cSource := filepath.Join(probe, "probe.c")
	cBody := "#include <linux/netfilter/xt_rateest.h>\n#include <linux/netfilter/xt_RATEEST.h>\nint main(void) { return 0; }\n"
	if err := os.WriteFile(cSource, []byte(cBody), 0o644); err != nil {
		t.Fatal(err)
	}
	clang := filepath.Join(binDir, "aarch64-linux-android35-clang")
	object := filepath.Join(probe, "probe.o")
	output, err = exec.Command(clang, "-c", cSource, "-o", object).CombinedOutput()
	if err != nil {
		t.Fatalf("C compile: %v: %s", err, output)
	}
	library := filepath.Join(probe, "libprobe.a")
	ar := filepath.Join(binDir, "llvm-ar")
	output, err = exec.Command(ar, "rcs", library, object).CombinedOutput()
	if err != nil {
		t.Fatalf("archive: %v: %s", err, output)
	}
	output, err = exec.Command(ar, "t", library).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "probe.o" {
		t.Fatalf("archive members: %v: %s", err, output)
	}
	cProgram := filepath.Join(probe, "probe-c")
	output, err = exec.Command(clang, "-fuse-ld=lld", object, "-o", cProgram).CombinedOutput()
	if err != nil {
		t.Fatalf("C link: %v: %s", err, output)
	}
	ccSource := filepath.Join(probe, "probe.cc")
	ccBody := "#include <vector>\nint main() { std::vector<int> value{0}; return value[0]; }\n"
	if err := os.WriteFile(ccSource, []byte(ccBody), 0o644); err != nil {
		t.Fatal(err)
	}
	ccProgram := filepath.Join(probe, "probe-cc")
	clangCC := filepath.Join(binDir, "aarch64-linux-android35-clang++")
	output, err = exec.Command(clangCC, "-fuse-ld=lld", ccSource, "-o", ccProgram).CombinedOutput()
	if err != nil {
		t.Fatalf("C++ link: %v: %s", err, output)
	}
	programs := []string{cProgram, ccProgram}
	if mainRoot := os.Getenv("CDH_MAIN_ROOT"); mainRoot != "" {
		mainSource := filepath.Join(mainRoot, "tools", "android", "getaddrinfo-probe.c")
		mainProgram := filepath.Join(probe, "getaddrinfo-probe")
		output, err = exec.Command(clang, "-fuse-ld=lld", mainSource, "-o", mainProgram).CombinedOutput()
		if err != nil {
			t.Fatalf("hub target link: %v: %s", err, output)
		}
		programs = append(programs, mainProgram)
	}
	project := filepath.Join(probe, "ndk-project")
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
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ndk-build: %v: %s", err, output)
	}
	programs = append(programs, filepath.Join(project, "libs", "arm64-v8a", "ndkprobe"))
	for _, program := range programs {
		binary, openErr := elf.Open(program)
		if openErr != nil {
			t.Fatal(openErr)
		}
		machine := binary.Machine
		kind := binary.Type
		interp := binary.Section(".interp")
		if machine != elf.EM_AARCH64 || kind != elf.ET_DYN || interp == nil {
			binary.Close()
			t.Fatalf("Android ELF %s machine=%v type=%v interp=%v", program, machine, kind, interp)
		}
		data, readErr := interp.Data()
		closeErr := binary.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("Android ELF %s: %v / %v", program, readErr, closeErr)
		}
		if got := strings.TrimRight(string(data), "\x00"); got != "/system/bin/linker64" {
			t.Fatalf("Android ELF %s interpreter=%q", program, got)
		}
	}
}

func TestBazel(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("需要Linux amd64宿主执行Bazel安装与运行验证")
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	spec := payloadSpec(t, "bazel851")
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
	if bin.Machine != elf.EM_X86_64 {
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("需要Linux amd64宿主执行Rust安装与编译验证")
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	spec := payloadSpec(t, "rust197")
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
