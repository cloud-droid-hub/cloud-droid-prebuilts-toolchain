// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import "strings"

var plans = []plan{
	{Host: "darwin-arm64", Prefix: outPrefix + "/Contents/Home",
		SrcSHA: "3623232f33a9c3baadf304480b2535f9a3cba8a58d42ecbb438ba267315d9998",
		Source: "OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz",
		Stem:   "OpenJDK21U-jdk-cdh-21.0.12.1-1-darwin-arm64", Sizes: []int64{94371840, 94371840, 11329724}},
	{Host: "linux-arm64", Prefix: outPrefix,
		SrcSHA: "23e37e026f12f3e706f18938ff611db3032d075b09d0879a25d06718c773e223",
		Source: "OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.12.1_1.tar.gz",
		Stem:   "OpenJDK21U-jdk-cdh-21.0.12.1-1-linux-arm64", Sizes: []int64{94371840, 94371840, 16897495}},
	{Host: "linux-amd64", Prefix: outPrefix,
		SrcSHA: "ce79869e1307ed8ee1e2baa86a412b1eb5b75d10a01006d788a6f968bcfaee94",
		Source: "OpenJDK21U-jdk_x64_linux_hotspot_21.0.12.1_1.tar.gz",
		Stem:   "OpenJDK21U-jdk-cdh-21.0.12.1-1-linux-amd64", Sizes: []int64{94371840, 94371840, 18729667}},
}

// 删除策略只针对构建时无需读取的模块副本、上游源码和手册。
func dropped(name string) bool {
	return name == "jmods" || strings.HasPrefix(name, "jmods/") ||
		name == "man" || strings.HasPrefix(name, "man/") || name == "lib/src.zip"
}
