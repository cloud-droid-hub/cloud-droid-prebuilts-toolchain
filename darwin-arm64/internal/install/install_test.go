// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/ulikunitz/xz"
	"golang.org/x/sys/unix"
)

func TestInstall(t *testing.T) {
	spec, repo := makeSample(t)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Join(dest, "go126"), 0o755)
	assertMode(t, filepath.Join(dest, "go126", "bin", "go"), 0o755)
	assertMode(t, filepath.Join(dest, "go126", "README.md"), 0o644)
	got, err := os.Readlink(filepath.Join(dest, "go126", "go-link"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "bin/go" {
		t.Fatalf("link=%q", got)
	}
}

func TestOneDelete(t *testing.T) {
	spec, repo := makeMany(t, 3)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dest, spec.Archives[1].Dir)); err != nil {
		t.Fatal(err)
	}
	dropParts(t, repo, spec.Archives[0], spec.Archives[2])
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatalf("补齐单个组件: %v", err)
	}
	if err := Check(context.Background(), dest, spec); err != nil {
		t.Fatal(err)
	}
}

func TestManyDelete(t *testing.T) {
	spec, repo := makeMany(t, 4)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{1, 3} {
		if err := os.RemoveAll(filepath.Join(dest, spec.Archives[index].Dir)); err != nil {
			t.Fatal(err)
		}
	}
	dropParts(t, repo, spec.Archives[0], spec.Archives[2])
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatalf("补齐多个组件: %v", err)
	}
	if err := Check(context.Background(), dest, spec); err != nil {
		t.Fatal(err)
	}
}

func TestInterrupt(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "1h")
	spec, repo := makeMany(t, 3)
	dest := filepath.Join(t.TempDir(), "tools")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	match := fmt.Sprintf("component=%q index=2 total=3 remaining=1 elapsed_seconds=0 state=installing", spec.Archives[1].Name)
	output := &cancelOut{cancel: cancel, match: match}
	err := RunTo(ctx, repo, dest, spec, output)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v want context canceled", err)
	}
	first := filepath.Join(dest, spec.Archives[0].Dir, "id.txt")
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("已完成组件未保留: %v", err)
	}
	dropParts(t, repo, spec.Archives[0])
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatalf("中断后继续: %v", err)
	}
	if err := Check(context.Background(), dest, spec); err != nil {
		t.Fatal(err)
	}
}

func TestCancelEdge(t *testing.T) {
	spec, repo := makeMany(t, 1)
	dest := filepath.Join(t.TempDir(), "tools")
	ctx := &stopCtx{Context: context.Background(), limit: 4, done: make(chan struct{})}
	err := Run(ctx, repo, dest, spec)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v want context canceled", err)
	}
	component := filepath.Join(dest, spec.Archives[0].Dir)
	if _, err := os.Lstat(component); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("取消后的当前组件已发布: %v", err)
	}
}

func TestRootSwap(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "1h")
	spec, repo := makeMany(t, 1)
	parent := t.TempDir()
	dest := filepath.Join(parent, "tools")
	moved := filepath.Join(parent, "tools-moved")
	outside := t.TempDir()
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	output := &swapOut{dest: dest, moved: moved, outside: outside}
	if err := RunTo(context.Background(), repo, dest, spec, output); err == nil {
		t.Fatal("目标根被替换后安装仍成功")
	}
	if output.err != nil {
		t.Fatalf("替换测试目标: %v", output.err)
	}
	outsideComp := filepath.Join(outside, spec.Archives[0].Dir)
	if _, err := os.Lstat(outsideComp); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("组件写入目标根之外: %v", err)
	}
}

func TestSourceSwap(t *testing.T) {
	spec, repo := makeMany(t, 1)
	parent := t.TempDir()
	dest := filepath.Join(parent, "tools")
	ctx := &sourceSwap{
		Context: context.Background(), parent: dest, dir: spec.Archives[0].Dir,
		body: []byte("Tool 1 1.26.1\n"),
	}
	if err := Run(ctx, repo, dest, spec); err == nil {
		t.Fatal("暂存源替换后安装仍成功")
	}
	if ctx.err != nil || !ctx.swapped {
		t.Fatalf("替换暂存源: swapped=%t error=%v", ctx.swapped, ctx.err)
	}
	marker := filepath.Join(dest, spec.Archives[0].Dir, "attacker.txt")
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("未经校验的暂存目录已发布: %v", err)
	}
}

func TestProbeSwap(t *testing.T) {
	spec, repo := makeSample(t)
	parent := t.TempDir()
	dest := filepath.Join(parent, "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	probeDir := filepath.Join(outside, spec.Archives[0].Dir, "bin")
	if err := os.MkdirAll(probeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(outside, "executed")
	script := "#!/bin/sh\ntouch \"$MARKER\"\necho go1.26.8\n"
	if err := os.WriteFile(filepath.Join(probeDir, "go"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	spec.Archives[0].Probes[0].Env = []string{"MARKER=" + marker}
	ctx := &probeSwap{
		Context: context.Background(), dest: dest,
		moved: filepath.Join(parent, "tools-moved"), outside: outside,
	}
	if err := Run(ctx, repo, dest, spec); err == nil {
		t.Fatal("目标根替换后校验仍成功")
	}
	if ctx.err != nil || !ctx.swapped {
		t.Fatalf("替换校验目标: swapped=%t error=%v", ctx.swapped, ctx.err)
	}
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("执行了替换目标中的探针: %v", err)
	}
}

func TestUnsafeParent(t *testing.T) {
	spec, repo := makeMany(t, 1)
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(parent, "tools")
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("可由其它用户写入的父目录被接受")
	}
	if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("拒绝不安全父目录前已创建目标: %v", err)
	}
}

func TestGroupParent(t *testing.T) {
	spec, repo := makeMany(t, 1)
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o775); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(parent, "tools")
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("普通组可写父目录被接受")
	}
	if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("拒绝组可写父目录前已创建目标: %v", err)
	}
}

