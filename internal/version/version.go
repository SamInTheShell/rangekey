package version

import (
	"fmt"
	"runtime"
)

// Build information set by ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

// Info holds version information
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	Commit    string `json:"commit"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		Commit:    Commit,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("RangeDB %s (built %s, commit %s, %s, %s)",
		i.Version, i.BuildTime, i.Commit, i.GoVersion, i.Platform)
}
