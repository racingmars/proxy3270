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
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func proxy(client net.Conn, targetHost string, targetPort uint, useTLS, ignoreCertErrors bool) error {
	server, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetHost,
		targetPort), 15*time.Second)
	if err != nil {
		return err
	}
	defer server.Close()

	if useTLS {
		tlsConfig := &tls.Config{
			ServerName:         targetHost,
			InsecureSkipVerify: ignoreCertErrors,
		}
		server = tls.Client(server, tlsConfig)
	}

	clientdone := make(chan bool)
	clientend := make(chan bool)
	serverdone := make(chan bool)
	serverend := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(2)
	go readAndFeed("client", client, server, &wg, clientend, clientdone)
	go readAndFeed("server", server, client, &wg, serverend, serverdone)

	select {
	case <-serverdone:
		log.Debug().Msg("got serverdone")
		clientend <- true
	case <-clientdone:
		log.Debug().Msg("got clientdone")
		serverend <- true
	}

	wg.Wait()

	return nil
}

func readAndFeed(name string, in, out net.Conn, wg *sync.WaitGroup, end, done chan bool) {
	defer func() {
		close(done)
		in.SetReadDeadline(time.Time{})
		log.Debug().Msgf("ending readAndFeed(): %s", name)
		wg.Done()
	}()
	log.Debug().Msgf("starting readAndFeed(): %s", name)
	buffer := make([]byte, 1024)
	finish := false
	for !finish {
		select {
		case <-end:
			log.Debug().Msgf("%s got end signal", name)
			finish = true
		default:
			in.SetReadDeadline(time.Now().Add(time.Second / 2))
			n, err := in.Read(buffer)
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				continue
			} else if err == io.EOF {
				log.Debug().Msgf("connection closed: %s", name)
				return
			} else if err != nil {
				log.Error().Err(err).Msgf("read error: %s", name)
				return
			}
			log.Trace().Hex("data", buffer[:n]).Msgf("%s read", name)
			if _, err := out.Write(buffer[:n]); err != nil {
				log.Error().Err(err).Msgf("write error: %s", name)
				return
			}
		}
	}
}
