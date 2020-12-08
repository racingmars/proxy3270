package main

import (
	"fmt"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":5433")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
		}
		go handleConnection(conn)
	}
}

func handleConnection(client net.Conn) {
	defer client.Close()
	server, err := net.Dial("tcp", "mw03.lab.mattwilson.org:23")
	if err != nil {
		panic(err)
	}
	defer server.Close()

	clientIn := make(chan byte)
	serverIn := make(chan byte)
	clientDone := make(chan bool)
	serverDone := make(chan bool)

	go readAndFeed(client, clientIn, clientDone)
	go readAndFeed(server, serverIn, serverDone)

	var clientByte, serverByte byte
mainloop:
	for {
		fmt.Println("Begin select")
		select {
		case clientByte = <-clientIn:
			fmt.Printf("client byte: %02x\n", clientByte)
			if _, err := server.Write([]byte{clientByte}); err != nil {
				panic(err)
			}
		case serverByte = <-serverIn:
			fmt.Printf("server byte: %02x\n", serverByte)
			if _, err := client.Write([]byte{serverByte}); err != nil {
				panic(err)
			}
		case <-serverDone:
			fmt.Println("server done signal")
			break mainloop
		case <-clientDone:
			fmt.Println("client done signal")
			break mainloop
		}
		fmt.Println("End select")
	}
}

func readAndFeed(conn net.Conn, data chan byte, done chan bool) {
	buffer := make([]byte, 256)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("read error: %v\n", err)
			done <- true
			return
		}
		fmt.Printf("read %d bytes\n", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%d: %02x\n", i, buffer[i])
			data <- buffer[i]
		}
	}
}
