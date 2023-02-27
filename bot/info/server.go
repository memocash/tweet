package info

import (
	"fmt"
	"github.com/memocash/tweet/config"
	"net/http"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (l *Server) Listen() error {
	cfg := config.GetConfig()
	if cfg.InfoServerPort == 0 {
		return fmt.Errorf("port is 0 for info server")
	}
	mux := http.NewServeMux()
	for _, handler := range []Handler{
		handlerBalance,
		handlerProfile,
	} {
		mux.HandleFunc(handler.Pattern, handler.Handler)
	}
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.InfoServerPort), mux)
	return fmt.Errorf("error listening for info api: %w", err)
}
