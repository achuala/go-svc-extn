package hash

import (
	"context"

	"github.com/pkg/errors"

	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct {
	c *BcryptConfiguration
}

type BcryptConfiguration struct {
	Cost int
}

func NewHasherBcrypt(c *BcryptConfiguration) *Bcrypt {
	return &Bcrypt{c: c}
}

func (h *Bcrypt) Generate(ctx context.Context, password []byte) ([]byte, error) {

	if err := validateBcryptPasswordLength(password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword(password, 10)
	if err != nil {
		return nil, err
	}

	return hash, nil
}

func validateBcryptPasswordLength(password []byte) error {
	// Bcrypt truncates the password to the first 72 bytes, following the OpenBSD implementation,
	// so if password is longer than 72 bytes, function returns an error
	// See https://en.wikipedia.org/wiki/Bcrypt#User_input
	if len(password) > 72 {
		return errors.New("password cannot exceed 72 bytes")
	}
	return nil
}

func (h *Bcrypt) Understands(hash []byte) bool {
	return IsBcryptHash(hash)
}
