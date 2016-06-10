package system

import "regexp"

func (s *System) PhysicalCores() (int, error) {
	cpuinfo, err := s.FS.Read("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}

	return s.uniq(cpuinfo, `physical id.*`) * s.uniq(cpuinfo, `core id.*`), nil
}

func (s *System) uniq(data []byte, regex string) int {
	compiledRegex := regexp.MustCompile(regex)
	matches := compiledRegex.FindAllStringSubmatch(string(data), -1)
	encounters := map[string]bool{}
	num := 0
	for _, match := range matches {
		if encounters[match[0]] != true {
			encounters[match[0]] = true
			num++
		}
	}
	return num
}
