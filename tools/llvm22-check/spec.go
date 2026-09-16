// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/ulikunitz/xz"
)

func partPaths(dir string) ([]string, error) {
	items, err := os.ReadDir(dir)
	if err != nil || len(items) == 0 {
		return nil, fmt.Errorf("missing parts: %s", dir)
	}
	var paths []string
	stem := strings.TrimSuffix(items[0].Name(), ".tar.xz.part-000")
	if stem == "" || stem == items[0].Name() {
		return nil, fmt.Errorf("first part must end in .tar.xz.part-000")
	}
	for i, item := range items {
		want := fmt.Sprintf("%s.tar.xz.part-%03d", stem, i)
		if item.Name() != want || !item.Type().IsRegular() {
			return nil, fmt.Errorf("unexpected payload entry %q, want regular %q", item.Name(), want)
		}
		paths = append(paths, filepath.Join(dir, item.Name()))
	}
	return paths, nil
}

type hostCfg struct {
	host    string
	goos    string
	arch    string
	version string
	lld     string
	machine catalog.Machine
}

func hostConfig(host string) (hostCfg, error) {
	switch host {
	case "darwin-arm64":
		return hostCfg{host, "darwin", "arm64", "22.1.8", "ld64.lld", catalog.MachARM64}, nil
	case "linux-arm64":
		return hostCfg{host, "linux", "arm64", "22.1.2", "ld.lld", catalog.ELFARM64}, nil
	case "linux-amd64":
		return hostCfg{host, "linux", "amd64", "22.1.2", "ld.lld", catalog.ELFAMD64}, nil
	default:
		return hostCfg{}, fmt.Errorf("unsupported native host: %s", host)
	}
}

type packInfo struct {
	spec catalog.Spec
	need uint64
	ids  idSet
}

func makeSpec(cfg hostCfg, dir, prefix string, log io.Writer) (packInfo, error) {
	var out packInfo
	if err := realDir(dir); err != nil {
		return out, err
	}
	paths, err := partPaths(dir)
	if err != nil {
		return out, err
	}
	a := catalog.Archive{Name: "LLVM 22", Version: cfg.version, Dir: "llvm22", Format: catalog.TarXZ, Prefix: prefix}
	full := sha256.New()
	out.ids = make(idSet)
	for _, name := range paths {
		f, info, err := openPlain(name)
		if err != nil {
			return out, err
		}
		if err := out.ids.add(name, info); err != nil {
			f.Close()
			return out, err
		}
		hash := sha256.New()
		n, err := io.Copy(io.MultiWriter(full, hash), f)
		check := stableFile(f, info)
		closeErr := f.Close()
		if err != nil || check != nil || closeErr != nil || n != info.Size() {
			return out, fmt.Errorf("read part %s: %v %v %v", name, err, check, closeErr)
		}
		sum := hex.EncodeToString(hash.Sum(nil))
		a.Parts = append(a.Parts, catalog.Part{Path: filepath.Base(name), Size: n, SHA256: sum})
		if err := writeLog(log, "part=%q bytes=%d sha256=%s\n", name, n, sum); err != nil {
			return out, err
		}
	}
	a.SHA256 = hex.EncodeToString(full.Sum(nil))
	clang := "clang version " + cfg.version
	lld := "LLD " + cfg.version
	ar := "LLVM version " + cfg.version
	for _, item := range [][2]string{{"clang", clang}, {cfg.lld, lld}, {"llvm-ar", ar}} {
		a.Probes = append(a.Probes, catalog.Probe{Path: "bin/" + item[0], Args: []string{"--version"}, Want: []string{item[1]}, Machine: cfg.machine})
	}
	out.spec = catalog.Spec{Host: cfg.host, Ready: true, Archives: []catalog.Archive{a}}
	if err := catalog.Valid(out.spec); err != nil {
		return out, err
	}
	out.need, err = scanParts(paths, prefix, a.SHA256)
	if err != nil {
		return out, err
	}
	err = writeLog(log, "archive_sha256=%s parts=%d install_reserve=%d\n", a.SHA256, len(a.Parts), out.need)
	return out, err
}

