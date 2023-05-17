package ses

import (
	"bytes"
	"github.com/jchavannes/jgo/jerr"
	"mime/multipart"
	"net/textproto"
	"strings"
)

type Email struct {
	UserId  uint
	From    string
	To      []string
	Subject string
	Body    string
}

func (e Email) GetMessageData() (string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	h := make(textproto.MIMEHeader)
	h.Set("From", e.From)
	h.Set("To", strings.Join(e.To, ","))
	h.Set("Subject", e.Subject)
	h.Set("Content-Language", "en-US")
	h.Set("Content-Type", "multipart/mixed; boundary=\""+writer.Boundary()+"\"")
	h.Set("MIME-Version", "1.0")
	if _, err := writer.CreatePart(h); err != nil {
		return "", jerr.Get("error writing e header", err)
	}
	h = make(textproto.MIMEHeader)
	h.Set("Content-Transfer-Encoding", "7bit")
	h.Set("Content-Type", "text/html; charset=us-ascii")
	part, err := writer.CreatePart(h)
	if err != nil {
		return "", jerr.Get("error writing e body part", err)
	}
	if _, err = part.Write([]byte(e.Body)); err != nil {
		return "", jerr.Get("error writing e body", err)
	}
	if err = writer.Close(); err != nil {
		return "", jerr.Get("error closing writer", err)
	}
	msgData := buf.String()
	if strings.Count(msgData, "\n") < 2 {
		return "", jerr.New("invalid e-mail content")
	}
	msgData = strings.SplitN(msgData, "\n", 2)[1]
	return msgData, nil
}
