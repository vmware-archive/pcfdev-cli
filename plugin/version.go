package plugin

import "fmt"

type Version struct {
	BuildVersion    string
	BuildSHA        string
	OVABuildVersion string
}

func (v *Version) getFullVersion() string {
	return fmt.Sprintf("PCF Dev version %s (CLI: %s, OVA: %s)", v.BuildVersion, v.BuildSHA, v.OVABuildVersion)
}
