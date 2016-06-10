package address

import (
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/network"
)

//go:generate mockgen -package mocks -destination mocks/network.go github.com/pivotal-cf/pcfdev-cli/address Network
type Network interface {
	Interfaces() (interfaces []*network.Interface, err error)
}

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/address Driver
type Driver interface {
	IsInterfaceInUse(interfaceName string) (inUse bool, err error)
}

type Picker struct {
	Network Network
	Driver  Driver
}

func (p *Picker) SelectAvailableNetworkInterface(candidates []*network.Interface) (selectedInterface *network.Interface, exists bool, err error) {
	allInterfaces, err := p.Network.Interfaces()
	if err != nil {
		return nil, false, err
	}

	for _, subnetIP := range allowedSubnets {
		if vboxAddr := p.addrInSet(subnetIP, candidates); vboxAddr != nil {
			if p.isDuplicateInterface(vboxAddr, allInterfaces) {
				continue
			}

			inUse, err := p.Driver.IsInterfaceInUse(vboxAddr.Name)
			if err != nil {
				return nil, false, err
			}

			if !inUse {
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

func (p *Picker) isDuplicateInterface(networkInterface *network.Interface, set []*network.Interface) bool {
	count := 0
	for _, netInterface := range set {
		if networkInterface.IP == netInterface.IP {
			count += 1
		}
	}

	return (count > 1)
}
