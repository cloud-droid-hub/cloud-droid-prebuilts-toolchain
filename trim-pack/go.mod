module github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/trim-pack

go 1.26.0

require (
	github.com/klauspost/compress v1.19.0
	github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit v0.15.33
	github.com/ulikunitz/xz v0.5.16
	golang.org/x/sys v0.47.0
)

replace github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit => ../../../pkg/devicekit
