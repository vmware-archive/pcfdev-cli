package vm

import (
	"fmt"
	"regexp"
)

type SensitiveInformationScrubber struct {
}

type sensitiveInformationRegex struct {
	label string
	regex *regexp.Regexp
}

var (
	sensitiveInformationRegexes = []sensitiveInformationRegex{
		sensitiveInformationRegex{
			label: "certificate",
			regex: regexp.MustCompile(`-----BEGIN CERTIFICATE-----.*-----END CERTIFICATE-----`),
		},
		sensitiveInformationRegex{
			label: "private-key",
			regex: regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----.*-----END RSA PRIVATE KEY-----`),
		},
		sensitiveInformationRegex{
			label: "email",
			regex: regexp.MustCompile(`(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))`),
		},
		sensitiveInformationRegex{
			label: "ip-address",
			regex: regexp.MustCompile(`((1?[0-9][0-9]?|2[0-4][0-9]|25[0-5])[.,]){3}(1?[0-9][0-9]?|2[0-4][0-9]|25[0-5])`),
		},
		sensitiveInformationRegex{
			label: "uri",
			regex: regexp.MustCompile(`([^\s]+:\/\/)(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}(\.[a-z]{2,6})?\b([-a-zA-Z0-9@:%_\+.~#?&\/\/=]*)`),
		},
		sensitiveInformationRegex{
			label: "secret",
			regex: regexp.MustCompile(`(\"|')*[A-Za-z0-9_-]*([sS]ecret|[pP]rivate[-_]?[Kk]ey|[Pp]assword|[sS]alt|SECRET|PRIVATE[-_]?KEY|PASSWORD|SALT)[\"']*\s*(=|:|\s|:=|=>)\s*[\"']*[A-Za-z0-9.$+=&\\_\\\\-]{12,}(\"|'|\s)`),
		},
	}
)

func (s *SensitiveInformationScrubber) Scrub(information string) string {
	scrubbedInformation := information
	for _, sregex := range sensitiveInformationRegexes {
		scrubbedInformation = sregex.regex.ReplaceAllString(
			scrubbedInformation,
			fmt.Sprintf("<redacted %s>", sregex.label),
		)
	}
	return scrubbedInformation
}
