package bot

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"io"
	"net/http"
	"net/url"
)

func InfoBalance(address wallet.Addr) error {
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
	fmt.Println(string(body))
	return nil
}
func InfoProfile(sender wallet.Addr, twittername string) (error){
	port := config.GetConfig().InfoServerPort
	if port == 0 {
		return fmt.Errorf("error info request port is 0")
	}
	resp, err := http.PostForm(fmt.Sprintf("http://localhost:%d/profile", port), url.Values{
		"sender": {sender.String()},
		"twittername": {twittername},
	})
	if err != nil {
		return fmt.Errorf("error info request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading info response body: %w", err)
	}
	fmt.Println(string(body))
	return nil
}
