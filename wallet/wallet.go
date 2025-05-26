package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
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
	return bytes.Compare(actualChecksum, targetChecksum) == 0

}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	public := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
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
