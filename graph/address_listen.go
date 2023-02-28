package graph

import (
	"github.com/hasura/go-graphql-client"
	"github.com/hasura/go-graphql-client/pkg/jsonutil"
	"github.com/jchavannes/jgo/jerr"
)

func AddressListen(addresses []string, fn func(Tx) error, errorChan chan error) error {
	client := graphql.NewSubscriptionClient(ServerUrlWs)
	defer client.Close()
	var subscription = new(Subscription)
	client.OnError(func(sc *graphql.SubscriptionClient, err error) error {
		errorChan <- jerr.Get("error in client subscription for address listen", err)
		return nil
	})
	var newTxHandler = func(dataValue []byte, errValue error) error {
		if errValue != nil {
			return jerr.Get("error in subscription for address listen new tx handler", errValue)
		}
		data := Subscription{}
		err := jsonutil.UnmarshalGraphQL(dataValue, &data)
		if err != nil {
			return jerr.Get("error marshaling subscription for address listen", err)
		}
		if err = fn(data.Addresses); err != nil {
			errorChan <- jerr.Get("error in address listen subscription message handler", err)
			return nil
		}
		return nil
	}
	var graphQlVars = map[string]interface{}{"addresses": addresses}
	if _, err := client.Subscribe(&subscription, graphQlVars, newTxHandler); err != nil {
		return jerr.Get("error subscribing to graphql for address listen", err)
	}
	go func() {
		if err := client.Run(); err != nil {
			errorChan <- jerr.Get("error running graphql address listen subscription client", err)
		}
	}()
	return nil
}
