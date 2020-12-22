package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const MaxServers = 26
const MaxNameLength = 30

type ServerConfig struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port uint   `json:"port"`
}

func loadConfig(path string) ([]ServerConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(f)
	var config []ServerConfig
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func validateConfig(config []ServerConfig) error {
	if len(config) > MaxServers {
		return fmt.Errorf("Too many server configurations (%d): max %d",
			len(config), MaxServers)
	}

	for i := range config {
		if len(strings.TrimSpace(config[i].Name)) == 0 {
			return fmt.Errorf("Server index %d has a blank name", i)
		}

		if len(config[i].Name) > MaxNameLength {
			return fmt.Errorf("Server `%s` name too long: max %d characters",
				config[i].Name, MaxNameLength)
		}

		if len(strings.TrimSpace(config[i].Host)) == 0 {
			return fmt.Errorf("Host missing on server `%s`", config[i].Name)
		}

		if config[i].Port == 0 {
			return fmt.Errorf("Port missing on server `%s`", config[i].Name)
		}

		if config[i].Port > 65535 {
			return fmt.Errorf("Port %d invalid on server `%s`",
				config[i].Port, config[i].Name)
		}
	}

	return nil
}
