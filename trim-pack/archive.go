// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

type node struct {
	Kind byte
	Mode int64
	Off  int64
	Size int64
	Link string
	Thin bool
	Data []byte
}

type bundle struct {
	Items   map[string]*node
	Drop    []string
	Size    int64
	License string
}

func cleanName(name string) (string, error) {
	name = strings.TrimPrefix(strings.TrimSuffix(name, "/"), "./")
	if name == "." || name == "" {
		return "", nil
	}
	if path.IsAbs(name) || path.Clean(name) != name || name == ".." ||
		strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00\t\r\n") {
		return "", fmt.Errorf("归档路径无效：%q", name)
	}
	return name, nil
}

func chosen(in input, name string) (string, rule, error) {
	var found rule
	var dest string
	if in.Prefix != "" {
		if !strings.HasPrefix(name, in.Prefix+"/") {
			return "", found, nil
		}
		name = strings.TrimPrefix(name, in.Prefix+"/")
	}
	for _, pick := range in.Rules {
		match := name == pick.From || pick.Tree && strings.HasPrefix(name, pick.From+"/")
		if pick.From == "" && pick.Tree {
			match = true
		}
		if pick.Glob != "" {
			var err error
			rel := name
			if pick.From != "" {
				rel = strings.TrimPrefix(name, pick.From+"/")
				match = strings.HasPrefix(name, pick.From+"/")
			} else {
				match = true
			}
			hit, next := path.Match(pick.Glob, rel)
			match, err = match && hit, next
			if err != nil {
				return "", found, err
			}
		}
		if !match {
			continue
		}
		rel := name
		if pick.From != "" {
			rel = strings.TrimPrefix(strings.TrimPrefix(name, pick.From), "/")
		}
		got := path.Join(pick.To, rel)
		if dest != "" && dest != got {
			return "", found, fmt.Errorf("选取规则冲突：%s", name)
		}
		dest, found = got, pick
	}
	return dest, found, nil
}

func (b *bundle) add(ctx context.Context, plan spec, in input, name string, kind byte,
	mode, size int64, link string, raw io.Reader, data *os.File) error {
	name, err := cleanName(name)
	if err != nil || name == "" {
		return err
	}
	rel := name
	if in.Prefix != "" {
		rel = strings.TrimPrefix(name, in.Prefix+"/")
	}
	for _, omit := range in.Drop {
		match, err := path.Match(omit, rel)
		if err != nil {
			return err
		}
		if match || strings.HasPrefix(rel, omit+"/") {
			b.Drop = append(b.Drop, in.Name+":"+rel)
			return nil
		}
	}
	dest, pick, err := chosen(in, name)
	if err != nil {
		return err
	}
	if dest == "" {
		if kind != tar.TypeDir {
			b.Drop = append(b.Drop, in.Name+":"+rel)
		}
		return nil
	}
	for _, omit := range plan.Drop {
		match, err := path.Match(omit, dest)
		if err != nil {
			return err
		}
		if match || strings.HasPrefix(dest, omit+"/") {
			b.Drop = append(b.Drop, dest)
			return nil
		}
	}
	if _, err := cleanName(dest); err != nil {
		return err
	}
	if prior := b.Items[dest]; prior != nil {
		if prior.Kind == tar.TypeDir && kind == tar.TypeDir {
			return nil
		}
		return fmt.Errorf("输出文件重复：%s", dest)
	}
	item := &node{Kind: kind, Mode: 0o644, Size: size, Thin: pick.Thin}
	if !plan.DataOnly && (mode&0o111 != 0 || pick.Exec) {
		item.Mode = 0o755
	}
	switch kind {
	case tar.TypeDir:
		item.Mode, item.Size = 0o755, 0
	case tar.TypeReg, tar.TypeRegA:
		item.Kind = tar.TypeReg
		item.Off, err = data.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}
		n, err := io.Copy(data, ctxRead{ctx: ctx, in: raw})
		if err != nil || n != size {
			return fmt.Errorf("读取%s bytes=%d want=%d：%v", name, n, size, err)
		}
	case tar.TypeLink, tar.TypeSymlink:
		if path.IsAbs(link) || strings.ContainsAny(link, "\\\x00\t\r\n") {
			return fmt.Errorf("链接越界：%s -> %s", name, link)
		}
		if kind == tar.TypeSymlink {
			link = path.Join(path.Dir(name), link)
		}
		item.Link, _, err = chosen(in, link)
		if err != nil || item.Link == "" {
			return fmt.Errorf("链接目标没有登记：%s -> %s：%v", name, link, err)
		}
		item.Size = 0
	default:
		return fmt.Errorf("选中不支持的文件类型：%s type=%d", name, kind)
	}
	b.Items[dest] = item
	return nil
}

