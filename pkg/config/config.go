package config

type Config struct {
	LogLevel       string
	FritzBoxURL    string
	Username       string
	Password       string
	MetricsAddress string
	LogPath        string
	LokiURL        string
}

func NewConfig() *Config {
	return &Config{}
}
