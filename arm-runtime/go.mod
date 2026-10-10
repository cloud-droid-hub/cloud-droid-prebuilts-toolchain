module github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/arm-runtime

go 1.26.0

require (
	github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit v0.8.0
	github.com/ulikunitz/xz v0.5.15
	golang.org/x/sys v0.47.0
)

replace github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit => ../../../pkg/devicekit
