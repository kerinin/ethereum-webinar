package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	eth "github.com/kerinin/ethereum-webinar"
)

var (
	key     = flag.String("key", "", "Infura API key")
	secret  = flag.String("secret", "", "Infura API secret")
	network = flag.String("network", "1", "Ethereum network")
)

func main() {
	flag.Parse()

	cursor := flag.Arg(1)

	url := fmt.Sprintf("wss://mainnet.infura.io/ws/v3/%s", *key)
	client, err := ethclient.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatalf("Failed to subscribe to new head events: %v", err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Subscription error: %v", err)
		case header := <-headers:
			// We have a new header, let's get its transactions
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Fatalf("Failed to get block: %v", err)
			}

			// Make the API request
			url := fmt.Sprintf("https://nft.api.infura.io/networks/%s/nfts/block/transfers", *network)

			for {
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					fmt.Println("Failed to create request:", err)
					return
				}
				q := req.URL.Query()
				q.Add("blockHashNumber", strconv.Itoa(int(block.NumberU64())))
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
				var transferResponse eth.NFTTransferResponse
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
	}
}
