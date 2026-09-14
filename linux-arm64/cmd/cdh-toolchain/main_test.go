// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
)

func TestCLIRejects(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLISelect(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"select"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "--name") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLIInstall(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "1h")
	spec, repo := makeCLISpec(t, 0)
	dest := filepath.Join(t.TempDir(), "tools")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := []string{"install", "--repo", repo, "--dest", dest}
	if code := runWith(context.Background(), args, &stdout, &stderr, spec); code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	want := "component=\"CLI Tool\" index=1 total=1 remaining=0 elapsed_seconds=0 state=installing"
	if !strings.Contains(stdout.String(), want) || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLIError(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "invalid")
	spec, repo := makeCLISpec(t, 0)
	dest := filepath.Join(t.TempDir(), "tools")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := []string{"install", "--repo", repo, "--dest", dest}
	if code := runWith(context.Background(), args, &stdout, &stderr, spec); code != 1 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "install failed:") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLISignal(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "1ms")
	spec, repo := makeCLISpec(t, 32<<20)
	parent := t.TempDir()
	dest := filepath.Join(parent, "tools")
	ctx, stop := signalCtx()
	defer stop()
	stdout := &notifyOut{started: make(chan struct{})}
	var stderr bytes.Buffer
	done := make(chan int, 1)
	args := []string{"install", "--repo", repo, "--dest", dest}
	go func() {
		done <- runWith(ctx, args, stdout, &stderr, spec)
	}()
	select {
	case <-stdout.started:
	case <-time.After(5 * time.Second):
		t.Fatal("未观察到安装heartbeat")
	}
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 1 {
			t.Fatalf("exit=%d stderr=%q", code, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SIGINT后安装未退出")
	}
	component := filepath.Join(dest, spec.Archives[0].Dir)
	if _, err := os.Lstat(component); !os.IsNotExist(err) {
		t.Fatalf("SIGINT后组件已发布: %v", err)
	}
	work, err := filepath.Glob(filepath.Join(parent, ".cdh-toolchain-*"))
	if err != nil || len(work) != 0 {
		t.Fatalf("SIGINT后临时目录=%v error=%v", work, err)
	}
}

type notifyOut struct {
	bytes.Buffer
	started chan struct{}
	once    sync.Once
	beats   int
}

func (output *notifyOut) Write(data []byte) (int, error) {
	count, err := output.Buffer.Write(data)
	if strings.Contains(string(data), "state=installing") {
		output.beats++
		if output.beats >= 2 {
			output.once.Do(func() { close(output.started) })
		}
	}
	return count, err
}

func makeCLISpec(t *testing.T, size int) (catalog.Spec, string) {
	t.Helper()
	version := "1.26.8"
	body := []byte("CLI Tool " + version + "\n")
	if size > len(body) {
		body = bytes.Repeat([]byte{'x'}, size)
		copy(body, []byte("CLI Tool "+version+"\n"))
	}
	var data bytes.Buffer
	pack := gzip.NewWriter(&data)
	tw := tar.NewWriter(pack)
	if err := tw.WriteHeader(&tar.Header{Name: "payload/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "payload/id.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pack.Close(); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	part := filepath.Join("payload", "cli.tar.gz.part-000")
	partPath := filepath.Join(repo, part)
	if err := os.MkdirAll(filepath.Dir(partPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partPath, data.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := catalog.Spec{Host: runtime.GOOS + "-" + runtime.GOARCH, Ready: true}
	spec.Archives = []catalog.Archive{{
		Name: "CLI Tool", Version: version, Dir: "cli-tool", Format: catalog.TarGzip,
		SHA256: cliSum(data.Bytes()), Prefix: "payload", DataOnly: true,
		Parts: []catalog.Part{{Path: filepath.ToSlash(part), Size: int64(data.Len()), SHA256: cliSum(data.Bytes())}},
		Files: []catalog.FileCheck{{Path: "id.txt", Size: int64(len(body)), SHA256: cliSum(body), Want: []string{version}}},
	}}
	return spec, repo
}

func cliSum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
