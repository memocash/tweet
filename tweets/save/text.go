package save

import (
	"fmt"
)

type Text struct {
	Text     string
	Link     string
	Date     string
	Media    string
	FlagLink bool
	FlagDate bool
}

func (t Text) Gen(size int) string {
	tweetText := t.Text
	var appendText string
	if t.Media != "" {
		appendText += fmt.Sprintf("\n%s", t.Media)
	}
	if t.FlagLink {
		appendText += fmt.Sprintf("\n%s", t.Link)
	}
	if t.FlagDate {
		appendText += fmt.Sprintf("\n%s", t.Date)
	}
	if len([]byte(tweetText))+len([]byte(appendText)) > size {
		if len([]byte(appendText)) > size/2 {
			appendText = appendText[:size/2] + "..."
		}
		tweetText = string([]byte(tweetText)[:size-len([]byte(appendText))-3]) + "..."
	}
	return tweetText + appendText
}
