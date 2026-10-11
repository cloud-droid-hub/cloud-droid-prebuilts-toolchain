// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/selectcfg"
)

func main() {
	ctx, stop := signalCtx()
	code := runCtx(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) int {
	return runCtx(context.Background(), args, stdout, stderr)
}

func runCtx(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runWith(ctx, args, stdout, stderr, catalog.Current())
}

func runWith(ctx context.Context, args []string, stdout, stderr io.Writer, spec catalog.Spec) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "install":
		return runInstall(ctx, args[1:], stdout, stderr, spec)
	case "check":
		return runCheck(ctx, args[1:], stdout, stderr, spec)
	case "select":
		return runSelect(ctx, args[1:], stdout, stderr, spec)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		usage(stderr)
		return 2
	}
}

func signalCtx() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func runSelect(ctx context.Context, args []string, stdout, stderr io.Writer, spec catalog.Spec) int {
	flags := flag.NewFlagSet("select", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dest := flags.String("dest", "", "installation root")
	name := flags.String("name", "", "tool selection name")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *dest == "" || *name == "" {
		fmt.Fprintln(stderr, "--dest and --name are required; positional arguments are not accepted")
		return 2
	}
	selected, err := selectcfg.Resolve(*dest, *name, spec)
	if err != nil {
		fmt.Fprintf(stderr, "select failed: %v\n", err)
		return 1
	}
	if err := install.Check(ctx, *dest, spec); err != nil {
		fmt.Fprintf(stderr, "select failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, selected)
	return 0
}

func runInstall(ctx context.Context, args []string, stdout, stderr io.Writer, spec catalog.Spec) int {
	repo, dest, spec, all, ok := parseArgs("install", args, stderr, spec)
	if !ok {
		return 2
	}
	if err := install.RunTo(ctx, repo, dest, spec, stdout); err != nil {
		fmt.Fprintf(stderr, "install failed: %v\n", err)
		return 1
	}
	if all {
		if err := install.CheckLegacy(repo, dest, spec.Host); err != nil {
			fmt.Fprintf(stderr, "install failed: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "installed %s\n", dest)
	return 0
}

func runCheck(ctx context.Context, args []string, stdout, stderr io.Writer, spec catalog.Spec) int {
	_, dest, spec, _, ok := parseArgs("check", args, stderr, spec)
	if !ok {
		return 2
	}
	if err := install.Check(ctx, dest, spec); err != nil {
		fmt.Fprintf(stderr, "check failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "checked %s\n", dest)
	return 0
}

func parseArgs(name string, args []string, stderr io.Writer, spec catalog.Spec) (string, string, catalog.Spec, bool, bool) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	dest := flags.String("dest", "", "installation root")
	component := flags.String("component", "", "只处理指定安装目录的组件")
	if err := flags.Parse(args); err != nil {
		return "", "", spec, false, false
	}
	if flags.NArg() != 0 || *dest == "" {
		fmt.Fprintln(stderr, "--dest is required and positional arguments are not accepted")
		return "", "", spec, false, false
	}
	if *component != "" {
		found := false
		for _, arc := range spec.Archives {
			if arc.Dir == *component {
				spec.Archives = []catalog.Archive{arc}
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(stderr, "unknown component: %s\n", *component)
			return "", "", spec, false, false
		}
	}
	return *repo, *dest, spec, *component == "", true
}

func usage(output io.Writer) {
	fmt.Fprintln(output, "usage: cdh-toolchain <install|check|select> --repo DIR --dest DIR [--name NAME] [--component DIR]")
}
