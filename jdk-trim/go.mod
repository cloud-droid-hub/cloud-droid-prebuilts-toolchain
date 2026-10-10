module github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/jdk-trim

go 1.26.0

require (
	github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit v0.15.33
	golang.org/x/sys v0.47.0
)

replace github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit => ../../../pkg/devicekit
