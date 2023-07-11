package eth

import (
	"time"

	jsontime "github.com/liamylian/jsontime/v2/v2"
)

func init() {
	jsontime.SetDefaultTimeFormat(time.RFC3339Nano, time.Local)
}

type NFTTransfer struct {
	TokenAddress    string    `json:"tokenAddress"`
	TokenID         string    `json:"tokenId"`
	FromAddress     string    `json:"fromAddress"`
	ToAddress       string    `json:"toAddress"`
	ContractType    string    `json:"contractType"`
	Price           string    `json:"price"`
	Quantity        string    `json:"quantity"`
	BlockNumber     string    `json:"blockNumber"`
	BlockTimestamp  time.Time `json:"blockTimestamp"`
	BlockHash       string    `json:"blockHash"`
	TransactionHash string    `json:"transactionHash"`
	TransactionType string    `json:"transactionType"`
}

type NFTTransferResponse struct {
	PageSize   int           `json:"pageSize"`
	PageNumber int           `json:"pageNumber"`
	Cursor     string        `json:"cursor"`
	Transfers  []NFTTransfer `json:"transfers"`
}
