/*
 * Copyright 2020 by Matthew R. Wilson <mwilson@mattwilson.org>
 *
 * This file is part of proxy3270.
 *
 * proxy3270 is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * proxy3270 is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with proxy3270. If not, see <https://www.gnu.org/licenses/>.
 */

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
