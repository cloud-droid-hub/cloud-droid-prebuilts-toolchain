// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

type debFile struct {
	Size int64
	SHA  string
	Ver  string
}

type debPkg struct {
	Name string
	Ver  string
	Pool string
	Lic  string
	ARM  debFile
	AMD  debFile
}

var llvmDebs = []debPkg{
	{"clang-22", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{126148, "d5dc4e2a1bb3eff5384bd5aa14142eb2794b771a662e4aefeed61d865204b762", ""}, debFile{128878, "88e7c3b717fa0573773316d41df8a81487bc1c4be79173a35a73abd352cbd8b3", ""}},
	{"libbrotli1", "1.2.0-3build1", "main/b/brotli", "libbrotli", debFile{345086, "14b47a091b6e04fa18baf42f456a25844f67904f15a900f07de651842fa74e62", ""}, debFile{342830, "858242571fc2585c1f8eb83ae01d020f0f4a92679d0de8fe31146f6687bdd84a", ""}},
	{"libclang-common-22-dev", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{771430, "858e604b6f1bba510a374b51bbdbb361da61fa17ccee6370f0968c6ee8077af7", ""}, debFile{771430, "8c390c08cd370b4d3b61ef788622be4f4886ee175d4888620a14cd67d40be964", ""}},
	{"libclang-cpp22", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{13879976, "32134565753e7b4d02d4962770051093ff7b0723d233c83ea98fe507418b9859", ""}, debFile{14194946, "77800834c5544ab887397c93856d371a03e3e899e3eee585621b48cf07e86f50", ""}},
	{"libclang1-22", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{8279002, "efa6fc4f2decccbe360d54db1f7502682c1b9281cc1162831a327d4d235ec3ef", ""}, debFile{8544562, "6e975db23d9b567384fac8d5e6f78feed2f90be9a5fd38db2db15973ceb1c549", ""}},
	{"libcurl4t64", "8.18.0-1ubuntu2.5", "main/c/curl", "libcurl", debFile{414052, "eebf0bcf6b3c3fc1d44c122d95061bfa29077de6a8508d829fc8cd1aa26983e9", ""}, debFile{426986, "ea42d65a232664d310f96da2303038cb4aa4bc8e69df0b33c505e798f644ea79", ""}},
	{"libedit2", "3.1-20251016-1", "main/libe/libedit", "libedit", debFile{96286, "a184af2ab98ee85ed2b7616a995b93bd9d0b65c7552b4f799ee27ada2b52707f", ""}, debFile{97586, "3afbc8ddf835049dd4d88275b9d13680e4c4cdd59e191cbd63424152cc08e00e", "3.1-20230828-1build1"}},
	{"libffi8", "3.5.2-4", "main/libf/libffi", "libffi", debFile{24428, "cfab73343f076b79df0f4a9d5430319fdb6153760d3e8c2964c448b49edad084", ""}, debFile{26348, "4a7423124bbd459c0aa38b7389ec7802fa6899654767a95fc27de70967fedd48", ""}},
	{"libgnutls30t64", "3.8.12-2ubuntu1.1", "main/g/gnutls28", "libgnutls", debFile{956940, "0460daa7a06ecd4f9869e21189a768a295d27fa69ddd6826d1f3e06e10a4db99", ""}, debFile{1002902, "b8944756260c5ea6b7fac019745f931f905aa7a4df0763676b1b25c911b692e2", "3.8.3-1.1ubuntu3.6"}},
	{"libgssapi-krb5-2", "1.22.1-2ubuntu4.1", "main/k/krb5", "libgssapi", debFile{143790, "a4e69632d9785ede429297e285b6c63adb84202adc8ce434b89cb2f70471d48d", ""}, debFile{146908, "d156dd940bff6dec8b6880cd0da4d515398d03d7b2e8b3163a935e701f80aa27", ""}},
	{"libhogweed6t64", "3.10.2-1", "main/n/nettle", "libhogweed", debFile{199520, "d6b297d01eb4a37a5f5277f8285221567c54537c20d730fc5f5c4bde72ffd2ce", ""}, debFile{200902, "1c8c808ab24ff3018ac886e3f311edee0f2d4d97d12d3908be52343e89807046", ""}},
	{"libidn2-0", "2.3.8-4build1", "main/libi/libidn2", "libidn2", debFile{68330, "8c0daa227474b787790f62fd15c9ad05890e22a66c2eaeabee12b4ccdb5102c6", ""}, debFile{67564, "92bdac0f443a648859d06dee3defb9c96864fd76df6cfe617ca70bb261344767", ""}},
	{"libk5crypto3", "1.22.1-2ubuntu4.1", "main/k/krb5", "libk5crypto", debFile{82216, "87857bbe222b0342bae1c5315da48d8de59b07f0c62a5aa4d119c7927627759c", ""}, debFile{81998, "8e99ef104a6bf49bea708747fdf4c28eb4fcebd6dba9e55ecfa919211eb5ab75", ""}},
	{"libkeyutils1", "1.6.3-6ubuntu3", "main/k/keyutils", "libkeyutils", debFile{10772, "0c0b4da672ca8e3e9ce972e3482a06f4cff5b4ebe774d2d04b2ad50d07bb566b", ""}, debFile{10634, "ea7d3cc642ffe2459c35a85c894f80e92499b9eba13bdcb225e0aa751c0970be", ""}},
	{"libkrb5-3", "1.22.1-2ubuntu4.1", "main/k/krb5", "libkrb5", debFile{356990, "3db98b0be2b704c38cc29c88dfa45dd5e8bfe1645a3150b284c355d8dca01c05", ""}, debFile{360714, "54612a2019863c44d80218f40d40670c3684108f4b2a760aae644ab68457e239", ""}},
	{"libkrb5support0", "1.22.1-2ubuntu4.1", "main/k/krb5", "libkrb5support", debFile{33522, "bc2dd02729cac287b0f6d5304b71fdb80d53591a68f6566462b996f780722e3a", ""}, debFile{33546, "127d735efece139dfb7109ac3709da7eb8923bef090c20a10dbaf4c4fa45001d", ""}},
	{"libldap2", "2.6.10+dfsg-1ubuntu5", "main/o/openldap", "libldap", debFile{200394, "da7c275bd1c86fc25221d94c81bd044d09be1596ef24cb59d9379f6819d031ec", ""}, debFile{204284, "8d3a361e8655a0ff4a390243059949fa6f22d42d0e266f58b57c124eb21d2993", ""}},
	{"libllvm22", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{30809840, "1a5ded78f49e811d1c6df2c025f25d827921a58e21c123bac90186654fc3e356", ""}, debFile{32105218, "c7724b2a524e3bbeadd997beeafccf44349d4b6f704f6af1bf76036223f5b342", ""}},
	{"libnettle8t64", "3.10.2-1", "main/n/nettle", "libnettle", debFile{189840, "922ad03de441c26ca6e2a437545b7afdec5cfb6aeb123ee51b6b6a4e5b5a0975", ""}, debFile{185378, "83a2a924c11759a2ed6e6f98f456d28e1b026974ce9e6e7ef12c9ac43b2d543e", ""}},
	{"libnghttp2-14", "1.68.0-2ubuntu0.2", "main/n/nghttp2", "libnghttp2", debFile{70696, "c82ed5194057474817bef69df8ce6774c835377c4d9eae404913d3a9b8d26536", ""}, debFile{71038, "9c92ca4435998b7981cf81f957177b71079563b022c282a4b8b7829113a5d08e", ""}},
	{"libp11-kit0", "0.26.2-2", "main/p/p11-kit", "libp11-kit", debFile{306238, "0776f4755a37442e4e4c7f2e0c8fbe4bea021ad9652fe5702d0cc003cf9f5683", ""}, debFile{312508, "e70d3422e24b267293215134458f340944bf10e92b27147594c69a7dd2823089", ""}},
	{"libpfm4", "4.13.0+git106-g3e4031b-1ubuntu0.1", "universe/libp/libpfm4", "libpfm", debFile{81498, "e2801def09699b1894a7552ec34943e34cc82f7b958e5375a2f9c972a9aced9c", ""}, debFile{426836, "34b1d3923be6ea7b4fde3efaf59165cc8f0ad1999f653051b13d7d1a6623f932", ""}},
	{"libpsl5t64", "0.21.2-1.1build2", "main/libp/libpsl", "libpsl", debFile{59744, "1ba472a3fffd896653cbabb67813fa481e76e0ab0b459e3e69f294ed56a1c075", ""}, debFile{59510, "1ae62705714f660ecdfd70273fce5b226b6cb7b9fe53776531799093edc99774", ""}},
	{"librtmp1", "2.4+20151223.gitfa8646d.1-3", "main/r/rtmpdump", "librtmp", debFile{58744, "a86935b5ca9454a51360b8995944ee09c89804088eaf91ef9f18a4ba64b7d540", ""}, debFile{58970, "77cbd0ddf7e7487079ec300ff7751322de5e7d67f0f46749591b78ff1442cbe8", ""}},
	{"libsasl2-2", "2.1.28+dfsg1-9ubuntu3", "main/c/cyrus-sasl2", "libsasl2", debFile{53730, "58ae0fc34067c27aa5ee50c6ca3823919057f1a4fb092478f9ff47a3dfe36496", ""}, debFile{53984, "11b9b2ce6702d86dd3dea60e08bd15255d7fd9545431a9e555eec47770404de0", ""}},
	{"libssh2-1t64", "1.11.1-1ubuntu0.26.04.4", "main/libs/libssh2", "libssh2", debFile{134296, "4d6ea3c94ae7190320879de7ab24e698c9f28e6d90d59ff06f1a4796ddd1f70d", ""}, debFile{138284, "bb3e334dc885d124d38cd4350f1cb4606c03ce58e9f53dc51a84c2c8d11c84e1", ""}},
	{"libssl3t64", "3.5.5-1ubuntu3.5", "main/o/openssl", "libssl", debFile{}, debFile{2362598, "f62cc56364e417e4d2478c5d6a1d07af0f40625e587dd8dceb96b23b1fa76e07", ""}},
	{"libtasn1-6", "4.21.0-2", "main/libt/libtasn1-6", "libtasn1", debFile{44426, "63d38a74ad15cc88fec21f408cdabf857d5c5a9d83887a21915c24b428fa6576", ""}, debFile{45272, "59c74951324ead0ff77ab945d561feb1b21bc189b793600b8e2b7cf4d39b1ee5", ""}},
	{"libunistring5", "1.3-2build1", "main/libu/libunistring", "libunistring", debFile{600874, "1a73d3d7690e26f926d7af025b44d6d57c2dd1e0abb35c0d5b1b11f4b9b7f4da", ""}, debFile{610496, "88c0163f1acb384abde07641b7e13fcaea36e0a3b88fc57a653779fc429fc46a", ""}},
	{"libxml2-16", "2.15.2+dfsg-0.1ubuntu0.1", "main/libx/libxml2", "libxml2", debFile{605442, "43e1d2d2ac42a33ee8871d54b8ca4a531b9170b027d31f5fe71b91c8f9e20315", ""}, debFile{606124, "14211c45bb75f9543e6053207533370047873ade7e4c70f7285400f1df6c898f", ""}},
	{"libz3-4", "4.13.3-1build1", "universe/z/z3", "libz3", debFile{6272974, "ea50bb2de8942c41c3736eeb3eb65d620dd9c4c98542b7145fb5b32003906514", ""}, debFile{6887480, "40390573227a6965210a477554d6c9aa0ab51edcc0daad51db9352a6b9bfdeb4", ""}},
	{"lld-22", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{1467950, "158e2d5595afd6aba28dc9912efd1b6ad8771cdc4e0fc9d2b691d26fe3246b3e", ""}, debFile{1549408, "f1a950ffa01dc0e3bbc00d71d9a0d96b160f7d2b78141638b9ff50f1f9762633", ""}},
	{"llvm-22-linker-tools", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{1197556, "282c0fb1aec3d77713108ea94e7a6a7ddcc6924a691f7d183ccb783c33feb791", ""}, debFile{1284178, "860341378699a29a91fe2e0fe3bb23aaaabba260ff0ccd80ba2bae4e0d1ee597", ""}},
	{"llvm-22-runtime", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "", debFile{576004, "fc5aae2eca1e4c5969e41d48b0f531999683d1bb228a92ae0563ff67eddabfed", ""}, debFile{598128, "7281a2820449b2cd2eb272962a98ffbb1891e51e2c06ef6209b7239dccace850", ""}},
	{"llvm-22", "1:22.1.2-1ubuntu1", "universe/l/llvm-toolchain-22", "llvm", debFile{19901480, "9dbf788533259b3e262708873462e5653661d145d0fd440322288bfe302b42d6", ""}, debFile{20689684, "1691c74ab075178ac3ab97fd4d4a61dce980a3bf2c7d825452faf938e69d49a7", ""}},
}
