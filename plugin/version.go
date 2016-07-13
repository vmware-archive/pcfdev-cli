package plugin

import "fmt"

type Version struct {
	BuildVersion    string
	BuildSHA        string
	OVABuildVersion string
}

func (v *Version) getFullVersion() string {
	return fmt.Sprintf("%s (CLI: %s, OVA: %s)", v.BuildVersion, v.BuildSHA, v.OVABuildVersion)
}
