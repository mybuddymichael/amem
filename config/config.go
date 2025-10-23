package config

// Config represents configuration at either ~/.config/amem/config.json or .amem/config.json
type Config struct {
	DBPath string `json:"db_path"`
}
