// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInputs(t *testing.T) {
	for _, host := range []string{"darwin-arm64", "linux-arm64", "linux-amd64"} {
		for _, comp := range []string{"llvm22", "ndk29", "sdkapp37", "aosp15"} {
			var out bytes.Buffer
			if err := run(context.Background(), []string{"inputs", "--host", host, "--component", comp}, &out); err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
				fields := strings.Split(line, "\t")
				if len(fields) != 3 || filepath.Base(fields[0]) != fields[0] {
					t.Fatalf("输入清单格式错误：%s", line)
				}
				hash, err := hex.DecodeString(fields[1])
				if err != nil || len(hash) != 32 {
					t.Fatalf("未登记完整摘要：%s", line)
				}
				addr, err := url.Parse(fields[2])
				if err != nil || addr.Scheme != "https" || addr.Host == "" {
					t.Fatalf("未登记HTTPS来源：%s", line)
				}
			}
		}
	}
	var out bytes.Buffer
	err := run(context.Background(), []string{"pack", "--host", "darwin-arm64", "--component", "sdkapp37",
		"--source", "missing-source", "--dest", "missing-output", "--published", "missing-published"}, &out)
	if err == nil || !strings.Contains(err.Error(), "pack不读取published") || out.Len() != 0 {
		t.Fatalf("pack接受现有裁剪包作为输入：%v %s", err, out.String())
	}
}

func TestSDKRetired(t *testing.T) {
	for _, host := range []string{"darwin-arm64", "linux-arm64", "linux-amd64"} {
		var out bytes.Buffer
		err := run(context.Background(), []string{"inputs", "--host", host, "--component", "sdk37"}, &out)
		if err == nil || out.Len() != 0 {
			t.Fatalf("重复SDK仍可生成：%s %v %s", host, err, out.String())
		}
	}
}

func TestConcurrent(t *testing.T) {
	dir := testDir(t)
	in := makeSource(t, dir)
	plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
	dest := filepath.Join(dir, "result")
	start, done := make(chan struct{}), make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			done <- generate(context.Background(), plan, dir, dest, io.Discard)
		}()
	}
	close(start)
	one, two := <-done, <-done
	if (one == nil) == (two == nil) {
		t.Fatalf("并发发布须仅成功一次：%v %v", one, two)
	}
	if err := verify(context.Background(), plan, dir, dest, "", io.Discard); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "trim-pack-") {
			t.Fatalf("并发失败残留候选：%s", entry.Name())
		}
	}
}
