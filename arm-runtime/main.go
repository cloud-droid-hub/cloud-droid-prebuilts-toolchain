package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: arm-runtime rust|debs|config|source|git|git-check|git-refresh|git-set|git-set-check|git-set-refresh|cargo-bin [options]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	src := flags.String("src", "", "registered ARM Rust repository or downloaded .deb directory")
	dest := flags.String("dest", "", "destination; new for extraction, existing for git-check")
	flags.String("work", "", "legacy argument; ordinary work uses the current directory")
	llvm := flags.String("llvm", "", "native LLVM 22 installation for config")
	sysroot := flags.String("sysroot", "", "ARM GNU sysroot for config")
	sum := flags.String("sha256", "", "registered source archive SHA-256")
	var paths []string
	var repos []string
	flags.Func("repo", "Git source set member as prefix=directory; may be repeated", func(value string) error {
		repos = append(repos, value)
		return nil
	})
	flags.Func("path", "Git source path to select; may be repeated", func(value string) error {
		paths = append(paths, value)
		return nil
	})
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *dest == "" {
		return fmt.Errorf("--dest is required; positional arguments are not accepted")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := space(cwd); err != nil {
		return err
	}
	switch args[0] {
	case "rust":
		if *src == "" {
			return fmt.Errorf("rust requires --src")
		}
		return rustData(*src, *dest, "")
	case "debs":
		if *src == "" {
			return fmt.Errorf("debs requires --src")
		}
		return debData(*src, *dest, "")
	case "config":
		return config(*dest, *llvm, *sysroot)
	case "source":
		return sourceData(*src, *dest, "", *sum)
	case "git":
		return gitSelect(*src, *dest, "", false, paths)
	case "git-check":
		return gitSelect(*src, *dest, "", true, paths)
	case "git-set", "git-set-check":
		return gitSet(repos, paths, *dest, "", args[0] == "git-set-check")
	case "git-refresh", "git-set-refresh":
		kind := "git"
		if args[0] == "git-set-refresh" {
			kind = "git-set"
		}
		call := []string{kind, "--src", *src, "--dest", *dest}
		for _, repo := range repos {
			call = append(call, "--repo", repo)
		}
		for _, pick := range paths {
			call = append(call, "--path", pick)
		}
		return refreshCopy(call, *dest)
	case "cargo-bin":
		if *src == "" {
			return fmt.Errorf("cargo-bin requires --src")
		}
		return cargoBin(*src, *dest, "")
	default:
		return fmt.Errorf("unknown operation: %s", args[0])
	}
}
