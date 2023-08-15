package email

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
)

type Sender struct {
	MessageId string
}

func (s *Sender) Send(email Email) error {
	msgData, err := email.GetMessageData()
	if err != nil {
		return jerr.Get("error getting message data for email send", err)
	}
	awsConfig := config.GetAwsSesCredentials()
	creds := credentials.NewStaticCredentials(awsConfig.Key, awsConfig.Secret, "")
	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(awsConfig.Region),
	})
	if err != nil {
		return jerr.Get("error getting aws session", err)
	}
	svc := ses.New(sess)
	emailList := []*string{}
	for _, e := range email.To {
		emailList = append(emailList, aws.String(e))
	}
	input := &ses.SendRawEmailInput{
		Destinations: emailList,
		Source:       aws.String(email.From),
		RawMessage: &ses.RawMessage{
			Data: []byte(msgData),
		},
	}
	result, err := svc.SendRawEmail(input)
	if err != nil {
		return jerr.Get("error sending raw email", err)
	}
	if result.MessageId != nil {
		s.MessageId = *result.MessageId
	}
	return nil
}

func NewSender() *Sender {
	return &Sender{}
}
