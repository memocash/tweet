package save_test

import (
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/tweet/tweets/save"
	"log"
	"testing"
)

func TestText_Gen(t *testing.T) {
	text := save.Text{
		Text:     "@spacesudoer @TeslaSynopsis @teslaownersSV @EvaFoxU @dvorahfr @Kristennetten @imPenny2x @JaneidyEve @SirineAti @kerrikgray That BBC reporter obviously had no idea what he was saying, but someone told him to make those false claims. \n\nThe question is who.",
		Link:     "https://twitter.com/elonmusk/status/1693470151331426631",
		FlagLink: true,
	}
	tweetText := text.Gen(memo.OldMaxPostSize)
	log.Printf("tweetText: %s\n", tweetText)
	if len([]byte(tweetText)) > memo.OldMaxPostSize {
		t.Errorf("tweet text is too long")
	}
}
