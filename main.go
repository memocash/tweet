package main

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
)

func main() {
	tweets.Load()
	if err := graph.BasicQuery(); err != nil {
		jerr.Get("error running basic query", err).Fatal()
	}
}
