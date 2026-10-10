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
	"testing"
)

type shortLog struct{}

func (shortLog) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

func TestLogShort(t *testing.T) {
	var data bytes.Buffer
	b, err := scan(bytes.NewReader(testTar(t, []tar.Header{{Name: "llvm/LICENSE", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4}})), "llvm", &data)
	if err != nil {
		t.Fatal(err)
	}
	err = writeTar(b, bytes.NewReader(data.Bytes()), io.Discard, shortLog{})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short report output was accepted: %v", err)
	}
}

func TestLogClosed(t *testing.T) {
	f, err := newFile(filepath.Join(t.TempDir(), "closed"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	prior := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = prior })
	if err := logf("result=%s\n", "complete"); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("closed log output was accepted: %v", err)
	}
}

func TestMainLog(t *testing.T) {
	if name := os.Getenv("CDH_LOG_FILE"); name != "" {
		f, err := newFile(name)
		if err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
		os.Stdout = f
		os.Args = []string{"llvm22-pack"}
		main()
		return
	}
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), program, "-test.run=^TestMainLog$")
	cmd.Env = append(os.Environ(), "CDH_LOG_FILE="+filepath.Join(t.TempDir(), "closed"))
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("closed stdout did not exit 1: %v\n%s", err, out)
	}
}
