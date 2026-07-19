package version

import _ "embed"

//go:embed VERSION
var Version string

//go:embed BUILD_TIME
var BuildTime string
