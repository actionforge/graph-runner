package core

import (
	"fmt"
	"runtime/debug"
)

var (
	// To set version number, build with:
	// $ go build -ldflags "-X actionforge/graph-runner/core.Version=v1.2.3"
	Version string
)

func GetBuildSettings() (map[string]string, bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, false
	}

	settings := map[string]string{}

	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}
	return settings, true

}

func GetAppVersion() string {
	if Version != "" {
		return Version
	} else {
		return "development build"
	}
}

func GetFulllVersionInfo() string {

	bi, ok := GetBuildSettings()
	if !ok {
		return "invalid build info"
	}

	if Version == "" {
		Version = "unknown"
	}

	// if git status returned no changes
	modified := ""
	if bi["vcs.modified"] == "true" {
		modified = ", workdir modified"
	}

	revision := bi["vcs.revision"]
	if len(revision) > 8 {
		revision = revision[:8]
	}

	return fmt.Sprintf("%s (%s %s, %s, %s%s)", Version, bi["GOOS"], bi["GOARCH"], bi["vcs.time"], revision, modified)
}
