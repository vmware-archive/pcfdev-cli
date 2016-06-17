package network

import (
	"net"
	"strings"
)

type Network struct{}

type Interface struct {
	Name string
	IP   string
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

		if len(addrs) > 0 {
			interfaces = append(interfaces, &Interface{
				IP:   strings.Split(addrs[0].String(), "/")[0],
				Name: iface.Name,
			})
		}
	}

	return interfaces, nil
}
