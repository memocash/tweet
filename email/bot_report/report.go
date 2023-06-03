package bot_report

import (
	"bytes"
	"fmt"
	"github.com/hasura/go-graphql-client"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/email"
	tweetWallet "github.com/memocash/tweet/wallet"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"html/template"
	"os"
)

type Report struct {
	Bots []*Bot
}

func New(bots []*Bot) *Report {
	return &Report{
		Bots: bots,
	}
}

func (r *Report) Run(scraper *twitterscraper.Scraper) error {
	graphQlUrl := config.GetGraphQlUrl()
	graphQlClient := graphql.NewClient(graphQlUrl, nil)
	client := lib.NewClient(graphQlUrl, &tweetWallet.Database{})
	for _, bot := range r.Bots {
		if err := bot.SetInfo(client, graphQlClient, scraper); err != nil {
			return fmt.Errorf("error setting info for bot %s; %w", bot.Address, err)
		}
	}
	templateText, err := os.ReadFile(email.GetTemplatePath(email.BotReportTemplate))
	if err != nil {
		return fmt.Errorf("error reading balance report template; %w", err)
	}
	tmpl, err := template.New("html").Parse(string(templateText))
	if err != nil {
		return fmt.Errorf("error parsing balance report template; %w", err)
	}
	var w = new(bytes.Buffer)
	if err := tmpl.Execute(w, r); err != nil {
		return fmt.Errorf("error executing balance report template; %w", err)
	}
	sender := email.NewSender()
	awsConfig := config.GetAwsSesCredentials()
	if err := sender.Send(email.Email{
		From:    awsConfig.FromEmail,
		To:      awsConfig.ToEmails,
		Subject: "Daily Twitter Bot Report",
		Body:    w.String(),
	}); err != nil {
		return fmt.Errorf("error sending ses balance report email; %w", err)
	}
	return nil
}
