package plugin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudfoundry/cli/plugin"
)

type Version struct {
	BuildVersion    string
	BuildSHA        string
	OVABuildVersion string
}

func (v *Version) getFullVersion() string {
	return fmt.Sprintf("PCF Dev version %s (CLI: %s, OVA: %s)", v.BuildVersion, v.BuildSHA, v.OVABuildVersion)
}

func (v *Version) getVersionForCLIMetadata() plugin.VersionType {
	versionParts := strings.SplitN(v.BuildVersion, ".", 3)
	majorVersion, errMajor := strconv.Atoi(versionParts[0])
	minorVersion, errMinor := strconv.Atoi(versionParts[1])
	buildVersion, errBuild := strconv.Atoi(versionParts[2])

	if errMajor != nil || errMinor != nil || errBuild != nil {
		panic("This CLI was built against an invalid build number.")
	}

	return plugin.VersionType{
		Major: majorVersion,
		Minor: minorVersion,
		Build: buildVersion,
	}
}
