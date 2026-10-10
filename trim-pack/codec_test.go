// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func smallTar(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, h := range []tar.Header{
		{Name: "up/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "up/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "up/bin/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: 4},
		{Name: "up/bin/alias", Typeflag: tar.TypeSymlink, Mode: 0o755, Linkname: "tool"},
	} {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := io.WriteString(tw, "tool"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTarCodec(t *testing.T) {
	for _, kind := range []string{"tar.gz", "tar.xz"} {
		t.Run(kind, func(t *testing.T) {
			dir := testDir(t)
			var buf bytes.Buffer
			var wr io.WriteCloser
			if kind == "tar.gz" {
				wr = gzip.NewWriter(&buf)
			} else {
				var err error
				wr, err = xz.NewWriter(&buf)
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := wr.Write(smallTar(t)); err != nil {
				t.Fatal(err)
			}
			if err := wr.Close(); err != nil {
				t.Fatal(err)
			}
			name := "source." + kind
			if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			in := input{Name: name, Kind: kind, Prefix: "up", SHA256: fmt.Sprintf("%x", sha256.Sum256(buf.Bytes())), Rules: []rule{{Tree: true}}}
			plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
			dest := filepath.Join(dir, "out")
			if err := generate(context.Background(), plan, dir, dest, io.Discard); err != nil {
				t.Fatal(err)
			}
			files := readFiles(t, dest)
			if files["sample/bin/alias"] != "tool" || files["sample/bin/tool"] != "tool" {
				t.Fatalf("别名或原始文件字节错误：%v", files)
			}
			if err := verify(context.Background(), plan, dir, dest, "", io.Discard); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDebCodec(t *testing.T) {
	dir := testDir(t)
	var compressed bytes.Buffer
	zr, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := zr.Write(smallTar(t)); err != nil {
		t.Fatal(err)
	}
	if err := zr.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.WriteString("!<arch>\n")
	for _, member := range []struct {
		name string
		body []byte
	}{{"debian-binary", []byte("2.0\n")}, {"data.tar.zst", compressed.Bytes()}} {
		fmt.Fprintf(&buf, "%-16s%-12d%-6d%-6d%-8o%-10d`\n", member.name+"/", 0, 0, 0, 0o644, len(member.body))
		buf.Write(member.body)
		if len(member.body)%2 != 0 {
			buf.WriteByte('\n')
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "source.deb"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	in := input{Name: "source.deb", Kind: "deb", Prefix: "up", SHA256: fmt.Sprintf("%x", sha256.Sum256(buf.Bytes())),
		Rules: []rule{{From: "bin", To: "bin", Tree: true}}}
	plan := spec{Host: "linux-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
	dest := filepath.Join(dir, "out")
	if err := generate(context.Background(), plan, dir, dest, io.Discard); err != nil {
		t.Fatal(err)
	}
	if files := readFiles(t, dest); files["sample/bin/tool"] != "tool" || files["sample/bin/alias"] != "tool" {
		t.Fatalf("DEB未保留原始程序与别名：%v", files)
	}
}

func TestMetadata(t *testing.T) {
	pkg := pkgDef{Path: "cmake;4.1.2", Kind: "generic", Version: "4.1.2", Display: "CMake 4.1.2"}
	if _, err := packageXML(pkg, ""); err == nil {
		t.Fatal("没有原始许可证仍生成元数据")
	}
	body, err := packageXML(pkg, "  <license id=\"android-sdk-license\" type=\"text\">original license</license>\n")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		License struct {
			ID   string `xml:"id,attr"`
			Text string `xml:",chardata"`
		} `xml:"license"`
		Package struct {
			Path string `xml:"path,attr"`
			Rev  struct {
				Major int `xml:"major"`
				Minor int `xml:"minor"`
				Micro int `xml:"micro"`
			} `xml:"revision"`
		} `xml:"localPackage"`
	}
	if err := xml.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}
	if got.License.Text != "original license" || got.License.ID != "android-sdk-license" || got.Package.Path != pkg.Path ||
		got.Package.Rev.Major != 4 || got.Package.Rev.Minor != 1 || got.Package.Rev.Micro != 2 {
		t.Fatalf("元数据版本或原始许可不符：%+v", got)
	}
}

func TestPublished(t *testing.T) {
	dir := testDir(t)
	in := makeSource(t, dir)
	plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
	dest := filepath.Join(dir, "new")
	if err := generate(context.Background(), plan, dir, dest, io.Discard); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dir, "old")
	if err := os.Mkdir(old, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(gw)
	for _, h := range []tar.Header{
		{Name: "sample/", Mode: 0o755, Typeflag: tar.TypeDir},
		{Name: "sample/bin/", Mode: 0o755, Typeflag: tar.TypeDir},
		{Name: "sample/bin/tool", Mode: 0o755, Typeflag: tar.TypeReg, Size: int64(len("original program"))},
		{Name: "sample/LICENSE", Mode: 0o644, Typeflag: tar.TypeReg, Size: int64(len("original license"))},
	} {
		h.Uid, h.Gid, h.ModTime = 1200, 1300, time.Unix(1234567, 0)
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			body := "original program"
			if h.Name == "sample/LICENSE" {
				body = "original license"
			}
			if _, err := io.WriteString(tw, body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	plan.OldSHA = fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	if err := os.WriteFile(filepath.Join(old, "old.tar.gz.part-000"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), plan, dir, dest, old, io.Discard); err != nil {
		t.Fatal(err)
	}
	plan.OldSHA = "wrong"
	if err := verify(context.Background(), plan, dir, dest, old, io.Discard); err == nil {
		t.Fatal("旧归档未绑定登记摘要")
	}
}