func TestParentLink(t *testing.T) {
	spec, repo := makeMany(t, 1)
	base := t.TempDir()
	real := t.TempDir()
	parent := filepath.Join(real, "existing")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(base, "linked")); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(base, "linked", "existing", "tools")
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("含中间符号链接的目标路径被接受")
	}
	if _, err := os.Lstat(filepath.Join(parent, "tools")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("拒绝链接路径前已创建目标: %v", err)
	}
}

func TestMountFix(t *testing.T) {
	uid := uint32(os.Geteuid())
	mode := os.FileMode(0o775) | os.ModeSetgid
	got, changed := mountFix(mode, uid, 2, 1, true)
	want := mode | os.ModeSticky
	if !changed || got != want {
		t.Fatalf("mountFix(create=true)=(%v,%t) want (%v,true)", got, changed, want)
	}
	got, changed = mountFix(mode, uid, 2, 1, false)
	if changed || got != mode {
		t.Fatalf("mountFix(create=false)=(%v,%t) want (%v,false)", got, changed, mode)
	}
}

func TestOwnedMount(t *testing.T) {
	spec, repo := makeMany(t, 1)
	base := t.TempDir()
	name := base
	found := false
	mount := ""
	var original os.FileMode
	for {
		info, err := os.Lstat(name)
		if err != nil {
			t.Fatal(err)
		}
		parent := filepath.Dir(name)
		if parent == name {
			break
		}
		parentInfo, err := os.Lstat(parent)
		if err != nil {
			t.Fatal(err)
		}
		stat, statOK := info.Sys().(*syscall.Stat_t)
		parentStat, parentOK := parentInfo.Sys().(*syscall.Stat_t)
		mode := info.Mode()
		if statOK && parentOK && stat.Uid == uint32(os.Geteuid()) &&
			mode.Perm()&0o020 != 0 && mode.Perm()&0o002 == 0 && stat.Dev != parentStat.Dev {
			found = true
			mount = name
			original = mode
			break
		}
		name = parent
	}
	if !found {
		t.Skip("测试路径没有当前用户拥有的组可写挂载根")
	}
	t.Cleanup(func() {
		if err := os.Chmod(mount, original); err != nil {
			t.Errorf("恢复挂载根模式: %v", err)
		}
	})
	baseMode := original.Perm() | original&(os.ModeSetuid|os.ModeSetgid)
	if err := os.Chmod(mount, baseMode); err != nil {
		t.Fatalf("设置挂载根测试模式: %v", err)
	}
	before, err := os.Lstat(mount)
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode()&os.ModeSticky != 0 {
		t.Fatalf("挂载根测试模式仍有sticky位: mode=%v", before.Mode())
	}
	dest := filepath.Join(base, "tools")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Check(context.Background(), dest, spec); err == nil {
		t.Fatal("Check接受了缺少sticky位的组可写挂载根")
	}
	afterCheck, err := os.Lstat(mount)
	if err != nil {
		t.Fatal(err)
	}
	if afterCheck.Mode() != before.Mode() {
		t.Fatalf("Check修改了挂载根模式: got=%v want=%v", afterCheck.Mode(), before.Mode())
	}
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatalf("当前用户拥有的组可写挂载根被拒绝: %v", err)
	}
	info, err := os.Lstat(mount)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSticky == 0 {
		t.Fatalf("安装后挂载根未设置sticky位: mode=%v", info.Mode())
	}
	if got, want := info.Mode()&^os.ModeSticky, before.Mode(); got != want {
		t.Fatalf("安装修改了sticky之外的挂载根模式: got=%v want=%v", got, want)
	}
}

func TestParentMode(t *testing.T) {
	uid := uint32(os.Geteuid())
	tests := []struct {
		name      string
		mode      os.FileMode
		uid       uint32
		dev       uint64
		parentDev uint64
		want      bool
	}{
		{name: "secure", mode: 0o755, uid: uid + 1, dev: 1, parentDev: 1, want: true},
		{name: "owned mount", mode: 0o775, uid: uid, dev: 2, parentDev: 1},
		{name: "sticky mount", mode: 0o775 | os.ModeSticky, uid: uid, dev: 2, parentDev: 1, want: true},
		{name: "same device", mode: 0o775, uid: uid, dev: 1, parentDev: 1},
		{name: "other owner", mode: 0o775, uid: uid + 1, dev: 2, parentDev: 1},
		{name: "world writable", mode: 0o777, uid: uid, dev: 2, parentDev: 1},
	}
	for _, test := range tests {
		if got := parentMode(test.mode, test.uid, test.dev, test.parentDev); got != test.want {
			t.Errorf("%s: parentMode=%t want %t", test.name, got, test.want)
		}
	}
}