func scanParts(paths []string, prefix, sum string) (uint64, error) {
	var files []*os.File
	var infos []os.FileInfo
	var readers []io.Reader
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()
	for _, name := range paths {
		f, info, err := openPlain(name)
		if err != nil {
			return 0, err
		}
		files, infos, readers = append(files, f), append(infos, info), append(readers, f)
	}
	hash := sha256.New()
	xr, err := xz.NewReader(io.TeeReader(io.MultiReader(readers...), hash))
	if err != nil {
		return 0, err
	}
	need, err := scanTar(xr, prefix)
	if err != nil {
		return 0, err
	}
	if _, err := io.Copy(io.Discard, xr); err != nil {
		return 0, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != sum {
		return 0, fmt.Errorf("archive changed between digest and link checks")
	}
	for i, f := range files {
		if err := stableFile(f, infos[i]); err != nil {
			return 0, err
		}
	}
	return need, nil
}

type arcEntry struct {
	kind byte
	link string
}

func linkOK(name, target string) bool {
	return name == "bin/clang" && target == "clang-22" || name == "bin/ld.lld" && target == "lld"
}

func scanTar(r io.Reader, prefix string) (uint64, error) {
	if prefix == "" || prefix == "." || prefix == ".." || path.Base(prefix) != prefix || strings.ContainsAny(prefix, "\\\x00\r\n\t") {
		return 0, fmt.Errorf("archive prefix must be one directory name")
	}
	tr := tar.NewReader(r)
	items := make(map[string]arcEntry)
	var need uint64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		name := strings.TrimSuffix(h.Name, "/")
		if path.Clean(name) != name || strings.ContainsAny(name, "\\\x00\r\n\t") || name != prefix && !strings.HasPrefix(name, prefix+"/") {
			return 0, fmt.Errorf("invalid archive path: %q", h.Name)
		}
		if _, exists := items[name]; exists {
			return 0, fmt.Errorf("duplicate archive path: %s", name)
		}
		if h.Mode&^int64(0o777) != 0 || h.Size < 0 {
			return 0, fmt.Errorf("unsupported archive metadata: %s", name)
		}
		if name == prefix && h.Typeflag != tar.TypeDir {
			return 0, fmt.Errorf("archive root is not a directory")
		}
		switch h.Typeflag {
		case tar.TypeDir, tar.TypeSymlink:
			if h.Size != 0 {
				return 0, fmt.Errorf("non-file entry has content: %s", name)
			}
			if h.Typeflag == tar.TypeSymlink && !linkOK(strings.TrimPrefix(name, prefix+"/"), h.Linkname) {
				return 0, fmt.Errorf("unregistered archive link: %s", name)
			}
		case tar.TypeReg, tar.TypeRegA:
			h.Typeflag = tar.TypeReg
		default:
			return 0, fmt.Errorf("unsupported archive type %d: %s", h.Typeflag, name)
		}
		if uint64(h.Size) > ^uint64(0)-need-4096 {
			return 0, fmt.Errorf("archive size overflows")
		}
		need += uint64(h.Size) + 4096
		items[name] = arcEntry{h.Typeflag, h.Linkname}
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("empty archive")
	}
	for name, item := range items {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if node, ok := items[parent]; ok && node.kind != tar.TypeDir {
				return 0, fmt.Errorf("archive ancestor is not a directory: %s", parent)
			}
		}
		if item.kind == tar.TypeSymlink {
			target := path.Join(path.Dir(name), item.link)
			if node, ok := items[target]; !ok || node.kind != tar.TypeReg {
				return 0, fmt.Errorf("registered link target is not a regular file: %s", target)
			}
		}
	}
	return need, nil
}
