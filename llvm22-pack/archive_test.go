// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"io"
	"os"
	"testing"

	"github.com/ulikunitz/xz"
)

// 只读归档条目，避免为验证类型而恢复链接。
func TestArchive(t *testing.T) {
	name := os.Getenv("CDH_PACK_INPUT")
	prefix := os.Getenv("CDH_PACK_PREFIX")
	if name == "" || prefix == "" {
		t.Skip("set CDH_PACK_INPUT and CDH_PACK_PREFIX for archive verification")
	}
	f, err := arcInput(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	xr, err := xz.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(xr)
	files, links := 0, 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch h.Typeflag {
		case tar.TypeDir:
		case tar.TypeReg, tar.TypeRegA:
			files++
		case tar.TypeSymlink:
			links++
			allowed := h.Name == prefix+"/bin/clang" && h.Linkname == "clang-22" ||
				h.Name == prefix+"/bin/ld.lld" && h.Linkname == "lld"
			if !allowed {
				t.Errorf("unregistered archive link: %s -> %s", h.Name, h.Linkname)
			}
		default:
			t.Errorf("unsupported archive type %d: %s", h.Typeflag, h.Name)
		}
	}
	if _, err := io.Copy(io.Discard, xr); err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("archive has no regular files")
	}
	t.Logf("regular=%d symlinks=%d", files, links)
}

func arcInput(name string) (*os.File, error) {
	return openInput(name)
}
