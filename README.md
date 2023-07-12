# Ethereum Webinar

## Pulling NFT transfers in bulk

```sh
go run bulk/main.go --key <infura-key> --secret <infura-secret>
```

## Streaming NFT transfers to Pulsar

```sh
go run stream/main.go --key <infura-key> --secret <infura-secret> --token
```


Notes
* Slicing is broken
* when(foo | shift_by(...)) is broken

```
%%fenl --var input_features_df
{
    avg_transfer: Transfer.Price | mean(window=sliding(10, is_valid(Transfer))),
    avg_purchase: (Transfer.Price * (Transfer.Quantity as i64)) | mean() | else(0),
    token_transfers: (Transfer.Quantity as i64) | else(0),
    transfer_value: Transfer.Price | last(),
} | when(Transfer.FromAddress == "0x0000000000000000000000000000000000000000")
```