func TestSafeCleanup(t *testing.T) {
	parent := t.TempDir()
	work := filepath.Join(parent, ".cdh-toolchain-work")
	original := filepath.Join(work, "component", "payload")
	if err := os.MkdirAll(original, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(original, "data"), []byte("temporary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parentRoot, err := openRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer parentRoot.Close()
	root, err := openRoot(work)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	moved := work + ".moved"
	if err := os.Rename(work, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(work, "keep.txt")
	if err := os.WriteFile(keep, []byte("user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clearTemp(parentRoot, root, filepath.Base(work))
	if got, err := os.ReadFile(keep); err != nil || string(got) != "user\n" {
		t.Fatalf("替换路径内容=%q error=%v", got, err)
	}
	if entries, err := os.ReadDir(moved); err != nil || len(entries) != 0 {
		t.Fatalf("原始暂存目录未清空: entries=%v error=%v", entries, err)
	}
}

func TestKeepInvalid(t *testing.T) {
	spec, repo := makeMany(t, 2)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	identity := filepath.Join(dest, spec.Archives[0].Dir, "id.txt")
	changed := []byte("user-edited\n")
	if err := os.WriteFile(identity, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(dest, "user-note.txt")
	if err := os.WriteFile(note, []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("校验失败的已有组件被覆盖")
	}
	got, err := os.ReadFile(identity)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, changed) {
		t.Fatalf("已有组件内容=%q", got)
	}
	if got, err := os.ReadFile(note); err != nil || string(got) != "keep\n" {
		t.Fatalf("用户文件=%q error=%v", got, err)
	}
}

func TestProgress(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "1h")
	spec, repo := makeMany(t, 2)
	dest := filepath.Join(t.TempDir(), "tools")
	var output bytes.Buffer
	if err := RunTo(context.Background(), repo, dest, spec, &output); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"checking", "installing", "installed"} {
		want := fmt.Sprintf("component=%q index=1 total=2 remaining=1 elapsed_seconds=0 state=%s", spec.Archives[0].Name, state)
		if !strings.Contains(output.String(), want) {
			t.Fatalf("progress misses %q:\n%s", want, output.String())
		}
	}
	output.Reset()
	if err := RunTo(context.Background(), repo, dest, spec, &output); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("component=%q index=2 total=2 remaining=0 elapsed_seconds=0 state=skipped", spec.Archives[1].Name)
	if !strings.Contains(output.String(), want) {
		t.Fatalf("resume progress misses %q:\n%s", want, output.String())
	}
	if strings.Contains(output.String(), "state=installing") {
		t.Fatalf("verified components reinstalled:\n%s", output.String())
	}
}

func TestHeartbeat(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "1ms")
	spec, repo := makeSized(t, []int{16 << 20})
	var output bytes.Buffer
	if err := RunTo(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec, &output); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(output.String(), "state=installing"); count < 2 {
		t.Fatalf("install heartbeats=%d want at least 2:\n%s", count, output.String())
	}
	want := "index=1 total=1 remaining=0 elapsed_seconds="
	if !strings.Contains(output.String(), want) {
		t.Fatalf("heartbeat misses %q:\n%s", want, output.String())
	}
}

func TestBadInterval(t *testing.T) {
	t.Setenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL", "invalid")
	spec, repo := makeMany(t, 1)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := RunTo(context.Background(), repo, dest, spec, io.Discard); err == nil {
		t.Fatal("非法进度间隔未被拒绝")
	}
	if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("失败前创建了目标: %v", err)
	}
}

func TestFileCheck(t *testing.T) {
	spec, repo := makeSample(t)
	spec.Archives[0].Files = []catalog.FileCheck{{
		Path: "README.md", Size: 7, SHA256: sumHex([]byte("sample\n")), Want: []string{"sample"},
	}}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(dest, "go126", "README.md")
	if err := os.WriteFile(readme, []byte("tamper\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Check(context.Background(), dest, spec); err == nil {
		t.Fatal("changed identity file accepted")
	}
}

func TestFileLink(t *testing.T) {
	spec, repo := makeSample(t)
	script := []byte("#!/bin/sh\necho go1.26.8\n")
	spec.Archives[0].Files = []catalog.FileCheck{{
		Path: "go-link", Size: int64(len("bin/go")), SHA256: sumHex(script), Want: []string{"go1.26.8"},
	}}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("linked identity file accepted")
	}
}

func TestFileOnly(t *testing.T) {
	spec, repo := makeSample(t)
	script := []byte("#!/bin/sh\necho go1.26.8\n")
	spec.Archives[0].Probes = nil
	spec.Archives[0].Files = []catalog.FileCheck{{
		Path: "bin/go", Size: int64(len(script)), SHA256: sumHex(script), Want: []string{"go1.26.8"},
	}}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
}

func TestFileParent(t *testing.T) {
	data := "identity\n"
	entries := probeItems([]tarItem{
		{header: tar.Header{Name: "payload/meta/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/meta/id", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(data))}, data: data},
	})
	spec, repo := makeSpec(t, entries)
	spec.Archives[0].Files = []catalog.FileCheck{{
		Path: "meta/id", Size: int64(len(data)), SHA256: sumHex([]byte(data)), Want: []string{"identity"},
	}}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(dest, "go126", "meta")
	if err := os.RemoveAll(meta); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "id"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, meta); err != nil {
		t.Fatal(err)
	}
	if err := Check(context.Background(), dest, spec); err == nil {
		t.Fatal("external identity file accepted")
	}
}

func TestModuleFence(t *testing.T) {
	spec, repo := makeSample(t)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("go.mod mode=%#o", info.Mode().Perm())
	}
}

func TestInstallXZ(t *testing.T) {
	script := "#!/bin/sh\necho go1.26.8\n"
	spec, repo := makePack(t, []tarItem{{header: tar.Header{
		Name: "payload/bin/go", Typeflag: tar.TypeReg, Mode: 0o700, Size: int64(len(script)),
	}, data: script}}, catalog.TarXZ)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Join(dest, "go126", "bin", "go"), 0o755)
}

func TestBadPart(t *testing.T) {
	spec, repo := makeSample(t)
	part := filepath.Join(repo, filepath.FromSlash(spec.Archives[0].Parts[0].Path))
	content, err := os.ReadFile(part)
	if err != nil {
		t.Fatal(err)
	}
	content[0] ^= 0xff
	if err := os.WriteFile(part, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("摘要错误的分片未被拒绝")
	}
}

func TestPartParent(t *testing.T) {
	spec, repo := makeSample(t)
	outside := t.TempDir()
	moved := filepath.Join(outside, "payload")
	if err := os.Rename(filepath.Join(repo, "payload"), moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, filepath.Join(repo, "payload")); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("分片父目录链接未被拒绝")
	}
}

