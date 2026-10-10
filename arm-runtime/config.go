package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func config(dest, llvm, sysroot string) error {
	for _, dir := range []string{dest, llvm, sysroot} {
		if !filepath.IsAbs(dir) || strings.ContainsAny(dir, "\x00\r\n'\\") {
			return fmt.Errorf("config requires absolute paths without quotes or control characters: %q", dir)
		}
	}
	if err := safeDir(filepath.Dir(dest)); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(dest), "arm-config-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := safeDir(filepath.Join(stage, "bin")); err != nil {
		return err
	}
	common := " --target=aarch64-linux-gnu --sysroot=" + shellQuote(sysroot) +
		" --gcc-toolchain=" + shellQuote(sysroot+"/usr")
	for _, pair := range [][2]string{{"clang", ""}, {"clang++", " --driver-mode=g++ --stdlib=libstdc++"}} {
		body := "#!/bin/sh\nset -eu\nlink=1\nfor arg do\n" +
			"  case \"$arg\" in -c|-S|-E|-fsyntax-only|-M|-MM|--version|--help|-print-*|--print-*|-dump*) link=0 ;; esac\n" +
			"done\nif [ \"$link\" = 1 ]; then\n  set -- --rtlib=libgcc --unwindlib=libgcc --ld-path=" +
			shellQuote(llvm+"/bin/ld.lld") + " \"$@\"\nfi\nexec " + shellQuote(llvm+"/bin/clang") + common + pair[1] + " \"$@\"\n"
		if err := writeText(filepath.Join(stage, "bin", pair[0]), body, 0o755); err != nil {
			return err
		}
	}
	pcdirs := strings.Join([]string{
		sysroot + "/opt/cloud-droid-hub/libslirp-static/lib/pkgconfig",
		sysroot + "/opt/cloud-droid-hub/pixman-static/lib/pkgconfig",
		sysroot + "/usr/lib/aarch64-linux-gnu/pkgconfig",
		sysroot + "/usr/share/pkgconfig",
	}, ":")
	body := "#!/bin/sh\nset -eu\nunset PKG_CONFIG_PATH\nexport PKG_CONFIG_SYSROOT_DIR=" + shellQuote(sysroot) +
		"\nexport PKG_CONFIG_LIBDIR=" + shellQuote(pcdirs) + "\nexec /usr/bin/pkg-config \"$@\"\n"
	if err := writeText(filepath.Join(stage, "bin", "pkg-config"), body, 0o755); err != nil {
		return err
	}
	body = fmt.Sprintf(`[binaries]
c = '%s/bin/clang'
cpp = '%s/bin/clang++'
ar = '%s/bin/llvm-ar'
strip = '%s/bin/llvm-strip'
pkg-config = '%s/bin/pkg-config'

[host_machine]
system = 'linux'
cpu_family = 'aarch64'
cpu = 'armv8-a'
endian = 'little'

[properties]
needs_exe_wrapper = true
sys_root = '%s'
pkg_config_libdir = '%s'
`, dest, dest, llvm, llvm, dest, sysroot, pcdirs)
	if err := writeText(filepath.Join(stage, "meson.ini"), body, 0o644); err != nil {
		return err
	}
	if err := os.Chmod(stage, 0o755); err != nil {
		return err
	}
	if err := publish(stage, dest); err != nil {
		return err
	}
	fmt.Printf("cc=%s/bin/clang cxx=%s/bin/clang++ host_cc=%s/bin/clang sysroot=%s pkg_config=%s/bin/pkg-config meson=%s/meson.ini\n", dest, dest, llvm, sysroot, dest, dest)
	return nil
}
