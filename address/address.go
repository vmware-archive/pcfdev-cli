package address

import (
	"fmt"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/network"
)

var allowedSubnets = []string{
	"192.168.11.1",
	"192.168.22.1",
	"192.168.33.1",
	"192.168.44.1",
	"192.168.55.1",
	"192.168.66.1",
	"192.168.77.1",
	"192.168.88.1",
	"192.168.99.1",
}

var AllowedAddresses = map[string]string{
	"192.168.11.11": "local.pcfdev.io",
	"192.168.22.11": "local2.pcfdev.io",
	"192.168.33.11": "local3.pcfdev.io",
	"192.168.44.11": "local4.pcfdev.io",
	"192.168.55.11": "local5.pcfdev.io",
	"192.168.66.11": "local6.pcfdev.io",
	"192.168.77.11": "local7.pcfdev.io",
	"192.168.88.11": "local8.pcfdev.io",
	"192.168.99.11": "local9.pcfdev.io",
}

func DomainForIP(ip string) string {
	domain, ok := AllowedAddresses[ip]
	if ok {
		return domain
	} else {
		return fmt.Sprintf("%s.xip.io", ip)
	}
}

func IPForSubnet(subnet string) string {
	return subnet + "1"
}

func SubnetForIP(ip string) (string, error) {
	if !network.IsIPV4(ip) {
		return "", fmt.Errorf("%s is not a supported IP address", ip)
	}

	splitIP := strings.Split(ip, ".")
	splitIP[3] = "1"

	return strings.Join(splitIP, "."), nil
}

func SubnetForDomain(requestedDomain string) (string, error) {
	var ip string

	for subnet, domain := range AllowedAddresses {
		if requestedDomain == domain {
			ip = subnet
		}
	}

	if ip == "" {
		return "", fmt.Errorf("%s is not one of the allowed PCF Dev domains", requestedDomain)
	}

	return ip[0 : len(ip)-1], nil
}

func IsDomainAllowed(domain string) bool {
	for _, allowedDomain := range AllowedAddresses {
		if allowedDomain == domain {
			return true
		}
	}
	return false
}
