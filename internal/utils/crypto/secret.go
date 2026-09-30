package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// SecretBox encrypts provider credentials with authenticated associated data.
type SecretBox struct{ aead cipher.AEAD }

func NewSecretBox(hexKey string) (*SecretBox, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("radar.encryption_key must be 64 hexadecimal characters")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: aead}, nil
}

func (b *SecretBox) Encrypt(userID uint64, provider, secret string) ([]byte, []byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext := b.aead.Seal(nil, nonce, []byte(secret), []byte(fmt.Sprintf("%d:%s", userID, provider)))
	return ciphertext, nonce, nil
}

func (b *SecretBox) Decrypt(userID uint64, provider string, ciphertext, nonce []byte) (string, error) {
	plain, err := b.aead.Open(nil, nonce, ciphertext, []byte(fmt.Sprintf("%d:%s", userID, provider)))
	if err != nil {
		return "", errors.New("credential decryption failed")
	}
	return string(plain), nil
}
