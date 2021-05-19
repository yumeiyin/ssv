package rsaencryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/pkg/errors"
)

var keySize = 2048

// GenerateKeys using rsa random generate keys and return []byte bas64
func GenerateKeys() ([]byte, []byte, error) {
	// generate random private key (secret)
	sk, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to generate private key")
	}
	// retrieve public key from the newly generated secret
	pk := &sk.PublicKey

	// convert to bytes
	skPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(sk),
		},
	)
	pkBytes, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to marshal public key")
	}
	pkPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pkBytes,
		},
	)
	return pkPem, skPem, nil
}

// DecodeKey with secret key (base64) and hash (base64), return the encrypted key string
func DecodeKey(sk *rsa.PrivateKey, hashBase64 string) (string, error) {
	return decryptHash(sk, hashBase64)
}

// ConvertPemToPrivateKey return rsa private key from secret key (base64)
func ConvertPemToPrivateKey(skPemBase64 string) (*rsa.PrivateKey, error) {
	skPem, err := base64.StdEncoding.DecodeString(skPemBase64)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to decode base64")
	}
	block, _ := pem.Decode(skPem)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	if enc {
		var err error
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to decrypt private key")
		}
	}
	parsedSk, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse private key")
	}
	return parsedSk, nil
}

func ExtractPublicKey(sk *rsa.PrivateKey) ([]byte, error) {
	pkBytes, err := x509.MarshalPKIXPublicKey(&sk.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal private key")
	}
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pkBytes,
		},
	), nil
}

// decryptHash using secret key and encrypted hash
func decryptHash(sk *rsa.PrivateKey, hashBase64 string) (string, error) {
	hash, _ := base64.StdEncoding.DecodeString(hashBase64)
	decryptedKey, err := rsa.DecryptPKCS1v15(rand.Reader, sk, hash)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decrypt key")
	}
	return string(decryptedKey), nil
}
