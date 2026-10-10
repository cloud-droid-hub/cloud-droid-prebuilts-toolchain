// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"context"
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestVerifySame(t *testing.T) {
	name := filepath.Join(t.TempDir(), "same.tar.xz")
	writeXZ(t, name, testTar(t, []tar.Header{{Name: "llvm/LICENSE", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4}}))
	if err := verify(name, name, "llvm"); err == nil {
		t.Fatal("the same input and output file was accepted")
	}
}

type pathMeta struct {
	os.FileInfo
	link bool
}

func (i pathMeta) Mode() os.FileMode {
	if i.link {
		return os.ModeSymlink | 0o777
	}
	return i.FileInfo.Mode()
}

func (i pathMeta) Sys() any {
	st := *i.FileInfo.Sys().(*syscall.Stat_t)
	if !i.link {
		st.Nlink = 2
	}
	return &st
}

func TestArcIdentity(t *testing.T) {
	name := filepath.Join(t.TempDir(), "input.tar.xz")
	writeXZ(t, name, testTar(t, []tar.Header{{Name: "llvm/LICENSE", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4}}))
	prior := pathInfo
	t.Cleanup(func() { pathInfo = prior })
	for _, link := range []bool{false, true} {
		pathInfo = func(p string) (os.FileInfo, error) {
			info, err := prior(p)
			if err == nil && p == name {
				return pathMeta{FileInfo: info, link: link}, nil
			}
			return info, err
		}
		f, err := arcInput(name)
		if err == nil {
			f.Close()
			t.Errorf("invalid archive input metadata was accepted: symbolic=%t", link)
		}
	}
}

func hostImage(t *testing.T, foreign bool) []byte {
	t.Helper()
	if runtime.GOOS == "darwin" {
		cpu := uint32(macho.CpuArm64)
		if foreign {
			cpu = uint32(macho.CpuAmd64)
		}
		data := make([]byte, 32)
		binary.LittleEndian.PutUint32(data, macho.Magic64)
		binary.LittleEndian.PutUint32(data[4:], cpu)
		binary.LittleEndian.PutUint32(data[12:], uint32(macho.TypeExec))
		return data
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		t.Fatal("unsupported host for architecture validation")
	}
	machine := uint16(elf.EM_AARCH64)
	if runtime.GOARCH == "amd64" {
		machine = uint16(elf.EM_X86_64)
	}
	if foreign {
		if machine == uint16(elf.EM_AARCH64) {
			machine = uint16(elf.EM_X86_64)
		} else {
			machine = uint16(elf.EM_AARCH64)
		}
	}
	data := make([]byte, 64)
	copy(data, []byte{'\x7f', 'E', 'L', 'F', byte(elf.ELFCLASS64), byte(elf.ELFDATA2LSB), 1})
	binary.LittleEndian.PutUint16(data[16:], uint16(elf.ET_EXEC))
	binary.LittleEndian.PutUint16(data[18:], machine)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint16(data[52:], 64)
	return data
}

func TestNativeGuard(t *testing.T) {
	prior := runNative
	t.Cleanup(func() { runNative = prior })
	called := false
	runNative = func(context.Context, []string, string, ...string) ([]byte, error) {
		called = true
		return []byte("process boundary reached"), nil
	}
	for _, foreign := range []bool{true, false} {
		name := filepath.Join(t.TempDir(), "compiler")
		if err := os.WriteFile(name, hostImage(t, foreign), 0o755); err != nil {
			t.Fatal(err)
		}
		called = false
		_, err := nativeCmd(t.Context(), nil, name)
		if foreign && (err == nil || called) {
			t.Errorf("foreign architecture reached execution: called=%t error=%v", called, err)
		}
		if !foreign && (err != nil || !called) {
			t.Errorf("native architecture was rejected: called=%t error=%v", called, err)
		}
	}
}
