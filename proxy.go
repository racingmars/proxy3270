package main

import (
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func proxy(client net.Conn, target string) error {
	server, err := net.DialTimeout("tcp", target, 15*time.Second)
	if err != nil {
		return err
	}
	defer server.Close()

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
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			} else if err == io.EOF {
				log.Debug().Msg("connection closed")
				return
			} else if err != nil {
				log.Error().Err(err).Msg("read error")
				return
			}
			if _, err := out.Write(buffer[:n]); err != nil {
				log.Error().Err(err).Msg("write error")
				return
			}
		}
	}
}
