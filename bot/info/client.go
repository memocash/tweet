package info

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func Balance(address wallet.Addr) error {
	port := config.GetConfig().InfoServerPort
	if port == 0 {
		return fmt.Errorf("error info request port is 0")
	}
	resp, err := http.PostForm(fmt.Sprintf("http://localhost:%d/balance", port), url.Values{
		"address": {address.String()},
	})
	if err != nil {
		return fmt.Errorf("error info request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading info response body: %w", err)
	}
	log.Println(string(body))
	return nil
}
func Profile(sender wallet.Addr, userId int64) error {
	port := config.GetConfig().InfoServerPort
	if port == 0 {
		return fmt.Errorf("error info request port is 0")
	}
	resp, err := http.PostForm(fmt.Sprintf("http://localhost:%d/profile", port), url.Values{
		"sender": {sender.String()},
		"userId": {strconv.FormatInt(userId, 10)},
	})
	if err != nil {
		return fmt.Errorf("error info request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading info response body: %w", err)
	}
	log.Println(string(body))
	return nil
}

func Report() error {
	port := config.GetConfig().InfoServerPort
	if port == 0 {
		return fmt.Errorf("error info request port is 0")
	}
	resp, err := http.PostForm(fmt.Sprintf("http://localhost:%d/report", port), url.Values{})
	if err != nil {
		return fmt.Errorf("error info request: %w", err)
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading info response body: %w", err)
	}
	return nil
}
