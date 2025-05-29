package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"github.com/nthskyradiated/go-bc/blockchain"
	"github.com/nthskyradiated/go-bc/wallet"
	"github.com/nthskyradiated/go-bc/network"
)
type CommandLine struct {}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  getbalance -address ADDRESS - Get balance of an address")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  print - Print the blockchain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT -mine - Send amount of coins. Then -mine flag is set, mine off of this node")
	fmt.Println("  createwallet - Create a new Wallet")
	fmt.Println("  listaddresses - List the addresses in our wallet file")
	fmt.Println("  reindexutxo - Rebuilds the UTXO set")
	fmt.Println(" startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) StartNode(nodeId, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeId)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	network.StartServer(nodeId, minerAddress)
}

func (cli *CommandLine) listAddresses(nodeId string) {
	wallets, _ := wallet.NewWallets(nodeId)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}


func (cli *CommandLine) createWallet(nodeId string) {
	wallets, _ := wallet.NewWallets(nodeId)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeId)

	fmt.Printf("New address is: %s\n", address)
}


func (cli *CommandLine) printChain(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	iter := chain.Iterator()
	for  {
		block := iter.Next()
		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createblockchain(address, nodeId string) {
		if !wallet.ValidateAddress(address) {
		log.Panicf("Invalid address: %s", address)
	}
	chain := blockchain.NewBlockChain(address, nodeId)
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()
	fmt.Println("Blockchain created successfully!")
}

func (cli *CommandLine) reindexUTXO(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) getbalance(address, nodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panicf("Invalid address: %s", address)
	}
	chain := blockchain.ContinueBlockChain(nodeId)
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
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

func (cli *CommandLine) send(from, to string, amount int, nodeId string, mineNow bool) {
		if !wallet.ValidateAddress(to) {
		log.Panic("Address is not Valid")	
	}
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is not Valid")	
	}
	chain := blockchain.ContinueBlockChain(nodeId)
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

		wallets, err := wallet.NewWallets(nodeId)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx := blockchain.NewTransaction(&wallet, to, amount, &UTXOSet)
	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		block := chain.MineBlock([]*blockchain.Transaction{cbTx, tx})
		UTXOSet.Update(block)
		
		} else {
			network.SendTx(network.KnownNodes[0], tx)
			fmt.Println("send tx")
		}
		fmt.Println("Transaction successful!")
}

func (cli *CommandLine) Run() {
	cli.validateArgs()
		nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime.Goexit()
	}
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "Address to get balance of")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "Address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Address to send from")
	sendTo := sendCmd.String("to", "", "Address to send to")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
			case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
		case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.HandleError(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getbalance(*getBalanceAddress, nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createblockchain(*createBlockchainAddress, nodeID)
		fmt.Printf("Creating blockchain with genesis block reward to %s\n", nodeID)
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
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		fmt.Printf("Starting node with ID: %s\n", nodeID)
		if nodeID == "" {
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}
}