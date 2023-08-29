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
		trim := size-len([]byte(appendText))-3
		if trim > 0 && trim < len([]byte(tweetText)) {
			tweetText = string([]byte(tweetText)[:trim]) + "..."
		}
	}
	return tweetText + appendText
}
