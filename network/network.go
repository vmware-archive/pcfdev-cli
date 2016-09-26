package network

import (
	"net"
	"strings"
)

type Network struct{}

type Interface struct {
	HardwareAddress string
	IP              string
	Name            string
	Exists          bool
}

func (n *Network) HasIPCollision(ip string) (bool, error) {
	interfaces, err := n.Interfaces()
	if err != nil {
		return false, err
	}

	for _, networkInterface := range interfaces {
		if networkInterface.IP == ip {
			return true, nil
		}
	}
	return false, nil
}

func (n *Network) Interfaces() (interfaces []*Interface, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	interfaces = make([]*Interface, 0)
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			addrString := strings.Split(addr.String(), "/")[0]

			if IsIPV4(addrString) {
				interfaces = append(interfaces, &Interface{
					IP:              addrString,
					HardwareAddress: iface.HardwareAddr.String(),
					Exists:          true,
				})
			}
		}
	}

	return interfaces, nil
}

func IsIPV4(ip string) bool {
	ip4 := net.ParseIP(ip).To4()
	if ip4 == nil {
		return false
	}

	return len(ip4) == net.IPv4len
}
