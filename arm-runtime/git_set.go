// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
)

type sourceRepo struct {
	prefix string
	dir    string
	picks  []string
	state  gitState
}

func relPick(name string) bool {
	return name != "" && name != "." && name != ".." && !path.IsAbs(name) && path.Clean(name) == name &&
		!strings.HasPrefix(name, "../") && !strings.ContainsAny(name, "\\\x00\r\n:*?[")
}

func gitSet(inputs, paths []string, dest, work string, check bool) (returnErr error) {
	if len(inputs) == 0 {
		return fmt.Errorf("Git source set requires at least one --repo")
	}
	repos := make([]sourceRepo, 0, len(inputs))
	for _, input := range inputs {
		prefix, dir, ok := strings.Cut(input, "=")
		if !ok || !relPick(prefix) || dir == "" {
			return fmt.Errorf("invalid Git source set member: %q", input)
		}
		for _, repo := range repos {
			if prefix == repo.prefix || strings.HasPrefix(prefix, repo.prefix+"/") || strings.HasPrefix(repo.prefix, prefix+"/") {
				return fmt.Errorf("overlapping Git source prefixes: %s", prefix)
			}
		}
		state, err := gitClean(dir)
		if err != nil {
			return err
		}
		repos = append(repos, sourceRepo{prefix: prefix, dir: dir, state: state})
	}
	for _, pick := range paths {
		prefix, name, ok := strings.Cut(pick, "=")
		if !ok || !relPick(name) {
			return fmt.Errorf("invalid Git source set selection: %q", pick)
		}
		found := false
		for index := range repos {
			if repos[index].prefix == prefix {
				repos[index].picks = append(repos[index].picks, name)
				found = true
			}
		}
		if !found {
			return fmt.Errorf("selection has no registered source: %s", prefix)
		}
	}
	tree, err := newTree(work)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, tree.close()) }()
	tree.raw = true
	for _, repo := range repos {
		if err := tree.readGitRepo(repo); err != nil {
			return err
		}
	}
	for _, repo := range repos {
		after, err := gitClean(repo.dir)
		if err != nil || after != repo.state {
			return fmt.Errorf("source Git state changed while reading the set: %s", repo.prefix)
		}
	}
	if check {
		err = tree.check(dest)
	} else {
		err = tree.emit(dest, "")
	}
	if err != nil {
		return err
	}
	for _, repo := range repos {
		fmt.Printf("git_prefix=%q git_url=%q git_state=clean\n", repo.prefix, repo.state.url)
	}
	return nil
}

func (tree *tarTree) readGitRepo(repo sourceRepo) error {
	args := []string{"-C", repo.dir, "archive", "--format=tar", "--prefix=" + repo.prefix + "/", repo.state.rev}
	if len(repo.picks) > 0 {
		args = append(append(args, "--"), repo.picks...)
	}
	cmd := exec.Command("git", args...)
	cmd.Env = gitEnv()
	cmd.Stderr = io.Discard
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	err = tree.readTar(reader, func(name string) bool { return name != "." })
	if err != nil {
		cmd.Process.Kill()
	} else {
		_, err = io.Copy(io.Discard, reader)
	}
	wait := cmd.Wait()
	if err != nil {
		return err
	}
	if wait != nil {
		return fmt.Errorf("Git source archive failed for %s: %w", repo.prefix, wait)
	}
	return nil
}
