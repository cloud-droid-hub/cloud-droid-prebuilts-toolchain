// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package catalog

// jdkArchive保留完整构建运行时，输入和裁剪过程由主仓jdk-trim登记。
func jdkArchive() Archive {
	return Archive{
		Name: "JDK", Version: "21.0.12.1+1", Dir: "jdk21", Format: TarGzip,
		Prefix: "jdk-21.0.12.1+1",
		SHA256: "95879f0608ebd1dea4229a1ff25b756a789e35a05fe431d7cfd4799e0dcd0c26",
		Parts: []Part{{
			Path: "payload/jdk21/OpenJDK21U-jdk-cdh-21.0.12.1-1-linux-arm64.tar.gz.part-000",
			Size: 75957224, SHA256: "95879f0608ebd1dea4229a1ff25b756a789e35a05fe431d7cfd4799e0dcd0c26",
		}},
		Files: []FileCheck{{
			Path: "cdh-jdk-trim.txt", Size: 152,
			SHA256: "7ba062a1b83a0ef6f09f3b44cb732bd6135cab417ee9617a3d407399b0245ed0",
			Want: []string{"profile=jdk21-build-v1", "version=21.0.12.1+1", "removed=jmods,man,lib/src.zip"},
		}},
		Probes: []Probe{
			{Path: "bin/java", Args: []string{"-version"}, Want: []string{"Temurin-21.0.12.1+1"}, Machine: ELFARM64, Env: jdkEnv()},
			{Path: "bin/javac", Args: []string{"-version"}, Want: []string{"javac 21.0.12.1"}, Machine: ELFARM64, Env: jdkEnv()},
			{Path: "bin/jar", Args: []string{"--version"}, Want: []string{"jar 21.0.12.1"}, Machine: ELFARM64, Env: jdkEnv()},
			{Path: "bin/keytool", Args: []string{"-J-version"}, Want: []string{"Temurin-21.0.12.1+1"}, Machine: ELFARM64, Env: jdkEnv()},
		},
	}
}

func jdkEnv() []string {
	return []string{"JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS="}
}
