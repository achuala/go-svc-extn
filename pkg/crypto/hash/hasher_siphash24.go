package hash

import (
	"context"
	"encoding/base64"

	"github.com/dchest/siphash"
)

type SipHash24 struct {
	c *SipHashConfiguration
}

type SipHashConfiguration struct {
	Key string
}

func NewHasherSipHash24(c *SipHashConfiguration) *SipHash24 {
	return &SipHash24{c: c}
}

func (h *SipHash24) Generate(ctx context.Context, data []byte) ([]byte, error) {

	hasher := siphash.New([]byte(h.c.Key))
	if _, err := hasher.Write(data); err != nil {
		return nil, err
	}
	hash := hasher.Sum(nil)

	return []byte(base64.StdEncoding.EncodeToString(hash)), nil

}

func (h *SipHash24) Understands(hash []byte) bool {
	return IsSip24Hash(hash)
}
