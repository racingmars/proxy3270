package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// func main() {
// 	ln, err := net.Listen("tcp", ":5433")
// 	if err != nil {
// 		panic(err)
// 	}
// 	for {
// 		conn, err := ln.Accept()
// 		if err != nil {
// 			// handle error
// 		}
// 		go handleConnection(conn)
// 	}
// }

func handleConnection(client net.Conn) {
	defer client.Close()
	err := proxy(client, "vmesa.lab.mattwilson.org:23")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Connection done")
}

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
		fmt.Println("got serverdone")
		clientend <- true
	case <-clientdone:
		fmt.Println("got clientdone")
		serverend <- true
	}

	wg.Wait()

	return nil
}

func readAndFeed(name string, in, out net.Conn, wg *sync.WaitGroup, end, done chan bool) {
	defer func() {
		close(done)
		in.SetReadDeadline(time.Time{})
		fmt.Println("ending readAndFeed(): " + name)
		wg.Done()
	}()
	buffer := make([]byte, 1024)
	finish := false
	for !finish {
		select {
		case <-end:
			fmt.Printf("%s got end signal\n", name)
			finish = true
		default:
			in.SetReadDeadline(time.Now().Add(time.Second / 2))
			n, err := in.Read(buffer)
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			} else if err != nil {
				fmt.Printf("read error: %v\n", err)
				return
			}
			//fmt.Printf("read %d bytes\n", n)
			if _, err := out.Write(buffer[:n]); err != nil {
				fmt.Printf("write error: %v", err)
				return
			}
		}
	}
}
