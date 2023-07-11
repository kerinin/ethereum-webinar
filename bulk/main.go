package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

var (
	key       = flag.String("key", "", "Infura API key")
	secret    = flag.String("secret", "", "Infura API secret")
	network   = flag.String("network", "1", "Ethereum network")
	startFlag = flag.Int("start", 0, "initial block")
	endFlag   = flag.Int("end", 18_000_000, "final block")
)

type NFTTransfer struct {
	TokenAddress    string `json:"tokenAddress"`
	TokenID         string `json:"tokenId"`
	FromAddress     string `json:"fromAddress"`
	ToAddress       string `json:"toAddress"`
	ContractType    string `json:"contractType"`
	Price           string `json:"price"`
	Quantity        string `json:"quantity"`
	BlockNumber     string `json:"blockNumber"`
	BlockTimestamp  string `json:"blockTimestamp"`
	TransactionType string `json:"transactionType"`
}

type NFTTransferResponse struct {
	PageSize   int           `json:"pageSize"`
	PageNumber int           `json:"pageNumber"`
	Cursor     string        `json:"cursor"`
	Transfers  []NFTTransfer `json:"transfers"`
}

func main() {
	flag.Parse()

	cursor := flag.Arg(1)

	// Make the API request
	url := fmt.Sprintf("https://nft.api.infura.io/networks/%s/nfts/transfers", *network)

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Failed to create request:", err)
			return
		}
		q := req.URL.Query()
		q.Add("fromBlock", strconv.Itoa(*startFlag))
		q.Add("toBlock", strconv.Itoa(*endFlag))
		if cursor != "" {
			q.Add("cursor", cursor)
		}
		req.URL.RawQuery = q.Encode()
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(*key, *secret)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Failed to send request:", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read response body:", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Println("API request failed with status:", resp.StatusCode)
			fmt.Println("Response:", string(body))
			return
		}

		// Parse the response
		var transferResponse NFTTransferResponse
		err = json.Unmarshal(body, &transferResponse)
		if err != nil {
			fmt.Println("Failed to parse response body:", err)
			return
		}

		if len(transferResponse.Transfers) == 0 {
			break
		}

		lastTransfer := transferResponse.Transfers[len(transferResponse.Transfers)-1]
		fmt.Printf("%d txns with cursor %s @ %s\n", len(transferResponse.Transfers), lastTransfer.BlockNumber, lastTransfer.BlockTimestamp)
		if transferResponse.Cursor == "" {
			fmt.Println("---")
			break
		}
		cursor = transferResponse.Cursor
	}
}
