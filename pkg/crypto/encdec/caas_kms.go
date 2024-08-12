package encdec

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/core/registry"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/testkeyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

const caasKmsPrefix = "caas-kms://"

var _ registry.KMSClient = (*caasKmsClient)(nil)

type caasKmsClient struct {
	uriPrefix string
}

// NewClient returns a fake KMS client which will handle keys with uriPrefix prefix.
// keyURI must have the following format: 'caas-kms://<base64 encoded aead keyset>'.
func newCaasKmsClient(uriPrefix string) (registry.KMSClient, error) {
	if !strings.HasPrefix(strings.ToLower(uriPrefix), caasKmsPrefix) {
		return nil, fmt.Errorf("uriPrefix must start with %s, but got %s", caasKmsPrefix, uriPrefix)
	}
	return &caasKmsClient{
		uriPrefix: uriPrefix,
	}, nil
}

// Supported returns true if this client does support keyURI.
func (c *caasKmsClient) Supported(keyURI string) bool {
	return strings.HasPrefix(keyURI, c.uriPrefix)
}

// GetAEAD returns an AEAD by keyURI.
func (c *caasKmsClient) GetAEAD(keyURI string) (tink.AEAD, error) {
	if !c.Supported(keyURI) {
		return nil, fmt.Errorf("keyURI must start with prefix %s, but got %s", c.uriPrefix, keyURI)
	}
	encodeKeyset := strings.TrimPrefix(keyURI, caasKmsPrefix)
	keysetData, err := base64.RawURLEncoding.DecodeString(encodeKeyset)
	if err != nil {
		return nil, err
	}
	reader := keyset.NewBinaryReader(bytes.NewReader(keysetData))
	handle, err := testkeyset.Read(reader)
	if err != nil {
		return nil, err
	}
	return aead.New(handle)
}

// NewKeyURI returns a new, random KMS key URI.
func NewKeyURI() (string, error) {
	handle, err := keyset.NewHandle(aead.AES128GCMKeyTemplate())
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	writer := keyset.NewBinaryWriter(buf)
	err = testkeyset.Write(handle, writer)
	if err != nil {
		return "", err
	}
	return caasKmsPrefix + base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}
