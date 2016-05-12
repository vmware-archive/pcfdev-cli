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
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []*Interface{}, err
	}

	interfaces = make([]*Interface, len(addrs))
	for i, addr := range addrs {
		interfaces[i] = &Interface{
			IP: strings.Split(addr.String(), "/")[0],
		}
	}
	return interfaces, nil
}
