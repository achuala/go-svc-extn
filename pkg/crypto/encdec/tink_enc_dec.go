package encdec

import (
	"bytes"
	"context"
	"encoding/base64"
	"log"

	"github.com/pkg/errors"
	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

type TinkConfiguration struct {
	KekUri     string
	KeySetData string
}

type TinkCryptoHandler struct {
	ksh  *keyset.Handle
	aead tink.AEAD
}

func NewTinkCryptoHandler(c *TinkConfiguration) *TinkCryptoHandler {
	client, err := newCaasKmsClient(c.KekUri)
	if err != nil {
		log.Fatal(err)
	}
	kekAEAD, err := client.GetAEAD(c.KekUri)
	if err != nil {
		log.Fatal(err)
	}

	keysetAssociatedData := []byte("caas kek")

	encryptedKeyset, err := base64.RawURLEncoding.DecodeString(c.KeySetData)
	if err != nil {
		log.Fatal(err)
	}
	// To use the primitive, we first need to decrypt the keyset. We use the same
	// KEK AEAD and the same associated data that we used to encrypt it.
	reader := keyset.NewBinaryReader(bytes.NewReader(encryptedKeyset))
	handle, err := keyset.ReadWithAssociatedData(reader, kekAEAD, keysetAssociatedData)
	if err != nil {
		log.Fatal(err)
	}

	aead, err := aead.New(handle)

	if err != nil {
		log.Fatal(err)
	}

	return &TinkCryptoHandler{ksh: handle, aead: aead}
}

func (h *TinkCryptoHandler) Encrypt(ctx context.Context, plain []byte) ([]byte, error) {
	ad := []byte("caas ums")
	cipher, err := h.aead.Encrypt(plain, ad)
	if err != nil {
		return nil, errors.Wrap(err, "unable to encrypt")
	}
	return cipher, nil
}

func (h *TinkCryptoHandler) Decrypt(ctx context.Context, cipher []byte) ([]byte, error) {
	ad := []byte("caas ums")
	decrypted, err := h.aead.Decrypt(cipher, ad)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decrypt")
	}
	return decrypted, nil
}
