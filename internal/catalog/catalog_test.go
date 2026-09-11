// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package catalog

import "testing"

func TestCatalog(t *testing.T) {
	spec := Current()
	if spec.Host != "linux-amd64" {
		t.Fatalf("host=%q", spec.Host)
	}
	if !spec.Ready || len(spec.Archives) != 1 {
		t.Fatalf("ready=%v archives=%d", spec.Ready, len(spec.Archives))
	}
	archive := spec.Archives[0]
	if archive.Name != "Go" || archive.Version != "1.26.8" || archive.Dir != "go126" {
		t.Fatalf("archive=%+v", archive)
	}
	if err := Valid(spec); err != nil {
		t.Fatal(err)
	}
}

func TestBadCatalog(t *testing.T) {
	base := Spec{
		Host:  "linux-amd64",
		Ready: true,
		Archives: []Archive{{
			Name: "Go", Version: "1.26.8", Dir: "go126", Format: TarGzip,
			Prefix: "payload",
			SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			Parts: []Part{{
				Path: "payload/go126/go.tar.gz.part-000", Size: 1,
				SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			}},
			Probes: []Probe{{Path: "bin/go", Want: []string{"go1.26.8"}, Machine: ELFAMD64}},
		}},
	}
	tests := []struct {
		name string
		edit func(*Spec)
	}{
		{name: "oversize", edit: func(spec *Spec) { spec.Archives[0].Parts[0].Size = 94371841 }},
		{name: "bad-sum", edit: func(spec *Spec) { spec.Archives[0].Parts[0].SHA256 = "z" + spec.Archives[0].Parts[0].SHA256[1:] }},
		{name: "absolute", edit: func(spec *Spec) { spec.Archives[0].Parts[0].Path = "/payload/part" }},
		{name: "escape", edit: func(spec *Spec) { spec.Archives[0].Parts[0].Path = "../part" }},
		{name: "bad-prefix", edit: func(spec *Spec) { spec.Archives[0].Prefix = "../payload" }},
		{name: "no-probes", edit: func(spec *Spec) { spec.Archives[0].Probes = nil }},
		{name: "empty-want", edit: func(spec *Spec) { spec.Archives[0].Probes[0].Want = nil }},
		{name: "version-miss", edit: func(spec *Spec) { spec.Archives[0].Probes[0].Want = []string{"go"} }},
		{name: "wrong-machine", edit: func(spec *Spec) { spec.Archives[0].Probes[0].Machine = ELFARM64 }},
		{name: "no-native", edit: func(spec *Spec) { spec.Archives[0].Probes[0].Machine = Script }},
		{name: "data-native", edit: func(spec *Spec) { spec.Archives[0].DataOnly = true }},
		{name: "bad-env", edit: func(spec *Spec) { spec.Archives[0].Probes[0].Env = []string{"BROKEN"} }},
		{name: "dup-env", edit: func(spec *Spec) { spec.Archives[0].Probes[0].Env = []string{"A=1", "A=2"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := base
			spec.Archives = append([]Archive(nil), base.Archives...)
			spec.Archives[0].Parts = append([]Part(nil), base.Archives[0].Parts...)
			spec.Archives[0].Probes = append([]Probe(nil), base.Archives[0].Probes...)
			spec.Archives[0].Probes[0].Want = append([]string(nil), base.Archives[0].Probes[0].Want...)
			test.edit(&spec)
			if err := Valid(spec); err == nil {
				t.Fatal("无效目录未被拒绝")
			}
		})
	}
}
