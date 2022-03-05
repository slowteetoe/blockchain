#!/bin/sh
rm -rf ./tmp && mkdir tmp
export ADDR1=$(go run main.go createwallet)
export ADDR2=$(go run main.go createwallet)
go run main.go createblockchain -address $ADDR1
# go run main.go reindexutxo
go run main.go getbalance -address $ADDR1
go run main.go getbalance -address $ADDR2
go run main.go send -from $ADDR1 -to $ADDR2 -amount 5
go run main.go getbalance -address $ADDR1
go run main.go getbalance -address $ADDR2
