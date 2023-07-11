package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	eth "github.com/kerinin/ethereum-webinar"
	jsontime "github.com/liamylian/jsontime/v2/v2"
)

var (
	key              = flag.String("key", "", "Infura API key")
	secret           = flag.String("secret", "", "Infura API secret")
	network          = flag.String("network", "1", "Ethereum network")
	brokerServiceUrl = flag.String("broker-service-url", "pulsar+ssl://pulsar-gcp-useast1.streaming.datastax.com:6651", "The Pulsar URL to produce new events to")
	// authPlugin       = flag.String("auth-plugin", "org.apache.pulsar.client.impl.auth.AuthenticationToken", "The Pulsar auth plugin to use")
	token     = flag.String("token", "", "The Pulsar auth token to send")
	topicName = flag.String("topic-name", "persistent://ethereum-webinar/default/token_transfers", "The Pulsar topic to write to")
	json      = jsontime.ConfigWithCustomTimeFormat
)

func main() {
	flag.Parse()

	cursor := flag.Arg(1)

	pulsarClient, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            *brokerServiceUrl,
		Authentication: pulsar.NewAuthenticationToken(*token),
	})
	if err != nil {
		log.Fatalf("Failed to create pulsar client: %s", err)
	}
	defer pulsarClient.Close()

	p, err := pulsarClient.TopicPartitions(*topicName)
	if err != nil {
		log.Fatalf("Failed to get topic partitions: %s", err)
	}
	fmt.Printf("Partitions for topic %s: %v\n", *topicName, p)

	producer, err := pulsarClient.CreateProducer(pulsar.ProducerOptions{
		Topic: *topicName,
	})
	if err != nil {
		log.Fatalf("Failed to create pulsar producer: %s", err)
	}
	defer producer.Close()

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

				for _, transfer := range transferResponse.Transfers {
					_, err = producer.Send(context.Background(), &pulsar.ProducerMessage{
						Value:     transfer,
						EventTime: transfer.BlockTimestamp,
					})
					if err != nil {
						log.Fatalf("Failed to write to Pulsar: %s", err)
					}
				}
				err = producer.Flush()
				if err != nil {
					log.Fatalf("Failed to flush Pulsar prodcuer")
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
