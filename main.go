package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/dialers"
)

var version = "dev" // has to be set by ldflags

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	f, _ := os.Open("example.config.toml")

	fmt.Println(parseRawConfig(f))

	bd, _ := dialers.NewDefaultBaseDialer(0, 0)
	d, _ := dialers.MakeDialer(bd, "9.9.9.9", 0)

	r, err := d.HTTP.Get("https://ifconfig.co")

	fmt.Println(err)
	body, _ := ioutil.ReadAll(r.Body)

	fmt.Println(string(body))
}
