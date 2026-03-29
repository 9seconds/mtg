// Echo server — runs on Amsterdam, simulates Telegram DC.
// Simply echoes back everything received on each connection.
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
)

var activeConns atomic.Int64

func main() {
	addr := flag.String("addr", "0.0.0.0:19999", "listen address")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("echo server listening on %s\n", *addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept: %v\n", err)
			continue
		}
		activeConns.Add(1)
		go func(c net.Conn) {
			defer c.Close()
			defer activeConns.Add(-1)
			io.Copy(c, c) //nolint: errcheck
		}(conn)
	}
}
