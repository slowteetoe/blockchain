package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/slowteetoe/blockchain/blockchain"
	"github.com/slowteetoe/blockchain/network"
	"github.com/slowteetoe/blockchain/wallet"
)

type CommandLine struct {
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("\t getbalance -address <ADDRESS> - get the balance for an address")
	fmt.Println("\t createblockchain -address <ADDRESS> - creates a blockchain")
	fmt.Println("\t printchain - Prints the blocks in the chain")
	fmt.Println("\t send -from <FROM> -to <TO> -amount <AMOUNT> -mine - send the amount from address to the other address")
	fmt.Println("\t createwallet - create a new wallet")
	fmt.Println("\t listaddresses - list the addresses in our wallet file")
	fmt.Println("\t reindexutxo - Rebuilds the UTXO set")
	fmt.Println("\t startnode -miner <ADDRESS> - start a node with ID specified in NODE_ID env")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()

	iter := chain.Iterator()

	for {
		block := iter.Next()
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Prev hash: %x\n", block.PrevHash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address, nodeID string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.InitBlockChain(address, nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{BlockChain: chain}
	UTXOSet.Reindex()
	fmt.Println("Blockchain created")
}

func (cli *CommandLine) getBalance(address, nodeID string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.ContinueBlockChain(nodeID)
	UTXOSet := blockchain.UTXOSet{BlockChain: chain}
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int, nodeID string, mineNow bool) {
	if !wallet.ValidateAddress(from) {
		log.Panic("From address is not valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("To address is not valid")
	}
	log.Printf("sending %d from %s to %s", amount, from, to)
	chain := blockchain.ContinueBlockChain(nodeID)
	if chain == nil {
		log.Panic("Unable to continue blockchain")
	}
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{BlockChain: chain}

	wallets, err := wallet.CreateWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	log.Println("Got wallets")

	wallet := wallets.GetWallet(from)
	if wallet == nil {
		log.Panic("wallet not found for ", from)
	}

	log.Printf("wallet for %s found: %x", from, wallet)

	tx := blockchain.NewTransaction(wallet, to, amount, &UTXOSet)
	fmt.Println("transaction created")
	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
		fmt.Println("mined blocks")
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("sent tx over wire")
	}
	fmt.Println("Success!")
}

func (cli *CommandLine) listAddresses(nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet(nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeID)
	fmt.Printf("%s\n", address)
}

func (cli *CommandLine) startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Invalid miner address!")
		}
	}
	network.StartServer(nodeID, minerAddress)
}

func (cli *CommandLine) reindexUTXO(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{BlockChain: chain}
	UTXOSet.Reindex()
	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Println("NODE_ID env must be set")
		runtime.Goexit()
	}

	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createBlockchainAddress := createBlockchainCmd.String("address", "", "the genesis address")

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	getBalanceAddress := getBalanceCmd.String("address", "", "the address")

	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Dest. wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	mineNow := sendCmd.Bool("mine", false, "Mine immediately on the same node")

	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)

	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)

	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining node and send reward to ")

	switch os.Args[1] {
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress, nodeID)
	}

	if sendCmd.Parsed() {
		if *sendAmount == 0 || *sendTo == "" || *sendFrom == "" {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *mineNow)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

	if startNodeCmd.Parsed() {
		cli.startNode(nodeID, *startNodeMiner)
	}
}
