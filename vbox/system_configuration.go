package vbox

type SystemConfiguration struct {}

func (s *SystemConfiguration) NetworkConfiguration(ip string) (string, error) {
	return "", nil
}

func (s *SystemConfiguration) EnvironmentConfiguration(httpProxy, httpsProxy, noProxy, ip, domain string) (string, error) {
	return "", nil
}