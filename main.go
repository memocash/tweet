package main

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		jerr.Get("fatal error executing command", err).Fatal()
	}
}
