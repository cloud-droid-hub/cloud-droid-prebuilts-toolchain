// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import "fmt"

func lookup(host, comp string) (spec, error) {
	if host != "darwin-arm64" && host != "linux-arm64" && host != "linux-amd64" {
		return spec{}, fmt.Errorf("未登记的宿主：%s", host)
	}
	switch comp {
	case "llvm22":
		if host == "darwin-arm64" {
			return llvmMac(), nil
		}
		return llvmLinux(host), nil
	case "sdk37":
		return sdk37(host), nil
	case "ndk29":
		return ndk29(host), nil
	case "aosp15":
		return aosp15(host), nil
	default:
		return spec{}, fmt.Errorf("未登记的裁剪组件：%s", comp)
	}
}
