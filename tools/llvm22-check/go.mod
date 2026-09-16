module github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/tools/llvm22-check

go 1.26.0

require (
	github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain v0.0.0
	github.com/ulikunitz/xz v0.5.16
)

require golang.org/x/sys v0.48.0 // indirect

replace github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain => ../../darwin-arm64
