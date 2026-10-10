// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

type rule struct {
	From string
	To   string
	Tree bool
	Exec bool
	Thin bool
	Glob string
}

type input struct {
	Name    string
	SHA256  string
	Kind    string
	Prefix  string
	Rules   []rule
	License bool
	Drop    []string
	URL     string
}

type extra struct {
	Path string
	Data string
	Mode int64
}

type spec struct {
	Host     string
	Comp     string
	Prefix   string
	OldSHA   string
	Inputs   []input
	Extras   []extra
	Drop     []string
	Aliases  map[string]string
	Links    map[string]string
	Packages []pkgDef
	Patch    bool
	Dirs     []string
	DataOnly bool
	ForceR   bool
	PreRPath map[string]string
	RunPaths map[string]bool
}
