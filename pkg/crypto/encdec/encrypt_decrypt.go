package encdec

import (
	"context"
)

// Crypto provides methods for encrypting and decrypting data
type CryptoHandler interface {
	// Encrypts the plain data and returns a cipher data
	Encrypt(ctx context.Context, plain []byte) ([]byte, error)
	// Decrypts the cipher data and returns plain data
	Decrypt(ctx context.Context, cipher []byte) ([]byte, error)
}

type CryptoProvider interface {
	CryptoHandler(ctx context.Context) CryptoHandler
}
