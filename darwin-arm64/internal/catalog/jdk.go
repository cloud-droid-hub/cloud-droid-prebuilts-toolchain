// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package catalog

// jdkArchive保留完整构建运行时，输入和裁剪过程由主仓jdk-trim登记。
func jdkArchive() Archive {
	return Archive{
		Name: "JDK", Version: "21.0.12.1+1", Dir: "jdk21", Format: TarGzip,
		Prefix: "jdk-21.0.12.1+1",
		SHA256: "405ef8055a7c282cfab0ba454106c09b167cc22775672c0a3854a99681b28dce",
		Parts: []Part{{
			Path: "payload/jdk21/OpenJDK21U-jdk-cdh-21.0.12.1-1-darwin-arm64.tar.gz.part-000",
			Size: 72392094, SHA256: "405ef8055a7c282cfab0ba454106c09b167cc22775672c0a3854a99681b28dce",
		}},
		Files: []FileCheck{{
			Path: "cdh-jdk-trim.txt", Size: 152,
			SHA256: "889a4249a2b3918276e188d0d5906ee2dacc33b5f3fd8500102eaf8328e45fdb",
			Want: []string{"profile=jdk21-build-v1", "version=21.0.12.1+1", "removed=jmods,man,lib/src.zip"},
		}},
		Probes: []Probe{
			{Path: "bin/java", Args: []string{"-version"}, Want: []string{"Temurin-21.0.12.1+1"}, Machine: MachARM64, Env: jdkEnv()},
			{Path: "bin/javac", Args: []string{"-version"}, Want: []string{"javac 21.0.12.1"}, Machine: MachARM64, Env: jdkEnv()},
			{Path: "bin/jar", Args: []string{"--version"}, Want: []string{"jar 21.0.12.1"}, Machine: MachARM64, Env: jdkEnv()},
			{Path: "bin/keytool", Args: []string{"-J-version"}, Want: []string{"Temurin-21.0.12.1+1"}, Machine: MachARM64, Env: jdkEnv()},
		},
	}
}

func jdkEnv() []string {
	return []string{"JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS="}
}
