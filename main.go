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
	"github.com/racingmars/goproxy/db"
)

func main() {
	debug := flag.Bool("debug", false, "sets log level to debug")
	port := flag.Int("port", 3270, "port number to listen on")
	//resetAdmin := flag.Bool("resetAdmin", false, "reset the admin password to a new random value")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "02 15:04:05"})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	db, err := db.New()
	if err != nil {
		log.Error().Err(err).Msg("Couldn't open database")
		return
	}
	defer db.Close()

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

	screen := go3270.Screen{
		{Row: 0, Col: 27, Intense: true, Content: "3270 Proxy Application"},
		{Row: 2, Col: 2, Content: "Select service to connect to:"},
		{Row: 2, Col: 32, Name: "input", Highlighting: go3270.Underscore, Write: true},
		{Row: 2, Col: 34}, // Field "stop" character
		{Row: 20, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"}, // a blank field for error messages
		{Row: 22, Col: 0, Content: "PF3 Exit"},
	}

	screenRules := go3270.Rules{
		"input": {Validator: go3270.IsInteger},
	}

	response, err := go3270.HandleScreen(screen, screenRules, nil,
		[]go3270.AID{go3270.AIDEnter}, []go3270.AID{go3270.AIDPF3},
		"errormsg", 2, 32, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"err: couldn't handle screen for %s:\n  %v\n",
			conn.RemoteAddr(), err)
		return
	}
	if response.AID == go3270.AIDPF3 {
		return
	}
	if err = proxy(conn, "mw03.lab.mattwilson.org:23"); err != nil {
		fmt.Fprintf(os.Stderr,
			"err: for %s:\n  %v\n", conn.RemoteAddr(), err)
		return
	}
}
