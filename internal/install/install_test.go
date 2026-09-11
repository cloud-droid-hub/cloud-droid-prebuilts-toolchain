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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/ulikunitz/xz"
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
	if err := reader.finish(spec.Archives[0].SHA256); err != nil {
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

func TestOldReplace(t *testing.T) {
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
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("旧目录仍位于安装目标: %v", err)
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