func TestPartSwap(t *testing.T) {
	spec, repo := makeSample(t)
	reader, err := openParts(repo, spec.Archives[0].Parts)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	part := filepath.Join(repo, filepath.FromSlash(spec.Archives[0].Parts[0].Path))
	if err := os.Rename(part, part+".old"); err != nil {
		t.Fatal(err)
	}
	bad := bytes.Repeat([]byte{0xff}, int(spec.Archives[0].Parts[0].Size))
	if err := os.WriteFile(part, bad, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := reader.finish(context.Background(), spec.Archives[0].SHA256); err != nil {
		t.Fatalf("已打开分片受路径替换影响: %v", err)
	}
}

func TestArchiveSum(t *testing.T) {
	spec, repo := makeSample(t)
	spec.Archives[0].SHA256 = strings.Repeat("0", 64)
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("完整归档摘要错误未被拒绝")
	}
}

func TestEscape(t *testing.T) {
	spec, repo := makeSpec(t, probeItems([]tarItem{{header: tar.Header{
		Name: "payload/../../escape", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1,
	}, data: "x"}}))
	wantReject(t, spec, repo)
}

func TestBadLink(t *testing.T) {
	spec, repo := makeSpec(t, probeItems([]tarItem{{header: tar.Header{
		Name: "payload/link", Typeflag: tar.TypeSymlink, Linkname: "../../outside", Mode: 0o777,
	}}}))
	wantReject(t, spec, repo)
}

func TestLinkChain(t *testing.T) {
	entries := []tarItem{
		{header: tar.Header{Name: "payload/b/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/a", Typeflag: tar.TypeSymlink, Linkname: ".", Mode: 0o777}},
		{header: tar.Header{Name: "payload/a/b/x", Typeflag: tar.TypeSymlink, Linkname: "../../outside", Mode: 0o777}},
	}
	spec, repo := makeSpec(t, probeItems(entries))
	wantReject(t, spec, repo)
}

func TestCaseAlias(t *testing.T) {
	caseRoot := t.TempDir()
	lower := filepath.Join(caseRoot, "lowercase")
	if err := os.Mkdir(lower, 0o755); err != nil {
		t.Fatal(err)
	}
	upperInfo, err := os.Stat(filepath.Join(caseRoot, "LOWERCASE"))
	lowerInfo, lowerErr := os.Stat(lower)
	if err != nil || lowerErr != nil || !os.SameFile(lowerInfo, upperInfo) {
		t.Skip("文件系统区分路径大小写")
	}
	script := "#!/bin/sh\necho go1.26.8\n"
	entries := []tarItem{
		{header: tar.Header{Name: "payload/a", Typeflag: tar.TypeSymlink, Linkname: ".", Mode: 0o777}},
		{header: tar.Header{Name: "payload/A/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/bin/go", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(script))}, data: script},
	}
	spec, repo := makeSpec(t, entries)
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("归档物理路径别名未被拒绝")
	}
}

func TestBadType(t *testing.T) {
	spec, repo := makeSpec(t, probeItems([]tarItem{{header: tar.Header{
		Name: "payload/fifo", Typeflag: tar.TypeFifo, Mode: 0o600,
	}}}))
	wantReject(t, spec, repo)
}

func TestDuplicate(t *testing.T) {
	items := []tarItem{
		{header: tar.Header{Name: "payload/file", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}, data: "a"},
		{header: tar.Header{Name: "payload/file", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}, data: "b"},
	}
	spec, repo := makeSpec(t, probeItems(items))
	wantReject(t, spec, repo)
}

func TestBadHost(t *testing.T) {
	spec, repo := makeSample(t)
	if spec.Host == "linux-amd64" {
		spec.Host = "linux-arm64"
	} else {
		spec.Host = "linux-amd64"
	}
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("错误宿主分支未被拒绝")
	}
}

func TestBadMachine(t *testing.T) {
	spec, repo := makeSample(t)
	spec.Archives[0].DataOnly = false
	spec.Archives[0].Probes[0].Machine = testMachine()
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("脚本被接受为宿主原生程序")
	}
}

func TestHostMachine(t *testing.T) {
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := checkMachine(program, testMachine()); err != nil {
		t.Fatal(err)
	}
}

func TestFatMach(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("仅macOS使用Mach-O")
	}
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	thin, err := os.ReadFile(program)
	if err != nil {
		t.Fatal(err)
	}
	const offset = 4096
	fat := make([]byte, offset+len(thin))
	binary.BigEndian.PutUint32(fat[0:4], 0xcafebabe)
	binary.BigEndian.PutUint32(fat[4:8], 1)
	binary.BigEndian.PutUint32(fat[8:12], binary.LittleEndian.Uint32(thin[4:8]))
	binary.BigEndian.PutUint32(fat[12:16], binary.LittleEndian.Uint32(thin[8:12]))
	binary.BigEndian.PutUint32(fat[16:20], offset)
	binary.BigEndian.PutUint32(fat[20:24], uint32(len(thin)))
	binary.BigEndian.PutUint32(fat[24:28], 12)
	copy(fat[offset:], thin)
	path := filepath.Join(t.TempDir(), "fat-mach")
	if err := os.WriteFile(path, fat, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := checkMachine(path, catalog.MachARM64); err == nil {
		t.Fatal("多架构Mach-O未被拒绝")
	}
}

