package address

import "fmt"

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

var allowedAddresses = map[string]string{
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

func DomainForIP(ip string) (string, error) {
	domain, ok := allowedAddresses[ip]
	if !ok {
		return "", fmt.Errorf("%s is not one of the allowed pcfdev ips", ip)
	}

	return domain, nil
}

func SubnetForIP(ip string) (string, error) {
	_, ok := allowedAddresses[ip]
	if !ok {
		return "", fmt.Errorf("%s is not one of the allowed pcfdev ips", ip)
	}

	return ip[0 : len(ip)-1], nil
}

func IPForSubnet(subnet string) (string, error) {
	ip := subnet + "1"
	_, ok := allowedAddresses[ip]
	if !ok {
		return "", fmt.Errorf("%s is not one of the allowed pcfdev subnets", subnet)
	}
	return ip, nil
}
