// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package version_test

import (
	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(workpath.Run(m)) }
