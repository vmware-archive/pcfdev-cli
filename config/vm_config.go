package config

type VMConfig struct {
	Name     string
	DiskName string
	Domain   string
	IP       string
	Memory   uint64
	CPUs     int
	SSHPort  string
}
