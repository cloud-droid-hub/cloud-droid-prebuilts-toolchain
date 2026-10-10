// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

import "testing"

func TestPython(t *testing.T) {
	spec := Current()
	for _, arc := range spec.Archives {
		if arc.Dir != "python314" {
			continue
		}
		if arc.Name != "Python" || arc.Version != "3.14.8" || len(arc.Files) == 0 || len(arc.Probes) < 2 {
			t.Fatalf("Python登记不完整：%+v", arc)
		}
		if err := Valid(Spec{Host: spec.Host, Ready: true, Archives: []Archive{arc}}); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatal("工具链未登记Python3.14")
}