func TestModeMask(t *testing.T) {
	spec, repo := makeSample(t)
	root := t.TempDir()
	oldMask := syscall.Umask(0o111)
	defer syscall.Umask(oldMask)
	dest := filepath.Join(root, "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Join(dest, "go126", "bin"), 0o755)
	assertMode(t, filepath.Join(dest, "go126", "bin", "go"), 0o755)
	assertMode(t, filepath.Join(dest, "go126", "README.md"), 0o644)
}

func TestOldKeep(t *testing.T) {
	spec, repo := makeSample(t)
	spec.Archives[0].Probes[0].Want = []string{"go1.26.8", "never-present"}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dest, "keep.txt")
	if err := os.WriteFile(marker, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err == nil {
		t.Fatal("失败探针未阻断发布")
	}
	content, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old\n" {
		t.Fatalf("旧目录内容=%q", content)
	}
}

func TestOldKeepExtra(t *testing.T) {
	spec, repo := makeSample(t)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dest, "old.txt")
	if err := os.WriteFile(marker, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != "old\n" {
		t.Fatalf("用户文件=%q error=%v", content, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "go126", "bin", "go")); err != nil {
		t.Fatal(err)
	}
}

func TestRepoDest(t *testing.T) {
	spec, repo := makeSample(t)
	part := filepath.Join(repo, filepath.FromSlash(spec.Archives[0].Parts[0].Path))
	if err := Run(context.Background(), repo, repo, spec); err == nil {
		t.Fatal("源仓被接受为安装目标")
	}
	if _, err := os.Stat(part); err != nil {
		t.Fatalf("源仓内容被修改: %v", err)
	}
}

func TestRepoParent(t *testing.T) {
	spec, repo := makeSample(t)
	part := filepath.Join(repo, filepath.FromSlash(spec.Archives[0].Parts[0].Path))
	if err := Run(context.Background(), repo, filepath.Dir(repo), spec); err == nil {
		t.Fatal("源仓父目录被接受为安装目标")
	}
	if _, err := os.Stat(part); err != nil {
		t.Fatalf("源仓内容被修改: %v", err)
	}
}

func TestRepoCase(t *testing.T) {
	spec, repo := makeSample(t)
	parent := t.TempDir()
	lower := filepath.Join(parent, "lowercase")
	if err := os.Rename(repo, lower); err != nil {
		t.Fatal(err)
	}
	repo = lower
	alias := filepath.Join(parent, "LOWERCASE")
	repoInfo, err := os.Stat(repo)
	if err != nil {
		t.Fatal(err)
	}
	aliasInfo, err := os.Stat(alias)
	if err != nil || !os.SameFile(repoInfo, aliasInfo) {
		t.Skip("文件系统区分路径大小写")
	}
	part := filepath.Join(repo, filepath.FromSlash(spec.Archives[0].Parts[0].Path))
	if err := Run(context.Background(), repo, alias, spec); err == nil {
		t.Fatal("源仓大小写别名被接受为安装目标")
	}
	if _, err := os.Stat(part); err != nil {
		t.Fatalf("源仓内容被修改: %v", err)
	}
}

