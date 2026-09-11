// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		usage(stderr)
		return 2
	}
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	repo, dest, ok := parseArgs("install", args, stderr)
	if !ok {
		return 2
	}
	if err := install.Run(context.Background(), repo, dest, catalog.Current()); err != nil {
		fmt.Fprintf(stderr, "install failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "installed %s\n", dest)
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	_, dest, ok := parseArgs("check", args, stderr)
	if !ok {
		return 2
	}
	if err := install.Check(context.Background(), dest, catalog.Current()); err != nil {
		fmt.Fprintf(stderr, "check failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "checked %s\n", dest)
	return 0
}

func parseArgs(name string, args []string, stderr io.Writer) (string, string, bool) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	dest := flags.String("dest", "", "installation root")
	if err := flags.Parse(args); err != nil {
		return "", "", false
	}
	if flags.NArg() != 0 || *dest == "" {
		fmt.Fprintln(stderr, "--dest is required and positional arguments are not accepted")
		return "", "", false
	}
	return *repo, *dest, true
}

func usage(output io.Writer) {
	fmt.Fprintln(output, "usage: cdh-toolchain <install|check> --repo DIR --dest DIR")
}