func readInput(ctx context.Context, plan spec, in input, raw *os.File, b *bundle, data *os.File) (ret error) {
	info, err := raw.Stat()
	if err != nil {
		return err
	}
	if in.License {
		body, err := io.ReadAll(io.LimitReader(raw, 10*1024*1024))
		if err != nil {
			return err
		}
		start := strings.Index(string(body), "  <license id=\"android-sdk-license\"")
		end := strings.Index(string(body), "</license>")
		if start < 0 || end < start {
			return errors.New("固定仓库XML缺少Android SDK许可文本")
		}
		b.License = string(body[start:end+len("</license>")]) + "\n"
		return nil
	}
	if in.Kind == "raw" {
		return b.add(ctx, plan, in, in.Name, tar.TypeReg, 0o644, info.Size(), "", raw, data)
	}
	if in.Kind == "zip" {
		zr, err := zip.NewReader(raw, info.Size())
		if err != nil {
			return err
		}
		for _, file := range zr.File {
			fr, err := file.Open()
			if err != nil {
				return err
			}
			kind, link := byte(tar.TypeReg), ""
			if file.Mode().IsDir() {
				kind = tar.TypeDir
			} else if file.Mode()&os.ModeSymlink != 0 {
				kind = tar.TypeSymlink
				if file.UncompressedSize64 > 4096 {
					fr.Close()
					return errors.New("ZIP链接目标超过4096字节")
				}
				buf, err := io.ReadAll(fr)
				if err != nil {
					fr.Close()
					return err
				}
				link = string(buf)
			}
			err = b.add(ctx, plan, in, file.Name, kind, int64(file.Mode().Perm()), int64(file.UncompressedSize64), link, fr, data)
			end := fr.Close()
			if err != nil || end != nil {
				return errors.Join(err, end)
			}
		}
		return nil
	}
	var source io.Reader = ctxRead{ctx: ctx, in: raw}
	kind := in.Kind
	if kind == "deb" {
		source, kind, err = debData(raw)
		if err != nil {
			return err
		}
		source = ctxRead{ctx: ctx, in: source}
	}
	switch kind {
	case "tar.gz":
		gr, err := gzip.NewReader(source)
		if err != nil {
			return err
		}
		defer gr.Close()
		source = gr
	case "tar.xz":
		if runtime.GOOS == "darwin" && in.Kind == "tar.xz" {
			var finish func() error
			source, finish, err = nativeStream(ctx, raw)
			if err != nil {
				return err
			}
			defer func() { ret = errors.Join(ret, finish()) }()
		} else {
			source, err = xz.NewReader(source)
			if err != nil {
				return err
			}
		}
	case "tar.zst":
		zr, err := zstd.NewReader(source, zstd.WithDecoderConcurrency(1))
		if err != nil {
			return err
		}
		defer zr.Close()
		source = zr
	case "tar":
	default:
		return fmt.Errorf("未知原始格式：%s", in.Kind)
	}
	tr := tar.NewReader(source)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := b.add(ctx, plan, in, h.Name, h.Typeflag, h.Mode, h.Size, h.Linkname, tr, data); err != nil {
			return err
		}
	}
	_, err = io.Copy(io.Discard, source)
	return err
}

func (b *bundle) resolve(name string) (*node, error) {
	seen := make(map[string]bool)
	for {
		if seen[name] {
			return nil, fmt.Errorf("链接循环：%s", name)
		}
		seen[name] = true
		item := b.Items[name]
		if item == nil {
			return nil, fmt.Errorf("必需文件或链接目标缺失：%s", name)
		}
		if item.Kind == tar.TypeReg {
			return item, nil
		}
		if item.Kind != tar.TypeSymlink && item.Kind != tar.TypeLink {
			return nil, fmt.Errorf("目标不是普通文件：%s", name)
		}
		name = item.Link
	}
}

