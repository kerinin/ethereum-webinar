package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	eth "github.com/kerinin/ethereum-webinar"
	jsontime "github.com/liamylian/jsontime/v2/v2"
	"github.com/segmentio/parquet-go"
	"github.com/segmentio/parquet-go/compress/snappy"
)

var (
	key       = flag.String("key", "", "Infura API key")
	secret    = flag.String("secret", "", "Infura API secret")
	network   = flag.String("network", "1", "Ethereum network")
	startFlag = flag.Int("start", 0, "initial block")
	endFlag   = flag.Int("end", 18_000_000, "final block")
	batchSize = flag.Int("batch", 100_000, "batch size before writing to Parquet file")
	cursor    = flag.String("cursor", "", "(Optional) Cursor to resume from")
	json      = jsontime.ConfigWithCustomTimeFormat
)

func main() {
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Make the API request
	url := fmt.Sprintf("https://nft.api.infura.io/networks/%s/nfts/transfers", *network)

	transferBatch := make([]eth.NFTTransfer, 0, *batchSize)
	defer func() {
		if len(transferBatch) > 0 {
			err := parquet.WriteFile(fmt.Sprintf("transfers_%d.parquet", time.Now().Unix()), transferBatch, parquet.Compression(&snappy.Codec{}))
			if err != nil {
				log.Fatalf("Failed to write parquet: %s", err)
			}
			fmt.Printf("Wrote batch of %d transfers to Parquet\n", len(transferBatch))
		}
	}()

	for {
		req, err := retryablehttp.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Failed to create request:", err)
			return
		}
		q := req.URL.Query()
		q.Add("fromBlock", strconv.Itoa(*startFlag))
		q.Add("toBlock", strconv.Itoa(*endFlag))
		if *cursor != "" {
			q.Add("cursor", *cursor)
		}
		req.URL.RawQuery = q.Encode()
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(*key, *secret)

		client := retryablehttp.NewClient()
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

		if resp.StatusCode == http.StatusBadRequest {
			fmt.Println("API request failed with status (retrying):", resp.StatusCode)
			continue
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

		transferBatch = append(transferBatch, transferResponse.Transfers...)
		if len(transferBatch) >= *batchSize {
			err := parquet.WriteFile(fmt.Sprintf("transfers_%d.parquet", time.Now().Unix()), transferBatch, parquet.Compression(&snappy.Codec{}))
			if err != nil {
				log.Fatalf("Failed to write parquet: %s", err)
			}
			fmt.Printf("Wrote batch of %d transfers to Parquet\n", len(transferBatch))
			transferBatch = transferBatch[:0]
		}

		fmt.Printf("%d txns with cursor %s @ %s\n", len(transferResponse.Transfers), lastTransfer.BlockNumber, lastTransfer.BlockTimestamp)
		if transferResponse.Cursor == "" {
			fmt.Println("---")
			break
		}
		if ctx.Err() != nil {
			break
		}
		*cursor = transferResponse.Cursor
	}
}
