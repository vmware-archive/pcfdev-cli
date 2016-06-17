package network

import (
	"net"
	"strings"
)

type Network struct{}

type Interface struct {
	HardwareAddress string
	Name            string
	IP              string
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

			if n.isIPV4(net.ParseIP(addrString)) {
				interfaces = append(interfaces, &Interface{
					IP:              addrString,
					HardwareAddress: iface.HardwareAddr.String(),
				})
			}
		}
	}

	return interfaces, nil
}

func (n *Network) isIPV4(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	return len(ip4) == net.IPv4len
}
