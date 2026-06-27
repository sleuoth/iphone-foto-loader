package main

import (
	"os"

	"github.com/sebastianleuoth/iphone-foto-loader/internal/cli"
)

func main() {
	app := cli.ParseFlags()
	os.Exit(app.Run())
}
