// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
)

func TestScriptPeer(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nset -eu\nbase=$(cd \"$(dirname \"$0\")\" && pwd)\ncat \"$base/version.txt\"\nprintf '<%s>\\n' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "ndk-build"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "version.txt"), []byte("NDK peer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	probe := catalog.Probe{
		Path: "bin/ndk-build", Machine: catalog.Script,
		Args: []string{"--version", "with space", ""},
		Want: []string{"NDK peer\n<--version>\n<with space>\n<>\n"},
	}
	if err := runProbe(t.Context(), root, "NDK", probe); err != nil {
		t.Fatal(err)
	}
}

type scriptSwap struct {
	context.Context
	from string
	to   string
	err  error
	ran  bool
}

func (ctx *scriptSwap) Done() <-chan struct{} {
	if !ctx.ran {
		ctx.ran = true
		ctx.err = os.Rename(ctx.from, ctx.to)
	}
	return ctx.Context.Done()
}

func TestScriptSwap(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "tool")
	replacement := filepath.Join(dir, "replacement")
	marker := filepath.Join(dir, "executed")
	if err := os.WriteFile(program, []byte("#!/bin/sh\necho expected\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("#!/bin/sh\n: > \"$MARK\"\necho expected\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ctx := &scriptSwap{Context: context.Background(), from: replacement, to: program}
	probe := catalog.Probe{
		Path: "tool", Machine: catalog.Script,
		Want: []string{"expected"}, Env: []string{"MARK=" + marker},
	}
	if err := runProbe(ctx, root, "script", probe); err == nil {
		t.Fatal("replaced script was accepted")
	}
	if !ctx.ran || ctx.err != nil {
		t.Fatalf("script replacement: ran=%v, error=%v", ctx.ran, ctx.err)
	}
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		t.Fatalf("replacement script ran: %v", err)
	}
}
