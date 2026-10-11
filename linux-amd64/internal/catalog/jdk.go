// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package catalog

// jdkArchive保留完整构建运行时，输入和裁剪过程由主仓jdk-trim登记。
func jdkArchive() Archive {
	return Archive{
		Name: "JDK", Version: "21.0.12.1+1", Dir: "jdk21", Format: TarGzip,
		Prefix: "jdk-21.0.12.1+1",
		SHA256: "2b76cb04fd9782d5ff461f2e38270a688a8ce038ceeb31961657aa7b6f59fb60",
		Parts: []Part{{
			Path: "payload/jdk21/OpenJDK21U-jdk-cdh-21.0.12.1-1-linux-amd64.tar.gz.part-000",
			Size: 76693983, SHA256: "2b76cb04fd9782d5ff461f2e38270a688a8ce038ceeb31961657aa7b6f59fb60",
		}},
		Files: []FileCheck{{
			Path: "cdh-jdk-trim.txt", Size: 152,
			SHA256: "eb99bb7507e0262e13b058238d28a13e278e2199fa0d0522d662c61beeacd573",
			Want: []string{"profile=jdk21-build-v1", "version=21.0.12.1+1", "removed=jmods,man,lib/src.zip"},
		}},
		Probes: []Probe{
			{Path: "bin/java", Args: []string{"-version"}, Want: []string{"Temurin-21.0.12.1+1"}, Machine: ELFAMD64, Env: jdkEnv()},
			{Path: "bin/javac", Args: []string{"-version"}, Want: []string{"javac 21.0.12.1"}, Machine: ELFAMD64, Env: jdkEnv()},
			{Path: "bin/jar", Args: []string{"--version"}, Want: []string{"jar 21.0.12.1"}, Machine: ELFAMD64, Env: jdkEnv()},
			{Path: "bin/keytool", Args: []string{"-J-version"}, Want: []string{"Temurin-21.0.12.1+1"}, Machine: ELFAMD64, Env: jdkEnv()},
		},
	}
}

func jdkEnv() []string {
	return []string{"JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS="}
}
