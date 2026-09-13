// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"archive/tar"
	"bufio"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
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
	"github.com/ulikunitz/xz"
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
	if len(spec.Archives) < 2 {
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

func TestLLVM19(t *testing.T) {
	spec := catalog.Current()
	if len(spec.Archives) < 3 {
		t.Fatalf("archives=%d", len(spec.Archives))
	}
	archive := spec.Archives[2]
	if archive.Name != "LLVM 19" || archive.Version != "19.1.7" || archive.Dir != "llvm19" {
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
		{name: "DEPENDENCIES.md", want: "4859c6f0c3b66f11d33c4eec75fa498a9e1186360a7fb42089ca821f5638d501"},
		{name: "LICENSE.TXT", want: "8d85c1057d742e597985c7d4e6320b015a9139385cff4cbae06ffc0ebe89afee"},
		{name: "icu-copyright", want: "5bbff5f646ce750ac0a347e7f91793279b7422a1119820b5f75d98879c201c68"},
		{name: "libxml2-copyright", want: "ee746b96cfa5be73c3ea3e4cfb1285e9b315d4c9267f99b2ee9c5d911d9fe3f4"},
	} {
		pathName := filepath.Join(repo, "licenses", "llvm19", item.name)
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
	spec.Archives = []catalog.Archive{archive}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dest, "llvm19")
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
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "clang", want: "clang version 19.1.7"},
		{name: "ld.lld", want: "LLD 19.1.7"},
		{name: "llvm-ar", want: "LLVM version 19.1.7"},
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
		if strings.Contains(string(output), "not found") {
			t.Fatalf("%s dependencies=%q", item.name, output)
		}
		if item.name == "ld.lld" {
			for _, want := range []string{
				"/lib/cdh/libxml2.so.2",
				"/lib/cdh/libicuuc.so.74",
				"/lib/cdh/libicudata.so.74",
			} {
				if !strings.Contains(string(output), want) {
					t.Fatalf("ld.lld dependencies miss %q: %s", want, output)
				}
			}
		}
	}
	probe := t.TempDir()
	object := filepath.Join(probe, "probe.o")
	source := filepath.Join(probe, "probe.c")
	if err := os.WriteFile(source, []byte("void _start(void) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(filepath.Join(root, "bin", "clang"), "-c", source, "-o", object).CombinedOutput()
	if err != nil {
		t.Fatalf("compile: %v: %s", err, output)
	}
	archivePath := filepath.Join(probe, "probe.a")
	output, err = exec.Command(filepath.Join(root, "bin", "llvm-ar"), "rcs", archivePath, object).CombinedOutput()
	if err != nil {
		t.Fatalf("archive: %v: %s", err, output)
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("archive file is empty")
	}
	for _, item := range []struct {
		name string
		ext  string
		body string
		tool string
	}{
		{name: "c", ext: ".c", body: "void _start(void) {}\n", tool: "clang"},
		{name: "cc", ext: ".cc", body: "extern \"C\" void _start() {}\n", tool: "clang++"},
	} {
		source := filepath.Join(probe, item.name+item.ext)
		if err := os.WriteFile(source, []byte(item.body), 0o644); err != nil {
			t.Fatal(err)
		}
		program := filepath.Join(probe, item.name)
		output, err := exec.Command(filepath.Join(root, "bin", item.tool), "-fuse-ld=lld", "-nostdlib", "-no-pie", "-Wl,-e,_start", source, "-o", program).CombinedOutput()
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

func TestLLVM22GLIBC(t *testing.T) {
	spec := catalog.Current()
	archive := spec.Archives[3]
	if len(archive.Parts) != 1 {
		t.Fatalf("LLVM 22 parts=%d", len(archive.Parts))
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	part, err := os.Open(filepath.Join(filepath.Dir(work), filepath.FromSlash(archive.Parts[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	defer part.Close()
	stream, err := xz.NewReader(bufio.NewReader(part))
	if err != nil {
		t.Fatal(err)
	}
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
		{name: "PACKAGES.md", want: "0b4f8fb97a0f9b013112f7ba2fee9577deea830306a0cb5598b2c3583686a3d5"},
		{name: "libbrotli-copyright", want: "24a64e5bb83d0960d1835696a1e23c0896ad6055b0ca47c66ab0eb9a766324b1"},
		{name: "libcurl-copyright", want: "b36986270d288caa9f7c44876802de7a491139f3903dcb6356707ece08e1df4a"},
		{name: "libedit-copyright", want: "2b53671050f6319be6fcdcafa434e8ccb8af629359b4afc83185867204f3927d"},
		{name: "libffi-copyright", want: "25ee86cfccc2b7eb9a156a127f1e1b38c46c231f516ff201912223eaf62ee186"},
		{name: "libgnutls-copyright", want: "51da5214308b87d6a0a2fbcaea1f7dbd15510be733e5921d9548ff6b88d84dd8"},
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
		{name: "libssl-copyright", want: "6a7da622fe0637a334d2a8fc470852d2ffb77d9a2b2f930f854e32a41ad6ef35"},
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
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return
	}
	spec.Archives = []catalog.Archive{archive}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dest, "llvm22")
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
	spec := catalog.Current()
	if len(spec.Archives) != 5 {
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
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "NOTICE", want: "faf0dc52cf6ce9ab09a04b0c074a378dd2565ee1ed9bc1179d83793bcdba012e"},
		{name: "NOTICE.toolchain", want: "789714a9076571c580cf531bdcebec776a4d889c12c3c1108adafc6e4338710a"},
	} {
		pathName := filepath.Join(repo, "licenses", "ndk29", item.name)
		data, readErr := os.ReadFile(pathName)
		if readErr != nil {
			t.Fatal(readErr)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != item.want {
			t.Fatalf("%s SHA-256=%s", item.name, got)
		}
		info, statErr := os.Stat(pathName)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("%s mode=%#o", item.name, info.Mode().Perm())
		}
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return
	}
	spec.Archives = []catalog.Archive{ndk}
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
