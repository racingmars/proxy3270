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
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/racingmars/go3270"
)

var config *Config
var screen go3270.Screen
var rules go3270.Rules

func main() {
	var err error

	debug := flag.Bool("debug", false, "sets log level to debug")
	debug3270 := flag.Bool("debug3270", false, "enables debugging in the go3270 library")
	trace := flag.Bool("trace", false, "sets log level to trace")
	port := flag.Int("port", 3270, "port number to listen on")
	configFile := flag.String("config", "config.json", "configuration file path")
	telnetTimeout := flag.Int("telnetTimeout", 1, "length of time to wait for telnet command response from clients when un-negotiating the 3270 session")
	logFile := flag.String("log", "", "log file name to enable logging to a file")
	flag.Parse()

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout /*, TimeFormat: "02 15:04:05"*/}
	log.Logger = log.Output(consoleWriter)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *trace {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if *logFile != "" {
		f, err := os.OpenFile(*logFile,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
		if err != nil {
			log.Error().Err(err).Msg("Couldn't open log file")
			return
		}
		defer f.Close()
		multi := zerolog.MultiLevelWriter(consoleWriter, f)
		log.Logger = zerolog.New(multi).With().Timestamp().Logger()
		log.Info().Msgf("Logging to file %s", *logFile)
	}

	if *debug3270 {
		go3270.Debug = os.Stderr
	}

	if *telnetTimeout < 1 {
		log.Error().Err(err).Msg("telnetTimeout must be positive")
		return
	}

	config, err = loadConfig(*configFile)
	if err != nil {
		log.Error().Err(err).Msg("Couldn't load config file")
		return
	}

	err = validateConfig(config)
	if err != nil {
		log.Error().Err(err).Msg("Config error")
		return
	}

	screen, rules = buildScreen(config)

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Error().Err(err).Msg("Couldn't start listener")
		return
	}
	log.Info().Msgf("LISTENING ON PORT %d FOR CONNECTIONS", *port)
	log.Info().Msg("Press Ctrl-C to end server.")

	// Run the accept loop in a goroutine so we can wait on the quit signal
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Error().Err(err).Msg("Couldn't accept connection")
			}
			log.Info().Msgf("New connection from %s", conn.RemoteAddr())
			go handle(conn, *telnetTimeout)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info().Msg("Interrupt signal received: quitting.")
	return
}

func handle(conn net.Conn, timeout int) {
	defer conn.Close()
	if err := go3270.NegotiateTelnet(conn); err != nil {
		log.Error().Err(err).Msgf("couldn't negotiate connection from %s", conn.RemoteAddr())
		return
	}

	response, err := go3270.HandleScreen(screen, rules, nil,
		[]go3270.AID{go3270.AIDEnter}, []go3270.AID{go3270.AIDPF3},
		"errormsg", 2, 33, conn)
	if err != nil {
		log.Error().Err(err).Msgf("err: couldn't handle screen for %s", conn.RemoteAddr())
		return
	}
	if response.AID == go3270.AIDPF3 {
		return
	}
	selection, _ := strconv.Atoi(response.Values["input"])
	selection = selection - 1
	remote := fmt.Sprintf("%s:%d", config.Servers[selection].Host,
		config.Servers[selection].Port)

	if err = go3270.UnNegotiateTelnet(conn, time.Second*time.Duration(timeout)); err != nil {
		log.Error().Err(err).Msgf("Couldn't unnegotiate client")
		return
	}

	log.Info().Msgf("Connecting client %s to server %s", conn.RemoteAddr(), remote)
	if err = proxy(conn, remote); err != nil {
		log.Error().Err(err).Msgf("Error proxying to %s", remote)
	}
	log.Info().Msgf("Client %s session ended", conn.RemoteAddr())
}

func buildScreen(config *Config) (go3270.Screen, go3270.Rules) {
	screen := make(go3270.Screen, 0)
	rules := make(go3270.Rules)

	titleStart := 39 - (len(config.Title) / 2)
	screen = append(screen, go3270.Field{Row: 0, Col: titleStart, Intense: true, Content: config.Title})
	screen = append(screen, go3270.Field{Row: 2, Col: 2, Content: "Select service to connect to:"})
	screen = append(screen, go3270.Field{Row: 2, Col: 32, Name: "input", Highlighting: go3270.Underscore, Write: true})
	screen = append(screen, go3270.Field{Row: 2, Col: 35}) // Field "stop" character
	screen = append(screen, go3270.Field{Row: 20, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"})
	screen = append(screen, go3270.Field{Row: 22, Col: 0, Content: "PF3 Exit"})

	for i := range config.Servers {
		var rowBase = 4
		var colBase = 2

		// Wrap to the second column
		if i > 12 {
			rowBase = -9
			colBase = 41
		}

		screen = append(screen, go3270.Field{Row: rowBase + i, Col: colBase, Content: fmt.Sprintf("%2d", i+1), Intense: true})
		screen = append(screen, go3270.Field{Row: rowBase + i, Col: colBase + 3, Content: config.Servers[i].Name})
	}

	v := func(input string) bool {
		if val, err := strconv.Atoi(input); err != nil {
			return false
		} else if val < 1 || val > len(config.Servers) {
			return false
		}
		return true
	}
	rules["input"] = go3270.FieldRules{Validator: v}

	return screen, rules
}
