package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version = byte(0x00)
)
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey []byte
}

func (w Wallet) Address() []byte {
	publicKeyHash := PublicKeyHash(w.PublicKey)
	versionedPayload := append([]byte{version}, publicKeyHash...)
	checksum := Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)
	
	return address
}

func NewKeyPair() (ecdsa.PrivateKey, []byte)  {
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