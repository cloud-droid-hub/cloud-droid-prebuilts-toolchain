// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

import "testing"

func TestBashCatalog(t *testing.T) {
	spec := Current()
	arc := bash53()
	if arc.Name != "Bash" || arc.Version != "5.3.15" || arc.Dir != "bash53" || len(arc.Probes) != 1 || arc.Probes[0].Machine != ELFARM64 {
		t.Fatalf("Bash登记不完整：%+v", arc)
	}
	if err := Valid(Spec{Host: spec.Host, Ready: true, Archives: []Archive{arc}}); err != nil {
		t.Fatal(err)
	}
}
