// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ulikunitz/xz"
)

type badWriter struct {
	after int
}

func (w *badWriter) Write(p []byte) (int, error) {
	if w.after > 0 {
		w.after--
		return len(p), nil
	}
	return 0, io.ErrClosedPipe
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

func TestShortLog(t *testing.T) {
	if err := writeLog(shortWriter{}, "result=%s\n", "complete"); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short output was accepted: %v", err)
	}
	_, err := parse([]string{"--help"}, shortWriter{})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short help output was accepted: %v", err)
	}
}

func TestLogStatus(t *testing.T) {
	var out, fail bytes.Buffer
	if code := finish(&out, &fail, time.Now(), nil); code != 0 || !strings.HasPrefix(out.String(), "exit=0 ") || fail.Len() != 0 {
		t.Fatalf("successful output: code=%d stdout=%q stderr=%q", code, out.String(), fail.String())
	}
	out.Reset()
	if code := finish(&badWriter{}, &fail, time.Now(), nil); code != 1 || !strings.HasPrefix(fail.String(), "exit=1 ") {
		t.Fatalf("lost final success output: code=%d stderr=%q", code, fail.String())
	}
	if code := finish(&out, &badWriter{}, time.Now(), io.ErrClosedPipe); code != 1 || out.Len() != 0 {
		t.Fatalf("lost error output: code=%d stdout=%q", code, out.String())
	}
}

func TestSpecLog(t *testing.T) {
	dir := t.TempDir()
	data := tarBytes(t, []tar.Header{{Name: "llvm/LICENSE", Typeflag: tar.TypeReg, Mode: 0o644}}, "license")
	var packed bytes.Buffer
	w, err := xz.NewWriter(&packed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "llvm.tar.xz.part-000"), packed.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := hostConfig(runtime.GOOS + "-" + runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	for _, after := range []int{0, 1} {
		_, err := makeSpec(cfg, dir, "llvm", &badWriter{after: after})
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("catalog output failure after %d writes was lost: %v", after, err)
		}
	}
}

func TestProbeLog(t *testing.T) {
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, after := range []int{0, 1} {
		p := probeCmd{ctx: t.Context(), log: &badWriter{after: after}, env: os.Environ(), work: t.TempDir()}
		_, err := p.run(program, "-test.run=^$")
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("command output failure after %d writes was lost: %v", after, err)
		}
	}
}

func TestMainLog(t *testing.T) {
	if stream := os.Getenv("CDH_LOG_MAIN"); stream != "" {
		f, err := os.Create(os.Getenv("CDH_LOG_PATH"))
		if err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
		if stream == "stdout" {
			os.Stdout = f
		} else {
			os.Stderr = f
		}
		os.Args = []string{"llvm22-check", "--help"}
		main()
		os.Exit(0)
	}
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), program, "-test.run=^TestMainLog$")
			cmd.Env = append(os.Environ(), "CDH_LOG_MAIN="+stream, "CDH_LOG_PATH="+filepath.Join(t.TempDir(), "closed"))
			out, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 {
				t.Fatalf("closed %s did not exit 1: %v\n%s", stream, err, out)
			}
		})
	}
}
