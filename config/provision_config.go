package config

type ProvisionConfig struct {
	Domain   string `json:"domain"`
	IP       string `json:"ip"`
	Services string `json:"services"`
}
