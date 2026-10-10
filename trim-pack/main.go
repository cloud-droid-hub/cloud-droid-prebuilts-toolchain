// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

const partLimit int64 = 90 * 1024 * 1024

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("用法：trim-pack pack|verify|inputs --component <组件> --host <宿主>；pack/verify还需 --source <原始源目录> --dest <制品目录> [--published <现有分片目录>]")
	}
	if args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprintln(out, "trim-pack pack|verify|inputs --component <组件> --host <宿主>；pack/verify还需 --source <原始源目录> --dest <制品目录> [--published <现有分片目录>]")
		return err
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	host := fs.String("host", "", "darwin-arm64、linux-arm64或linux-amd64")
	comp := fs.String("component", "", "登记的裁剪组件")
	source := fs.String("source", "", "包含固定摘要原始包的目录")
	dest := fs.String("dest", "", "pack创建新目录；verify只读核验")
	old := fs.String("published", "", "verify可选核对现有归档的文件集合与字节")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 || args[0] != "inputs" && (*source == "" || *dest == "") {
		return errors.New("pack/verify的source、dest必填，不能使用位置参数")
	}
	plan, err := lookup(*host, *comp)
	if err != nil {
		return err
	}
	switch args[0] {
	case "inputs":
		for _, in := range plan.Inputs {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\n", in.Name, in.SHA256, origin(in)); err != nil {
				return err
			}
		}
		return nil
	case "pack":
		if *old != "" {
			return errors.New("pack不读取published，选取规则只来自编译期定义")
		}
		return generate(ctx, plan, *source, *dest, out)
	case "verify":
		return verify(ctx, plan, *source, *dest, *old, out)
	default:
		return errors.New("未知操作，只支持pack、verify与inputs")
	}
}
