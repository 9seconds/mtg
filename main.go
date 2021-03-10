package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

var version = "dev" // has to be set by ldflags

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	f, _ := os.Open("example.config.toml")

	fmt.Println(parseConfig(f))
}
