package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// WalletGob is used for custom gob encoding/decoding of Wallet.
type WalletGob struct {
	PrivateKey []byte
	PublicKey  []byte
}

func (w Wallet) Address() []byte {
	publicKeyHash := PublicKeyHash(w.PublicKey)
	versionedPayload := append([]byte{version}, publicKeyHash...)
	checksum := Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))
	return bytes.Equal(actualChecksum, targetChecksum)

}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	public := elliptic.Marshal(curve, private.PublicKey.X, private.PublicKey.Y)
	return *private, public

}

func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}
	return &wallet

}

func PublicKeyHash(publicKey []byte) []byte {
	// Perform SHA-256 hashing
	hash := sha256.Sum256(publicKey)
	// Perform RIPEMD-160 hashing
	ripemd160Hasher := ripemd160.New()
	_, err := ripemd160Hasher.Write(hash[:])
	if err != nil {
		log.Panic(err)
	}
	publicRipMD := ripemd160Hasher.Sum(nil)
	return publicRipMD
}

func Checksum(payload []byte) []byte {
	hash := sha256.Sum256(payload)
	hash = sha256.Sum256(hash[:])
	return hash[:checksumLength]
}

func (w *Wallet) Bytes() ([]byte, []byte) {
	return w.PrivateKey.D.Bytes(), w.PublicKey
}

func (w *Wallet) LoadFromBytes(privKey, pubKey []byte) {
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, pubKey)

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = curve
	priv.PublicKey.X = x
	priv.PublicKey.Y = y
	priv.D = new(big.Int).SetBytes(privKey)

	w.PrivateKey = *priv
	w.PublicKey = pubKey
}

// GobEncode encodes the Wallet into a byte slice.
func (w *Wallet) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	data := WalletGob{
		PrivateKey: w.PrivateKey.D.Bytes(),
		PublicKey:  w.PublicKey,
	}
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode decodes the Wallet from a byte slice.
func (w *Wallet) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)
	var data WalletGob
	if err := decoder.Decode(&data); err != nil {
		return err
	}
	w.LoadFromBytes(data.PrivateKey, data.PublicKey)
	return nil
}
