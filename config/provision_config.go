package config

type ProvisionConfig struct {
	Domain     string   `json:"domain"`
	IP         string   `json:"ip"`
	Services   string   `json:"services"`
	Registries []string `json:"registries"`
	Provider   string   `json:"provider"`
}
