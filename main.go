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
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/racingmars/go3270"
)

type userSession struct {
	page       int
	totalPages int
}

const errFieldName = "errmessage"
const pageSize = 12

var config *Config

func main() {
	var err error

	debug := flag.Bool("debug", false, "sets log level to debug")
	debug3270 := flag.Bool("debug3270", false, "enables debugging in the go3270 library")
	trace := flag.Bool("trace", false, "sets log level to trace")
	port := flag.Int("port", 3270, "port number to listen on")
	tlsport := flag.Int("tlsport", 4270, "port number to listen for TLS connection")
	pubkey := flag.String("pubkey", "pubkey.pem", "public certificate and bundle (PEM)")
	privkey := flag.String("privkey", "privkey.pem", "private key (PEM)")
	tlsenable := flag.Bool("tlsenable", false, "Enable TLS listener?")
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

	var tlsln net.Listener
	if *tlsenable {
		cert, err := tls.LoadX509KeyPair(*pubkey, *privkey)
		if err != nil {
			log.Error().Err(err).Msg("Couldn't load X.509 certificate")
			return
		}

		certConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		tlsln, err = tls.Listen("tcp", ":"+strconv.Itoa(*tlsport), certConfig)
		if err != nil {
			log.Error().Err(err).Msg("Couldn't start TLS listener")
			return
		}
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Error().Err(err).Msg("Couldn't start unencrypted listener")
		return
	}
	log.Info().Msgf("LISTENING ON PORT %d FOR CONNECTIONS", *port)
	if *tlsenable {
		log.Info().Msgf("LISTENING ON PORT %d FOR TLS CONNECTIONS", *tlsport)
	}
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

	// Run the accept loop in a goroutine so we can wait on the quit signal
	if *tlsenable {
		go func() {
			for {
				conn, err := tlsln.Accept()
				if err != nil {
					log.Error().Err(err).Msg("Couldn't accept TLS connection")
				}
				log.Info().Msgf("New TLS connection from %s", conn.RemoteAddr())
				go handle(conn, *telnetTimeout)
			}
		}()
	}

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

	session := &userSession{}
	session.totalPages = len(config.Servers) / pageSize
	if session.totalPages*pageSize < len(config.Servers) {
		session.totalPages++
	}

	var response go3270.Response
	var errmsg string
	for {
		screen, rules := buildScreen(config, session, "")
		var err error
		response, err = go3270.HandleScreen(screen, rules,
			map[string]string{errFieldName: errmsg},
			[]go3270.AID{go3270.AIDEnter}, []go3270.AID{go3270.AIDPF3,
				go3270.AIDPF7, go3270.AIDPF8},
			errFieldName, 2, 33, conn)
		if err != nil {
			log.Error().Err(err).Msgf("err: couldn't handle screen for %s", conn.RemoteAddr())
			return
		}
		errmsg = ""
		switch response.AID {
		case go3270.AIDPF3:
			return
		case go3270.AIDPF7:
			// page up
			if session.page <= 0 {
				errmsg = "Already on the first page"
				continue
			}
			session.page--
			continue
		case go3270.AIDPF8:
			// page down
			if session.page >= session.totalPages-1 {
				errmsg = "Already on the last page"
				continue
			}
			session.page++
			continue
		case go3270.AIDEnter:
			break
		default:
			log.Error().Msgf("Somehow we got an unexpected key from HandleScreen()")
			continue
		}

		// otherwise... continue with rest of function
		break
	}
	selection, _ := strconv.Atoi(response.Values["input"])
	selection = selection - 1
	remote := fmt.Sprintf("%s:%d", config.Servers[selection].Host,
		config.Servers[selection].Port)

	if err := go3270.UnNegotiateTelnet(conn, time.Second*time.Duration(timeout)); err != nil {
		log.Error().Err(err).Msgf("Couldn't unnegotiate client")
		return
	}

	log.Info().Msgf("Connecting client %s to server %s", conn.RemoteAddr(), remote)
	if err := proxy(conn, config.Servers[selection].Host,
		config.Servers[selection].Port, config.Servers[selection].UseTLS,
		config.Servers[selection].IgnoreCertValidation); err != nil {
		log.Error().Err(err).Msgf("Error proxying to %s", remote)
	}
	log.Info().Msgf("Client %s session ended", conn.RemoteAddr())
}

func buildScreen(config *Config, session *userSession, errmsg string) (go3270.Screen, go3270.Rules) {
	screen := make(go3270.Screen, 0)
	rules := make(go3270.Rules)

	discline1, discline2 := wrapDisclaimer(config.Disclaimer, 79)
	titleStart := 39 - (len(config.Title) / 2)
	screen = append(screen, go3270.Field{Row: 0, Col: titleStart, Intense: true, Content: config.Title})
	screen = append(screen, go3270.Field{Row: 2, Col: 2, Content: "Select service to connect to:"})
	screen = append(screen, go3270.Field{Row: 2, Col: 32, Name: "input", Highlighting: go3270.Underscore, Write: true})
	screen = append(screen, go3270.Field{Row: 2, Col: 36}) // Field "stop" character
	screen = append(screen, go3270.Field{Row: 17, Col: 0, Intense: true, Color: go3270.Red, Name: errFieldName})
	screen = append(screen, go3270.Field{Row: 19, Col: 0, Color: go3270.Red, Content: discline1})
	screen = append(screen, go3270.Field{Row: 20, Col: 0, Color: go3270.Red, Content: discline2})
	screen = append(screen, go3270.Field{Row: 22, Col: 0, Content: "PF3 Exit"})

	if session.page > 0 {
		screen = append(screen, go3270.Field{Row: 22, Col: 13, Content: "PF7 PgUp"})
	}
	if session.page < session.totalPages-1 {
		screen = append(screen, go3270.Field{Row: 22, Col: 25, Content: "PF8 PgDn"})
	}

	for i := range config.Servers[session.page*pageSize:] {
		if i > 11 {
			break
		}

		const rowBase = 4

		screen = append(screen, go3270.Field{Row: rowBase + i, Col: 2, Content: fmt.Sprintf("%3d", session.page*pageSize+i+1), Intense: true})
		screen = append(screen, go3270.Field{Row: rowBase + i, Col: 6, Content: config.Servers[session.page*pageSize+i].Name})
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

// wrapDisclaimer will split the input string into line1 with no more than
// linelength characters, and the remaining text in line2.
// CAVEATS: line2 may extend longer than the linelength. This function is
// currently only intended for use with displaying the disclaimer text in the
// buildScreen() function. If additional word wrap uses cases arise, this
// function can be modified to return a slice of strings with unlimited
// wrapped lines.
func wrapDisclaimer(disclaimer string, linelength int) (line1, line2 string) {
	disclaimer = strings.TrimSpace(disclaimer)

	// String already fits entirely on first line
	if len(disclaimer) <= linelength {
		return disclaimer, ""
	}

	// Look for word boundary to wrap on
	for i := linelength - 1; i >= 0; i-- {
		if disclaimer[i] == ' ' {
			line1 = disclaimer[0:i]
			line2 = disclaimer[i+1:]
			return line1, line2
		}
	}

	// Handle case where no word boundary was found
	line1 = disclaimer[0:linelength]
	line2 = disclaimer[linelength:]
	return line1, line2
}
