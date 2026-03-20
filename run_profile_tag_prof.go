//go:build prof

package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint: gosec
	"os"
)

const DefaultProfPort = "6000"

func runProfile() {
	port := os.Getenv("MTG_PROF_PORT")
	if port == "" {
		port = DefaultProfPort
	}

	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		panic(err)
	}

	go http.Serve(listener, nil)
}
