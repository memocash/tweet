package main

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
)

func main() {
	tweets := tweets.Load()
	for _,tweet := range tweets {
		if err := graph.BasicQuery(tweet.Text); err != nil {
			jerr.Get("error running basic query", err).Fatal()
		}
	}
}
