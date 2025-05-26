package wallet

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
)

const walletFile = "./tmp/wallets.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallets() (*Wallets, error) {
	ws := Wallets{}
	ws.Wallets = make(map[string]*Wallet)
	err := ws.LoadFile()
	return &ws, err

}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := string(wallet.Address())
	ws.Wallets[address] = wallet
	ws.SaveFile()
	log.Printf("New wallet created with address: %s", address)
	return address
}

func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}

func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

type SerializedWallet struct {
	PrivateKey []byte
	PublicKey  []byte
}

type SerializedWallets struct {
	Wallets map[string]SerializedWallet
}

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	var serialized SerializedWallets
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&serialized)
	if err != nil {
		return err
	}

	wallets := make(map[string]*Wallet)
	for addr, data := range serialized.Wallets {
		wallet := &Wallet{}
		wallet.LoadFromBytes(data.PrivateKey, data.PublicKey)
		wallets[addr] = wallet
	}

	ws.Wallets = wallets
	return nil
}

func (ws *Wallets) SaveFile() {
	serialized := SerializedWallets{
		Wallets: make(map[string]SerializedWallet),
	}

	for addr, wallet := range ws.Wallets {
		privKey, pubKey := wallet.Bytes()
		serialized.Wallets[addr] = SerializedWallet{
			PrivateKey: privKey,
			PublicKey:  pubKey,
		}
	}

	var content bytes.Buffer
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(serialized)
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
