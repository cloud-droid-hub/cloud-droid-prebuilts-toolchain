// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

func fixture(t *testing.T, extra ...tar.Header) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	names := []string{"python/install/bin/python3.14", "python/install/lib/python3.14/os.py",
		"python/install/lib/python3.14/venv/__init__.py", "python/install/lib/python3.14/ensurepip/__init__.py",
		"python/install/lib/python3.14/site-packages/pip/__init__.py", "python/licenses/LICENSE.cpython.txt"}
	var headers []tar.Header
	for _, name := range names {
		headers = append(headers, tar.Header{Name: name, Mode: 0o755, Size: 4})
	}
	for _, h := range append(headers, extra...) {
		if err := w.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(bytes.Repeat([]byte("x"), int(h.Size))); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTrim(t *testing.T) {
	in := fixture(t,
		tar.Header{Name: "python/build/obj.o", Size: 50},
		tar.Header{Name: "python/install/lib/python3.14/test/sample.py", Size: 30},
		tar.Header{Name: "python/install/lib/python3.14/__pycache__/os.pyc", Size: 20},
		tar.Header{Name: "python/install/share/man/man1/python.1", Size: 10},
		tar.Header{Name: "python/install/lib/libpython3.14.a", Typeflag: tar.TypeSymlink, Linkname: "python3.14/config-test/libpython3.14.a"},
		tar.Header{Name: "python/install/lib/python3.14/config-test/libpython3.14.a", Size: 100},
		tar.Header{Name: "python/install/bin/python3", Typeflag: tar.TypeSymlink, Linkname: "python3.14"},
		tar.Header{Name: "python/install/lib/python3.14/unittest/__init__.py", Size: 6},
		tar.Header{Name: "python/licenses/LICENSE.openssl.txt", Size: 8},
	)
	var out, replay bytes.Buffer
	rep, err := trim(context.Background(), sources[0], bytes.NewReader(in), &out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Removed != 6 || rep.Before-rep.After != 210 {
		t.Fatalf("裁剪统计不符：%+v", rep)
	}
	if _, err := trim(context.Background(), sources[0], bytes.NewReader(in), &replay); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), replay.Bytes()) {
		t.Fatal("同一输入未生成一致字节")
	}
	gz, err := gzip.NewReader(&out)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	r := tar.NewReader(gz)
	files := make(map[string]tar.Header)
	for {
		h, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		files[h.Name] = *h
	}
	if len(files) != 10 || files["python/bin/python3"].Linkname != "python3.14" ||
		files["python/bin/python3.14"].Mode != 0o755 || files["python/cdh-python-trim.txt"].Mode != 0o644 {
		t.Fatalf("归档布局、链接或权限不符：%v", files)
	}
	if _, ok := files["python/lib/python3.14/unittest/__init__.py"]; !ok {
		t.Fatal("运行模块unittest被裁掉")
	}
}

func TestReject(t *testing.T) {
	for _, h := range []tar.Header{
		{Name: "python/install/../escape"},
		{Name: "/python/install/bad"},
		{Name: "python/install/bin/bad", Typeflag: tar.TypeSymlink, Linkname: "../../../outside"},
		{Name: "python/install/bin/bad", Typeflag: tar.TypeSymlink, Linkname: "missing"},
		{Name: "python/install/bin/python3.14"},
	} {
		t.Run(h.Name, func(t *testing.T) {
			if _, err := trim(context.Background(), sources[0], bytes.NewReader(fixture(t, h)), io.Discard); err == nil {
				t.Fatal("异常归档未被拒绝")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := trim(ctx, sources[0], bytes.NewReader(fixture(t)), io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消未生效：%v", err)
	}
}

func TestKeepTarget(t *testing.T) {
	dir, clean, err := workpath.MkdirTemp("python-publish-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := clean(); err != nil {
			t.Error(err)
		}
	})
	for _, name := range []string{"source", "target"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := publish(int(f.Fd()), "source", "target"); err == nil {
		t.Fatal("已存在目标被覆盖")
	}
	for _, name := range []string{"source", "target"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(data) != name {
			t.Fatalf("原文件被改变：%s：%q：%v", name, data, err)
		}
	}
}
