#!/bin/sh
rm -rf ./tmp && mkdir tmp
export CHAIN="go run main.go"
export ADDR1=$($CHAIN createwallet)
export ADDR2=$($CHAIN createwallet)
$CHAIN createblockchain -address $ADDR1
# $CHAIN reindexutxo
$CHAIN getbalance -address $ADDR1
$CHAIN getbalance -address $ADDR2
$CHAIN send -from $ADDR1 -to $ADDR2 -amount 5
$CHAIN getbalance -address $ADDR1
$CHAIN getbalance -address $ADDR2
$CHAIN printchain