func thin(item *node, data *os.File) error {
	raw := io.NewSectionReader(data, item.Off, item.Size)
	var magic [4]byte
	if _, err := raw.ReadAt(magic[:], 0); err != nil {
		return nil
	}
	value := binary.BigEndian.Uint32(magic[:])
	if value != macho.MagicFat && value != macho.Magic32 && value != macho.Magic64 &&
		binary.LittleEndian.Uint32(magic[:]) != macho.Magic32 && binary.LittleEndian.Uint32(magic[:]) != macho.Magic64 {
		return nil
	}
	fat, err := macho.NewFatFile(raw)
	if err == nil {
		for _, arch := range fat.Arches {
			if arch.Cpu == macho.CpuArm64 {
				item.Off += int64(arch.Offset)
				item.Size = int64(arch.Size)
				return nil
			}
		}
		return errors.New("Mach-O通用文件没有arm64切片")
	}
	if !errors.Is(err, macho.ErrNotFat) {
		return err
	}
	file, err := macho.NewFile(raw)
	if err != nil {
		return err
	}
	if file.Cpu != macho.CpuArm64 {
		return errors.New("Mach-O原始文件不是arm64")
	}
	return nil
}

func finish(plan spec, b *bundle, data *os.File) error {
	for name, item := range b.Items {
		if item.Kind != tar.TypeDir {
			target, err := b.resolve(name)
			if err != nil {
				return err
			}
			if target.Thin {
				if err := thin(target, data); err != nil {
					return fmt.Errorf("arm64切片%s：%w", name, err)
				}
				target.Thin = false
			}
			if link, ok := plan.Links[name]; ok {
				if item.Kind != tar.TypeSymlink || path.Join(path.Dir(name), link) != item.Link {
					return fmt.Errorf("允许链接不符：%s", name)
				}
			} else {
				*item = *target
			}
		}
	}
	for name, from := range plan.Aliases {
		if b.Items[name] != nil {
			return fmt.Errorf("别名已存在：%s", name)
		}
		target, err := b.resolve(from)
		if err != nil {
			return err
		}
		copy := *target
		b.Items[name] = &copy
	}
	for _, name := range plan.Dirs {
		if prior := b.Items[name]; prior != nil && prior.Kind != tar.TypeDir {
			return fmt.Errorf("生成目录与文件冲突：%s", name)
		}
		b.Items[name] = &node{Kind: tar.TypeDir, Mode: 0o755}
	}
	for _, add := range plan.Extras {
		if b.Items[add.Path] != nil {
			return fmt.Errorf("生成元数据已存在：%s", add.Path)
		}
		b.Items[add.Path] = &node{Kind: tar.TypeReg, Mode: add.Mode, Size: int64(len(add.Data)), Data: []byte(add.Data)}
	}
	for _, pkg := range plan.Packages {
		name := strings.ReplaceAll(pkg.Path, ";", "/") + "/package.xml"
		if b.Items[name] != nil {
			return fmt.Errorf("输入已包含生成包元数据：%s", name)
		}
		body, err := packageXML(pkg, b.License)
		if err != nil {
			return err
		}
		b.Items[name] = &node{Kind: tar.TypeReg, Mode: 0o644, Size: int64(len(body)), Data: []byte(body)}
	}
	for name, item := range b.Items {
		for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
			if prior := b.Items[dir]; prior != nil && prior.Kind != tar.TypeDir {
				return fmt.Errorf("父路径不是目录：%s", dir)
			}
			if b.Items[dir] == nil {
				b.Items[dir] = &node{Kind: tar.TypeDir, Mode: 0o755}
			}
		}
		b.Size += item.Size
	}
	sort.Strings(b.Drop)
	return nil
}

func writeArc(ctx context.Context, plan spec, b *bundle, data *os.File, out io.Writer) error {
	gw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(gw)
	names := []string{""}
	for name := range b.Items {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		item := &node{Kind: tar.TypeDir, Mode: 0o755}
		if name != "" {
			item = b.Items[name]
		}
		head := tar.Header{Name: path.Join(plan.Prefix, name), Typeflag: item.Kind, Mode: item.Mode,
			Size: item.Size, Linkname: item.Link, ModTime: time.Unix(0, 0), Format: tar.FormatPAX}
		if link, ok := plan.Links[name]; ok {
			head.Linkname = link
		}
		if item.Kind == tar.TypeDir {
			head.Name += "/"
		}
		if err := tw.WriteHeader(&head); err != nil {
			return err
		}
		if item.Kind != tar.TypeReg {
			continue
		}
		var raw io.Reader = io.NewSectionReader(data, item.Off, item.Size)
		if item.Data != nil {
			raw = strings.NewReader(string(item.Data))
		}
		if n, err := io.Copy(tw, ctxRead{ctx: ctx, in: raw}); err != nil || n != item.Size {
			return fmt.Errorf("写入%s bytes=%d want=%d：%v", name, n, item.Size, err)
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}