func TestProbeLink(t *testing.T) {
	spec, repo := makeSample(t)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(dest, "go126", "bin", "go")
	if err := os.Remove(program); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside-go")
	if err := os.WriteFile(outside, []byte("#!/bin/sh\necho go1.26.8\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, program); err != nil {
		t.Fatal(err)
	}
	if err := Check(context.Background(), dest, spec); err == nil {
		t.Fatal("工具目录之外的探针程序未被拒绝")
	}
}

func TestProbeArg0(t *testing.T) {
	program := buildArg0(t)
	items := []tarItem{
		{header: tar.Header{Name: "payload/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/bin/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(program))}, data: string(program)},
		{header: tar.Header{Name: "payload/bin/mode-link", Typeflag: tar.TypeSymlink, Linkname: "tool", Mode: 0o777}},
	}
	spec, repo := makeSpec(t, items)
	spec.Archives[0].DataOnly = false
	spec.Archives[0].Probes[0] = catalog.Probe{
		Path: "bin/mode-link", Want: []string{"go1.26.8", "mode-link"}, Machine: testMachine(),
	}
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err != nil {
		t.Fatal(err)
	}
}

func TestProbeRelLink(t *testing.T) {
	useLocalTmp(t)
	program := buildArg0(t)
	items := []tarItem{
		{header: tar.Header{Name: "payload/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/lib/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/lib/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(program))}, data: string(program)},
		{header: tar.Header{Name: "payload/bin/link", Typeflag: tar.TypeSymlink, Linkname: "../lib/tool", Mode: 0o777}},
	}
	spec, repo := makeSpec(t, items)
	spec.Archives[0].DataOnly = false
	spec.Archives[0].Probes = []catalog.Probe{{
		Path: "bin/link", Want: []string{"go1.26.8 link"}, Machine: testMachine(),
	}}
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err != nil {
		t.Fatal(err)
	}
}

func TestProbeCancel(t *testing.T) {
	useLocalTmp(t)
	script := "#!/bin/sh\n: > \"$READY\"\necho go1.26.8 ready\nwhile :; do :; done\n"
	items := []tarItem{
		{header: tar.Header{Name: "payload/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "payload/bin/wait", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(script))}, data: script},
	}
	spec, repo := makeSpec(t, items)
	marker := filepath.Join(t.TempDir(), "ready")
	spec.Archives[0].Probes = []catalog.Probe{{
		Path: "bin/wait", Want: []string{"go1.26.8", "ready"}, Machine: catalog.Script, Env: []string{"READY=" + marker},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dest := filepath.Join(t.TempDir(), "tools")
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, repo, dest, spec)
	}()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		if _, err := os.Stat(marker); err == nil {
			cancel()
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			cancel()
			t.Fatalf("inspect probe marker: %v", err)
		}
		select {
		case runErr := <-done:
			t.Fatalf("probe ended before ready: %v", runErr)
		case <-deadline.C:
			cancel()
			t.Fatal("probe did not become ready")
		case <-ticker.C:
		}
	}
	select {
	case runErr := <-done:
		if !errors.Is(runErr, context.Canceled) {
			t.Fatalf("error=%v want context canceled", runErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("probe did not stop after cancellation")
	}
}

func buildArg0(t *testing.T) []byte {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	code := "package main\nimport (\"fmt\"; \"os\"; \"path/filepath\")\nfunc main(){fmt.Println(\"go1.26.8\", filepath.Base(os.Args[0]))}\n"
	if err := os.WriteFile(source, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(dir, "arg0-bin")
	output, err := exec.Command("go", "build", "-o", program, source).CombinedOutput()
	if err != nil {
		t.Fatalf("build arg0 probe: %v: %s", err, output)
	}
	data, err := os.ReadFile(program)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSwapDirs(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	if err := os.Mkdir(left, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(right, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(left, "left"), []byte("left"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(right, "right"), []byte("right"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := swapDirs(left, right); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(left, "right")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(right, "left")); err != nil {
		t.Fatal(err)
	}
}

type tarItem struct {
	header tar.Header
	data   string
}

func makeSample(t *testing.T) (catalog.Spec, string) {
	t.Helper()
	script := "#!/bin/sh\necho go1.26.8\n"
	return makeSpec(t, []tarItem{
		{header: tar.Header{Name: "payload/", Typeflag: tar.TypeDir, Mode: 0o700}},
		{header: tar.Header{Name: "payload/bin/", Typeflag: tar.TypeDir, Mode: 0o700}},
		{header: tar.Header{Name: "payload/bin/go", Typeflag: tar.TypeReg, Mode: 0o700, Size: int64(len(script))}, data: script},
		{header: tar.Header{Name: "payload/README.md", Typeflag: tar.TypeReg, Mode: 0o600, Size: 7}, data: "sample\n"},
		{header: tar.Header{Name: "payload/go-link", Typeflag: tar.TypeSymlink, Linkname: "bin/go", Mode: 0o777}},
	})
}

func makeSpec(t *testing.T, entries []tarItem) (catalog.Spec, string) {
	t.Helper()
	return makePack(t, entries, catalog.TarGzip)
}

func makeMany(t *testing.T, count int) (catalog.Spec, string) {
	t.Helper()
	return makeSized(t, make([]int, count))
}

func makeSized(t *testing.T, sizes []int) (catalog.Spec, string) {
	t.Helper()
	repo := t.TempDir()
	spec := catalog.Spec{Host: runtime.GOOS + "-" + runtime.GOARCH, Ready: true}
	for index, size := range sizes {
		name := fmt.Sprintf("Tool %d", index+1)
		version := fmt.Sprintf("1.26.%d", index+1)
		dir := fmt.Sprintf("tool%d", index+1)
		prefix := fmt.Sprintf("payload%d", index+1)
		body := []byte(name + " " + version + "\n")
		if size > len(body) {
			body = bytes.Repeat([]byte{'x'}, size)
			copy(body, []byte(name+" "+version+"\n"))
		}
		var data bytes.Buffer
		pack := gzip.NewWriter(&data)
		tw := tar.NewWriter(pack)
		if err := tw.WriteHeader(&tar.Header{Name: prefix + "/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
			t.Fatal(err)
		}
		if err := tw.WriteHeader(&tar.Header{Name: prefix + "/id.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
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
		part := filepath.ToSlash(filepath.Join("payload", dir, "archive.tar.gz.part-000"))
		partPath := filepath.Join(repo, filepath.FromSlash(part))
		if err := os.MkdirAll(filepath.Dir(partPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(partPath, data.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		spec.Archives = append(spec.Archives, catalog.Archive{
			Name: name, Version: version, Dir: dir, Format: catalog.TarGzip,
			SHA256: sumHex(data.Bytes()), Prefix: prefix, DataOnly: true,
			Parts: []catalog.Part{{Path: part, Size: int64(data.Len()), SHA256: sumHex(data.Bytes())}},
			Files: []catalog.FileCheck{{Path: "id.txt", Size: int64(len(body)), SHA256: sumHex(body), Want: []string{version}}},
		})
	}
	return spec, repo
}

func dropParts(t *testing.T, repo string, archives ...catalog.Archive) {
	t.Helper()
	for _, archive := range archives {
		for _, part := range archive.Parts {
			if err := os.Remove(filepath.Join(repo, filepath.FromSlash(part.Path))); err != nil {
				t.Fatal(err)
			}
		}
	}
}

type stopCtx struct {
	context.Context
	calls   int
	limit   int
	stopped bool
	done    chan struct{}
}

func (ctx *stopCtx) Done() <-chan struct{} {
	return ctx.done
}

type swapOut struct {
	bytes.Buffer
	dest    string
	moved   string
	outside string
	swapped bool
	err     error
}

type sourceSwap struct {
	context.Context
	parent  string
	dir     string
	body    []byte
	swapped bool
	err     error
}

func (ctx *sourceSwap) Err() error {
	if ctx.swapped || ctx.err != nil {
		return nil
	}
	workDirs, err := filepath.Glob(filepath.Join(ctx.parent, ".cdh-toolchain-*"))
	if err != nil {
		ctx.err = err
		return nil
	}
	for _, work := range workDirs {
		got, err := os.ReadFile(filepath.Join(work, ctx.dir, "id.txt"))
		if err != nil || !bytes.Equal(got, ctx.body) {
			continue
		}
		moved := work + ".checked"
		if err := os.Rename(work, moved); err != nil {
			ctx.err = err
			return nil
		}
		component := filepath.Join(work, ctx.dir)
		if err := os.MkdirAll(component, 0o755); err != nil {
			ctx.err = err
			return nil
		}
		if err := os.WriteFile(filepath.Join(component, "id.txt"), ctx.body, 0o644); err != nil {
			ctx.err = err
			return nil
		}
		ctx.err = os.WriteFile(filepath.Join(component, "attacker.txt"), []byte("unverified\n"), 0o644)
		ctx.swapped = ctx.err == nil
		return nil
	}
	return nil
}

type probeSwap struct {
	context.Context
	dest    string
	moved   string
	outside string
	swapped bool
	err     error
}

func (ctx *probeSwap) Err() error {
	if ctx.swapped || ctx.err != nil {
		return nil
	}
	if err := os.Rename(ctx.dest, ctx.moved); err != nil {
		ctx.err = err
		return nil
	}
	ctx.err = os.Symlink(ctx.outside, ctx.dest)
	ctx.swapped = ctx.err == nil
	return nil
}

func (output *swapOut) Write(data []byte) (int, error) {
	count, err := output.Buffer.Write(data)
	if output.swapped || !strings.Contains(string(data), "state=installing") {
		return count, err
	}
	output.swapped = true
	if renameErr := os.Rename(output.dest, output.moved); renameErr != nil {
		output.err = renameErr
		return count, err
	}
	output.err = os.Symlink(output.outside, output.dest)
	return count, err
}

func (ctx *stopCtx) Err() error {
	if ctx.stopped {
		return context.Canceled
	}
	ctx.calls++
	if ctx.calls >= ctx.limit {
		ctx.stopped = true
		close(ctx.done)
		return context.Canceled
	}
	return nil
}

type cancelOut struct {
	bytes.Buffer
	cancel context.CancelFunc
	match  string
}

func (output *cancelOut) Write(data []byte) (int, error) {
	count, err := output.Buffer.Write(data)
	if strings.Contains(string(data), output.match) {
		output.cancel()
	}
	return count, err
}

func probeItems(entries []tarItem) []tarItem {
	script := "#!/bin/sh\necho go1.26.8\n"
	return append(entries,
		tarItem{header: tar.Header{Name: "payload/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		tarItem{header: tar.Header{Name: "payload/bin/go", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(script))}, data: script},
	)
}

func makePack(t *testing.T, entries []tarItem, format catalog.Format) (catalog.Spec, string) {
	t.Helper()
	var data bytes.Buffer
	var pack io.WriteCloser
	switch format {
	case catalog.TarGzip:
		pack = gzip.NewWriter(&data)
	case catalog.TarXZ:
		var err error
		pack, err = xz.NewWriter(&data)
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("不支持的测试归档格式: %d", format)
	}
	tw := tar.NewWriter(pack)
	for _, entry := range entries {
		if err := tw.WriteHeader(&entry.header); err != nil {
			t.Fatal(err)
		}
		if entry.data != "" {
			if _, err := tw.Write([]byte(entry.data)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pack.Close(); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	cut := data.Len() / 2
	partData := [][]byte{data.Bytes()[:cut], data.Bytes()[cut:]}
	parts := make([]catalog.Part, 0, len(partData))
	for i, content := range partData {
		name := filepath.ToSlash(filepath.Join("payload", "go126", "go.tar.gz.part-00"+string(rune('0'+i))))
		path := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
		parts = append(parts, catalog.Part{Path: name, Size: int64(len(content)), SHA256: sumHex(content)})
	}
	host := runtime.GOOS + "-" + runtime.GOARCH
	return catalog.Spec{
		Host:  host,
		Ready: true,
		Archives: []catalog.Archive{{
			Name:     "Go",
			Version:  "1.26.8",
			Dir:      "go126",
			Format:   format,
			SHA256:   sumHex(data.Bytes()),
			Prefix:   "payload",
			Parts:    parts,
			Probes:   []catalog.Probe{{Path: "bin/go", Args: []string{"version"}, Want: []string{"go1.26.8"}, Machine: catalog.Script}},
			DataOnly: true,
		}},
	}, repo
}

func testMachine() catalog.Machine {
	if runtime.GOOS == "darwin" {
		return catalog.MachARM64
	}
	if runtime.GOARCH == "arm64" {
		return catalog.ELFARM64
	}
	return catalog.ELFAMD64
}

func wantReject(t *testing.T, spec catalog.Spec, repo string) {
	t.Helper()
	if err := catalog.Valid(spec); err != nil {
		t.Fatalf("测试目录无效: %v", err)
	}
	if err := Run(context.Background(), repo, filepath.Join(t.TempDir(), "tools"), spec); err == nil {
		t.Fatal("危险归档未被拒绝")
	}
}

func sumHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("mode=%#o want=%#o path=%s", info.Mode().Perm(), want, path)
	}
}

type finalLinkOut struct {
	bytes.Buffer
	parent string
	moved  string
	ran    bool
	err    error
}

func (out *finalLinkOut) Write(data []byte) (int, error) {
	n, err := out.Buffer.Write(data)
	if err != nil || out.ran || !strings.Contains(string(data), "state=installed") {
		return n, err
	}
	out.ran = true
	if err := os.Rename(out.parent, out.moved); err != nil {
		out.err = err
		return n, nil
	}
	out.err = os.Symlink(filepath.Base(out.moved), out.parent)
	return n, nil
}

func TestFinalLink(t *testing.T) {
	useLocalTmp(t)
	spec, repo := makeMany(t, 1)
	base := t.TempDir()
	parent := filepath.Join(base, "parent")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(parent, "tools")
	out := &finalLinkOut{
		parent: parent,
		moved:  filepath.Join(base, "parent-old"),
	}
	runErr := RunTo(context.Background(), repo, dest, spec, out)
	if out.err != nil {
		t.Fatal(out.err)
	}
	if !out.ran {
		t.Fatal("final parent replacement did not run")
	}
	if runErr == nil {
		t.Fatal("install accepted a final intermediate symlink")
	}
}

type compSwapCtx struct {
	context.Context
	once sync.Once
	dest string
	dir  string
	body []byte
	ran  bool
	err  error
}

func (ctx *compSwapCtx) Err() error {
	return ctx.Context.Err()
}

func (ctx *compSwapCtx) Done() <-chan struct{} {
	ctx.once.Do(ctx.swap)
	return ctx.Context.Done()
}

func (ctx *compSwapCtx) swap() {
	work, err := filepath.Glob(filepath.Join(ctx.dest, ".cdh-toolchain-*"))
	if err != nil {
		ctx.err = err
		return
	}
	for _, item := range work {
		comp := filepath.Join(item, ctx.dir)
		got, err := os.ReadFile(filepath.Join(comp, "id.txt"))
		if err != nil || !bytes.Equal(got, ctx.body) {
			continue
		}
		if err := os.Rename(comp, comp+"-old"); err != nil {
			ctx.err = err
			return
		}
		if err := os.Mkdir(comp, 0o755); err != nil {
			ctx.err = err
			return
		}
		if err := os.WriteFile(filepath.Join(comp, "id.txt"), ctx.body, 0o644); err != nil {
			ctx.err = err
			return
		}
		ctx.err = os.WriteFile(filepath.Join(comp, "bad"), []byte("unverified"), 0o644)
		ctx.ran = ctx.err == nil
		return
	}
}

func TestCompSwap(t *testing.T) {
	useLocalTmp(t)
	spec, repo := makeMany(t, 1)
	dest := filepath.Join(t.TempDir(), "tools")
	ctx := &compSwapCtx{
		Context: context.Background(),
		dest:    dest,
		dir:     spec.Archives[0].Dir,
		body:    []byte("Tool 1 1.26.1\n"),
	}
	runErr := Run(ctx, repo, dest, spec)
	if ctx.err != nil {
		t.Fatal(ctx.err)
	}
	if !ctx.ran {
		t.Fatal("staged component replacement did not run")
	}
	if runErr == nil {
		t.Fatal("install published an unchecked staged component")
	}
	bad := filepath.Join(dest, spec.Archives[0].Dir, "bad")
	if _, err := os.Lstat(bad); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replacement component was published: %v", err)
	}
}

type probeRaceCtx struct {
	context.Context
	once    sync.Once
	bin     string
	moved   string
	outside string
	ran     bool
	err     error
}

func (ctx *probeRaceCtx) Done() <-chan struct{} {
	ctx.once.Do(func() {
		ctx.ran = true
		if err := os.Rename(ctx.bin, ctx.moved); err != nil {
			ctx.err = err
			return
		}
		ctx.err = os.Symlink(ctx.outside, ctx.bin)
	})
	return ctx.Context.Done()
}

func (ctx *probeRaceCtx) state() (bool, error) {
	ctx.once.Do(func() {})
	return ctx.ran, ctx.err
}

func TestProbeRace(t *testing.T) {
	useLocalTmp(t)
	spec, repo := makeSample(t)
	mark := filepath.Join(t.TempDir(), "mark")
	spec.Archives[0].Probes[0].Env = []string{"MARK=" + mark}
	dest := filepath.Join(t.TempDir(), "tools")
	if err := Run(context.Background(), repo, dest, spec); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	evil := "#!/bin/sh\n: > \"$MARK\"\necho go1.26.8\n"
	if err := os.WriteFile(filepath.Join(outside, "go"), []byte(evil), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dest, spec.Archives[0].Dir, "bin")
	ctx := &probeRaceCtx{
		Context: context.Background(),
		bin:     bin,
		moved:   bin + "-old",
		outside: outside,
	}
	runErr := Check(ctx, dest, spec)
	ran, hookErr := ctx.state()
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if !ran {
		t.Fatal("post-open probe replacement did not run")
	}
	if _, err := os.Lstat(mark); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replacement probe executed: %v", err)
	}
	if runErr == nil {
		t.Fatal("check accepted a replaced probe path")
	}
}

func TestFIFOFast(t *testing.T) {
	useLocalTmp(t)
	if os.Getenv("CDH_FIFO_CASE") == "1" {
		spec, repo := makeMany(t, 1)
		dest := filepath.Join(t.TempDir(), "tools")
		if err := os.Mkdir(dest, 0o755); err != nil {
			t.Fatal(err)
		}
		fifo := filepath.Join(dest, spec.Archives[0].Dir)
		if err := unix.Mkfifo(fifo, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := Run(context.Background(), repo, dest, spec); err == nil {
			t.Fatal("FIFO was accepted")
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFIFOFast$", "-test.count=1")
	cmd.Env = append(os.Environ(), "CDH_FIFO_CASE=1")
	output, runErr := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("FIFO was not rejected within two seconds:\n%s", output)
	}
	if runErr != nil {
		t.Fatalf("FIFO child failed: %v\n%s", runErr, output)
	}
}

func useLocalTmp(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp, err := os.MkdirTemp(cwd, ".install-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Errorf("remove local temp: %v", err)
		}
	})
	t.Setenv("TMPDIR", tmp)
}

func TestMkdirRace(t *testing.T) {
	useLocalTmp(t)
	base := t.TempDir()
	root, err := openRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	calls := 0
	mk := func(fd int, name string, mode uint32) error {
		calls++
		if err := unix.Mkdirat(fd, name, 0o775); err != nil {
			return err
		}
		if err := os.Chmod(filepath.Join(base, name), 0o775); err != nil {
			return err
		}
		return unix.EEXIST
	}
	dir, runErr := dirAtWith(root, "race", true, 0o755, mk)
	if dir != nil {
		dir.Close()
	}
	if calls != 1 {
		t.Fatalf("Mkdirat calls=%d", calls)
	}
	if runErr == nil {
		t.Fatal("Mkdirat EEXIST was accepted")
	}
	assertMode(t, filepath.Join(base, "race"), 0o775)
}
