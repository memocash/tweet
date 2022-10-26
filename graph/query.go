package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func BasicQuery() {
	jsonData := map[string]string{
		"query": `
            {
                address (address: "1JJhfR2fD3mmxipTXxdtvehBiep1WNzM9q") {
                    balance
                }
            }
        `,
	}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "http://localhost:26770/graphql", bytes.NewBuffer(jsonValue))
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	defer response.Body.Close()
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	}
	data, _ := ioutil.ReadAll(response.Body)
	fmt.Println(string(data))
}
