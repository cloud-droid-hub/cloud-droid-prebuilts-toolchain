// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const outPrefix = "jdk-21.0.12.1+1"
const profile = "jdk21-build-v1"
const markName = "cdh-jdk-trim.txt"
const partLimit int64 = 90 * 1024 * 1024

var mustFiles = []string{"bin/java", "bin/javac", "bin/jar", "bin/keytool", "lib/modules",
	"lib/ct.sym", "lib/security/cacerts", "include/jni.h", "legal/java.base/LICENSE", "release", "NOTICE"}

type plan struct {
	Host   string
	Prefix string
	SrcSHA string
	Source string
	Stem   string
	Sizes  []int64
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(os.Stdout, "用法：jdk-trim pack|verify --host <宿主> --source <上游归档或分片目录> --dest <制品目录>")
		return err
	}
	if len(args) == 0 {
		return errors.New("用法：jdk-trim pack|verify --host <宿主> --source <上游归档或分片目录> --dest <制品目录>")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	host := fs.String("host", "", "darwin-arm64、linux-arm64或linux-amd64")
	in := fs.String("source", "", "固定版本的完整上游tar.gz或原始分片目录")
	dest := fs.String("dest", "", "pack创建新目录；verify只读核验目录")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 || *in == "" || *dest == "" {
		return errors.New("source与dest必填，不能附带位置参数")
	}
	spec, err := hostPlan(*host)
	if err != nil {
		return err
	}
	switch args[0] {
	case "pack":
		rep, err := pack(ctx, spec, *in, *dest)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(os.Stdout, "host=%s profile=%s source=%s archive=%s before=%d after=%d bytes=%d parts=%d exit=0\n",
			rep.Host, rep.Profile, rep.SourceSHA, rep.ArchiveSHA, rep.Before, rep.After, rep.Bytes, len(rep.Parts))
		return err
	case "verify":
		if err := verify(ctx, spec, *in, *dest); err != nil {
			return err
		}
		_, err := fmt.Fprintf(os.Stdout, "host=%s profile=%s replay=identical exit=0\n", spec.Host, profile)
		return err
	default:
		return errors.New("未知操作，只支持pack与verify")
	}
}

func hostPlan(host string) (plan, error) {
	for _, spec := range plans {
		if spec.Host == host {
			return spec, nil
		}
	}
	return plan{}, fmt.Errorf("未登记宿主：%q", host)
}
