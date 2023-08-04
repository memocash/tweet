package info

import (
	"fmt"
	"github.com/memocash/tweet/config"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"net/http"
)

const (
	BalanceEndpointPath = "/balance"
	ProfileEndpointPath = "/profile"
	ReportEndpointPath  = "/report"
)

type Server struct {
	Scraper   *twitterscraper.Scraper
	Mux       *http.ServeMux
	ErrorChan chan error
}

func NewServer(scraper *twitterscraper.Scraper) *Server {
	return &Server{
		Scraper:   scraper,
		Mux:       http.NewServeMux(),
		ErrorChan: make(chan error),
	}
}

func (l *Server) Listen() error {
	cfg := config.GetConfig()
	if cfg.InfoServerPort == 0 {
		return fmt.Errorf("port is 0 for info server")
	}
	l.Mux.HandleFunc(BalanceEndpointPath, l.balanceHandler)
	l.Mux.HandleFunc(ProfileEndpointPath, l.profileHandler)
	l.Mux.HandleFunc(ReportEndpointPath, l.reportHandler)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.InfoServerPort), l.Mux)
		l.ErrorChan <- fmt.Errorf("error admin server listener; %w", err)
	}()
	return <-l.ErrorChan
}
