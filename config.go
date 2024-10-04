/*
 * Copyright 2020-2021 by Matthew R. Wilson <mwilson@mattwilson.org>
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
	"regexp"
	"strings"
)

const MaxServers = 999
const MaxNameLength = 65
const MaxAppTitleLength = 79
const MaxDisclaimerLineLength = 79

const defaultTitle = "3270 Proxy Application"

type Config struct {
	Title      string         `json:"title"`
	Disclaimer string         `json:"disclaimer"`
	Servers    []ServerConfig `json:"servers"`
}

type ServerConfig struct {
	Name                 string `json:"name"`
	Host                 string `json:"host"`
	Port                 uint   `json:"port"`
	UseTLS               bool   `json:"secure"`
	IgnoreCertValidation bool   `json:"ignoreCertValidation"`
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	// Ensure a title is set
	config.Title = strings.TrimSpace(config.Title)
	if config.Title == "" {
		config.Title = defaultTitle
	}

	// Trim the disclaimer, but blank is permitted
	config.Disclaimer = strings.TrimSpace(config.Disclaimer)

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Title) > MaxAppTitleLength {
		return fmt.Errorf("Application title is too long: max %d characters",
			MaxAppTitleLength)
	}

	if !validateEbcdicString(config.Title) {
		return fmt.Errorf("Application title contains illegal character")
	}

	if !validateEbcdicString(config.Disclaimer) {
		return fmt.Errorf("Disclaimer text contains illegal character")
	}
	if _, line2 := wrapDisclaimer(
		config.Disclaimer, MaxDisclaimerLineLength); len(line2) > MaxDisclaimerLineLength {
		return fmt.Errorf("The word-wrapped disclaimer text exceeds two lines")
	}

	if len(config.Servers) > MaxServers {
		return fmt.Errorf("Too many server configurations (%d): max %d",
			len(config.Servers), MaxServers)
	}
	for i := range config.Servers {
		if len(strings.TrimSpace(config.Servers[i].Name)) == 0 {
			return fmt.Errorf("Server index %d has a blank name", i)
		}

		if len(config.Servers[i].Name) > MaxNameLength {
			return fmt.Errorf("Server `%s` name too long: max %d characters",
				config.Servers[i].Name, MaxNameLength)
		}

		if len(strings.TrimSpace(config.Servers[i].Host)) == 0 {
			return fmt.Errorf("Host missing on server `%s`", config.Servers[i].Name)
		}

		if config.Servers[i].Port == 0 {
			return fmt.Errorf("Port missing on server `%s`", config.Servers[i].Name)
		}

		if config.Servers[i].Port > 65535 {
			return fmt.Errorf("Port %d invalid on server `%s`",
				config.Servers[i].Port, config.Servers[i].Name)
		}
	}

	return nil
}

// validateEbcdicString will return true if the input string contains only
// allowed characters, false otherwise.
var validEdcdicStringRegexp = regexp.MustCompile("^[a-zA-Z0-9 ,.;:!|\\\\/<>@#$%^&*(){}\\-_+=~`\"']*$")

func validateEbcdicString(input string) bool {
	return validEdcdicStringRegexp.MatchString(input)
}
