package main

import (
	"math/rand"
	"time"
)

var version = "dev" // has to be set by ldflags

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
}
