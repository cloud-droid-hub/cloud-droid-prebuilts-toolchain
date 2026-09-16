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
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

type options struct {
	host   string
	parts  string
	prefix string
	dest   string
	work   string
}

func main() {
	start := time.Now()
	err := writeLog(os.Stdout, "command=%q started=%s runtime=%s/%s\n", os.Args, start.UTC().Format(time.RFC3339), runtime.GOOS, runtime.GOARCH)
	var opts options
	if err == nil {
		opts, err = parse(os.Args[1:], os.Stderr)
	}
	if err == flag.ErrHelp {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err == nil {
		err = check(ctx, opts, os.Stdout)
	}
	if code := finish(os.Stdout, os.Stderr, start, err); code != 0 {
		os.Exit(code)
	}
}

func parse(args []string, log io.Writer) (options, error) {
	var opts options
	w := logSink{out: log}
	flags := flag.NewFlagSet("llvm22-check", flag.ContinueOnError)
	flags.SetOutput(&w)
	flags.StringVar(&opts.host, "host", "", "darwin-arm64, linux-arm64 or linux-amd64")
	flags.StringVar(&opts.parts, "parts", "", "directory containing only one archive's ordered parts")
	flags.StringVar(&opts.prefix, "prefix", "", "archive top-level directory")
	flags.StringVar(&opts.dest, "install-dir", "", "new installation directory under repository build/output")
	flags.StringVar(&opts.work, "probe-dir", "", "new native probe directory under repository build/output")
	err := flags.Parse(args)
	if w.err != nil {
		return opts, fmt.Errorf("write argument output: %w", w.err)
	}
	if err != nil {
		return opts, err
	}
	if flags.NArg() != 0 || opts.host == "" || opts.parts == "" || opts.prefix == "" || opts.dest == "" || opts.work == "" {
		return opts, errors.New("host, parts, prefix, install-dir and probe-dir are required")
	}
	return opts, nil
}

func native(cfg hostCfg, goos, arch string) error {
	if cfg.goos != goos || cfg.arch != arch {
		return fmt.Errorf("host %s requires native %s/%s, process is %s/%s", cfg.host, cfg.goos, cfg.arch, goos, arch)
	}
	return nil
}

func related(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func check(ctx context.Context, opts options, log io.Writer) error {
	cfg, err := hostConfig(opts.host)
	if err != nil {
		return err
	}
	if err := native(cfg, runtime.GOOS, runtime.GOARCH); err != nil {
		return err
	}
	for _, value := range []*string{&opts.parts, &opts.dest, &opts.work} {
		*value, err = filepath.Abs(*value)
		if err != nil {
			return err
		}
	}
	if related(opts.dest, opts.work) || related(opts.work, opts.dest) || related(opts.dest, opts.parts) || related(opts.work, opts.parts) ||
		related(opts.parts, opts.dest) || related(opts.parts, opts.work) {
		return errors.New("input and output trees must be separate")
	}
	for _, dir := range []string{opts.dest, opts.work} {
		if err := absent(dir); err != nil {
			return err
		}
		if err := freeSpace(filepath.Dir(dir), probeSpace); err != nil {
			return err
		}
	}
	pack, err := makeSpec(cfg, opts.parts, opts.prefix, log)
	if err != nil {
		return err
	}
	if pack.need > ^uint64(0)-probeSpace {
		return errors.New("installation size overflows")
	}
	if err := freeSpace(filepath.Dir(opts.dest), pack.need+probeSpace); err != nil {
		return err
	}
	if err := os.Mkdir(opts.dest, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(opts.work, 0o755); err != nil {
		return err
	}
	if err := install.RunTo(ctx, opts.parts, opts.dest, pack.spec, log); err != nil {
		return err
	}
	if err := install.Check(ctx, opts.dest, pack.spec); err != nil {
		return err
	}
	root := filepath.Join(opts.dest, "llvm22")
	if err := audit(root, true, pack.ids); err != nil {
		return err
	}
	if err := runProbe(ctx, cfg, root, opts.work, log); err != nil {
		return err
	}
	if err := audit(opts.work, false, pack.ids); err != nil {
		return err
	}
	if err := freeSpace(opts.dest, 0); err != nil {
		return err
	}
	if err := freeSpace(opts.work, 0); err != nil {
		return err
	}
	return writeLog(log, "host=%s installed=true native_probe=true regular_files=%d independent=true\n", cfg.host, len(pack.ids))
}
