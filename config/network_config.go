package config

import "github.com/pivotal-cf/pcfdev-cli/network"

type NetworkConfig struct {
	VMIP      string
	VMDomain  string
	Interface *network.Interface
}
