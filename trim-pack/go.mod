module github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/trim-pack

go 1.26.0

require (
	github.com/cloud-droid-hub/cloud-droid-build/work v0.1.16
	github.com/klauspost/compress v1.19.0
	github.com/ulikunitz/xz v0.5.16
	golang.org/x/sys v0.47.0
)

replace github.com/cloud-droid-hub/cloud-droid-build/work => ../../build/work
