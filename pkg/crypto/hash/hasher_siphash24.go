package hash

import (
	"context"
	"encoding/base64"

	"github.com/dchest/siphash"
	"github.com/pkg/errors"
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
		return nil, errors.WithStack(err)
	}
	hash := hasher.Sum(nil)
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(hash)))
	base64.StdEncoding.Encode(dst, hash)
	return dst, nil
}

func (h *SipHash24) Understands(hash []byte) bool {
	return IsSip24Hash(hash)
}
