package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port: port,
	}
}

func (c Config) Address() string {
	return ":" + c.Port
}
