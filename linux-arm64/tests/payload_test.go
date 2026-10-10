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
	spec := catalog.Current()
	if spec.Host != "linux-arm64" || !spec.Ready || len(spec.Archives) < 1 {
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
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
	if strings.TrimSpace(string(output)) != "go version go1.26.8 linux/arm64" {
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
	spec := catalog.Current()
	if len(spec.Archives) < 2 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	archive := spec.Archives[1]
	if archive.Name != "JDK" || archive.Version != "21.0.12.1+1" || archive.Dir != "jdk21trim1" {
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
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
	root := filepath.Join(dest, "jdk21trim1")
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
	spec := catalog.Current()
	if len(spec.Archives) < 4 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	archive := spec.Archives[3]
	if archive.Name != "LLVM 22" || archive.Version != "22.1.2" || archive.Dir != "llvm22" {
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
		{name: "PACKAGES.md", want: "af216aa392fb2f4a739e983c546ac030cb9ef1f6104896f519effec289ade032"},
		{name: "libbrotli-copyright", want: "24a64e5bb83d0960d1835696a1e23c0896ad6055b0ca47c66ab0eb9a766324b1"},
		{name: "libcurl-copyright", want: "b36986270d288caa9f7c44876802de7a491139f3903dcb6356707ece08e1df4a"},
		{name: "libedit-copyright", want: "2b53671050f6319be6fcdcafa434e8ccb8af629359b4afc83185867204f3927d"},
		{name: "libffi-copyright", want: "25ee86cfccc2b7eb9a156a127f1e1b38c46c231f516ff201912223eaf62ee186"},
		{name: "libgnutls-copyright", want: "b32655aad8bb1530de36933f8702af1e669fabb9bf3e712c533803c7092d77da"},
		{name: "libgssapi-copyright", want: "936728f4181718f42951b881c1e8f1386bf6b2723c4fbc533c374d6f42c71816"},
		{name: "libhogweed-copyright", want: "344396197142cd506468b20029ea5722d507c980129bb3db4af02424ed6f39c6"},
		{name: "libidn2-copyright", want: "a63ceb8fab281177da9a27403f0be7ae69aa6f33bf6f63c455ecc587ca131268"},
		{name: "libk5crypto-copyright", want: "936728f4181718f42951b881c1e8f1386bf6b2723c4fbc533c374d6f42c71816"},
		{name: "libkeyutils-copyright", want: "73f5682b0466f41cea2b9b7f1ff8a8052464f6c09fb29eeac4490512af09bd50"},
		{name: "libkrb5-copyright", want: "936728f4181718f42951b881c1e8f1386bf6b2723c4fbc533c374d6f42c71816"},
		{name: "libkrb5support-copyright", want: "936728f4181718f42951b881c1e8f1386bf6b2723c4fbc533c374d6f42c71816"},
		{name: "libldap-copyright", want: "2656a8ac4f2fda614be8fe6fd7f649665a6ca3abcf19f6ac125e2ff165b60939"},
		{name: "libnettle-copyright", want: "344396197142cd506468b20029ea5722d507c980129bb3db4af02424ed6f39c6"},
		{name: "libnghttp2-copyright", want: "a4d7f9d9d350e7f89aa0688cea74838f33147462d68a3542290387101bf4f274"},
		{name: "libp11-kit-copyright", want: "a5611fbad151d1fa0aa484f6c393a5166af33ce9a9783dfb2712d6b87b462b37"},
		{name: "libpfm-copyright", want: "ef51ef5cf53a26a1eda506137c5864156c52724da7d9d8cae3c9903c7ef43e28"},
		{name: "libpsl-copyright", want: "bbafb0cf52d34465fc69f5ffd22d724224f6a2f28402c88b045c69d88af06aaf"},
		{name: "librtmp-copyright", want: "2aa6df2ec29edf3e26571e89bda27f0c3a3f187cdf08f75dcc6a7ce32d236824"},
		{name: "libsasl2-copyright", want: "ddb128a81d3a140ef5b3574ae000a1a50d0cb9a2ec59917121a155f91b670166"},
		{name: "libssh2-copyright", want: "fb5d6b7a53710b6ba04e412a016d222a3e56b9d6e1a2572a8ffc2eae05e6acfb"},
		{name: "libtasn1-copyright", want: "722a2780f3f3865c8bed997b8a6283eebf302c9fe4a5b2fb5fd221ea9a51a5a6"},
		{name: "libunistring-copyright", want: "a585cd2cc2a5fcb8e58f12c7a710a35e06e5b897c59cd5ae4a7d835f0b49eae3"},
		{name: "libxml2-copyright", want: "ee746b96cfa5be73c3ea3e4cfb1285e9b315d4c9267f99b2ee9c5d911d9fe3f4"},
		{name: "libz3-copyright", want: "807f874c81a0dbce7d4e32b1a4e1ef1e094b9a3fe8579f34e60fda4f2077289c"},
		{name: "llvm-copyright", want: "b2ed2fd344e8ddbf2e0c83113bd40fa1853438be61a0ff20af5acd187c07cbf6"},
	} {
		pathName := filepath.Join(repo, "licenses", "llvm22", item.name)
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		return
	}
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
	spec := catalog.Current()
	if len(spec.Archives) != 8 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	ndk := spec.Archives[4]
	if ndk.Name != "Android NDK" || ndk.Version != "29.0.14206865" || ndk.Dir != "ndk29" {
		t.Fatalf("archive=%+v", ndk)
	}
	if !ndk.DataOnly || len(ndk.Probes) != 0 {
		t.Fatalf("data-only=%v probes=%d", ndk.DataOnly, len(ndk.Probes))
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
		{name: "NOTICE", want: "faf0dc52cf6ce9ab09a04b0c074a378dd2565ee1ed9bc1179d83793bcdba012e"},
		{name: "NOTICE.toolchain", want: "789714a9076571c580cf531bdcebec776a4d889c12c3c1108adafc6e4338710a"},
		{name: "SOURCE.md", want: "44de4e3cf43bea650f6ee2938e9ec332f2d55bf9d3ada455b84a708217b55287"},
	} {
		pathName := filepath.Join(repo, "licenses", "ndk29", item.name)
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		return
	}
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
	if len(spec.Archives) != 8 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	bzl := spec.Archives[5]
	if bzl.Name != "Bazel" || bzl.Version != "8.5.1" || bzl.Dir != "bazel851" {
		t.Fatalf("archive=%+v", bzl)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		return
	}
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
	if len(spec.Archives) != 8 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	rust := spec.Archives[6]
	if rust.Name != "Rust" || rust.Version != "1.97.1" || rust.Dir != "rust197" {
		t.Fatalf("archive=%+v", rust)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		return
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
