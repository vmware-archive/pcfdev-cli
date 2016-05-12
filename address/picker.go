package address

import (
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/network"
)

//go:generate mockgen -package mocks -destination mocks/network.go github.com/pivotal-cf/pcfdev-cli/address Network
type Network interface {
	Interfaces() (interfaces []*network.Interface, err error)
}

//go:generate mockgen -package mocks -destination mocks/ping.go github.com/pivotal-cf/pcfdev-cli/address Pinger
type Pinger interface {
	TryIP(ip string) (responds bool, err error)
}

type Picker struct {
	Pinger  Pinger
	Network Network
}

func (p *Picker) SelectAvailableNetworkInterface(candidates []*network.Interface) (selectedInterface *network.Interface, exists bool, err error) {
	allInterfaces, err := p.Network.Interfaces()
	if err != nil {
		return nil, false, err
	}

	for _, subnetIP := range AllowedSubnets {
		if vboxAddr := p.addrInSet(subnetIP, candidates); vboxAddr != nil {
			vmIP, err := IPForSubnet(subnetIP)
			if err != nil {
				return nil, false, err
			}

			responds, err := p.Pinger.TryIP(vmIP)
			if err != nil {
				return nil, false, err
			}

			if !responds {
				return vboxAddr, true, nil
			}
		}

		if p.addrInSet(subnetIP, allInterfaces) == nil {
			return &network.Interface{IP: subnetIP}, false, nil
		}
	}

	return nil, false, fmt.Errorf("all allowed network interfaces are currently taken")
}

func (p *Picker) addrInSet(ip string, set []*network.Interface) (addr *network.Interface) {
	for _, addr := range set {
		if addr.IP == ip {
			return addr
		}
	}

	return nil
}
