package version

import (
	"fmt"
	"runtime"
)

var (
	version   = "NotSet"
	builddate = "NotSet"
)

func Show() {
	fmt.Printf("Version:    \t%s\n", version)
	fmt.Printf("Build date: \t%s\n", builddate)
	fmt.Printf("Go Compiler:\t%s\n", runtime.Version())
}
