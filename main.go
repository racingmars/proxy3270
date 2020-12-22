package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/racingmars/go3270"
)

var config []ServerConfig
var screen go3270.Screen
var rules go3270.Rules

func main() {
	var err error
	debug := flag.Bool("debug", false, "sets log level to debug")
	port := flag.Int("port", 3270, "port number to listen on")
	configFile := flag.String("config", "config.json", "configuration file path")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "02 15:04:05"})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
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
			go handle(conn)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info().Msg("Interrupt signal received: quitting.")
	return
}

func handle(conn net.Conn) {
	defer conn.Close()
	if err := go3270.NegotiateTelnet(conn); err != nil {
		fmt.Fprintf(os.Stderr,
			"err: couldn't negotiate connection from %s\n", conn.RemoteAddr())
		return
	}

	response, err := go3270.HandleScreen(screen, rules, nil,
		[]go3270.AID{go3270.AIDEnter}, []go3270.AID{go3270.AIDPF3},
		"errormsg", 2, 33, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"err: couldn't handle screen for %s:\n  %v\n",
			conn.RemoteAddr(), err)
		return
	}
	if response.AID == go3270.AIDPF3 {
		return
	}
	selection, _ := strconv.Atoi(response.Values["input"])
	selection = selection - 1
	remote := fmt.Sprintf("%s:%d", config[selection].Host, config[selection].Port)
	if err = proxy(conn, remote); err != nil {
		log.Error().Err(err).Msgf("Error proxying to %s", remote)
		return
	}
}

func buildScreen(config []ServerConfig) (go3270.Screen, go3270.Rules) {
	screen := make(go3270.Screen, 0)
	rules := make(go3270.Rules)

	screen = append(screen, go3270.Field{Row: 0, Col: 27, Intense: true, Content: "3270 Proxy Application"})
	screen = append(screen, go3270.Field{Row: 2, Col: 2, Content: "Select service to connect to:"})
	screen = append(screen, go3270.Field{Row: 2, Col: 32, Name: "input", Highlighting: go3270.Underscore, Write: true})
	screen = append(screen, go3270.Field{Row: 2, Col: 35}) // Field "stop" character
	screen = append(screen, go3270.Field{Row: 20, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"})
	screen = append(screen, go3270.Field{Row: 22, Col: 0, Content: "PF3 Exit"})

	for i := range config {
		var rowBase = 4
		var colBase = 2

		// Wrap to the second column
		if i > 12 {
			rowBase = -9
			colBase = 41
		}

		screen = append(screen, go3270.Field{Row: rowBase + i, Col: colBase, Content: fmt.Sprintf("%2d", i+1), Intense: true})
		screen = append(screen, go3270.Field{Row: rowBase + i, Col: colBase + 3, Content: config[i].Name})
	}

	v := func(input string) bool {
		if val, err := strconv.Atoi(input); err != nil {
			return false
		} else if val < 1 || val > len(config) {
			return false
		}
		return true
	}
	rules["input"] = go3270.FieldRules{Validator: v}

	return screen, rules
}