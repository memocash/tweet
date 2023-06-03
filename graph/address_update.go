package graph

import (
	"context"
	"github.com/hasura/go-graphql-client"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/tweet/config"
	"time"
)

func GetAddressUpdates(address string, start time.Time) ([]Tx, error) {
	client := graphql.NewClient(config.GetGraphQlUrl(), nil)
	var updateQuery = new(UpdateQuery)
	var vars = map[string]interface{}{
		"address": address,
	}
	var startDate string
	if !jutil.IsTimeZero(start) {
		startDate = start.Format(time.RFC3339)
	} else {
		startDate = time.Date(2009, 1, 1, 0, 0, 0, 0, time.Local).Format(time.RFC3339)
	}
	vars["start"] = Date(startDate)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Query(ctx, updateQuery, vars); err != nil {
		return nil, jerr.Get("error querying graphql process missed txs", err)
	}
	return updateQuery.Address.Txs, nil
}
