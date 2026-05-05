package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Cryptor defines the interface for symmetric encryption/decryption.
type Cryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// aesCryptor implements Cryptor using AES-256-CBC with PKCS7 padding.
type aesCryptor struct {
	key []byte // 32-byte key derived via SHA-256
}

// NewCryptor creates a new AES-256-CBC Cryptor.
// The key is derived from the provided string using SHA-256 to produce 32 bytes.
func NewCryptor(key string) Cryptor {
	hash := sha256.Sum256([]byte(key))
	return &aesCryptor{key: hash[:]}
}

// Encrypt encrypts plaintext using AES-256-CBC with a random IV.
// The IV is prepended to the ciphertext and the result is base64-encoded.
func (c *aesCryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("plaintext is required")
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)

	// Random IV
	ciphertext := make([]byte, aes.BlockSize+len(padded))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], padded)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes a base64 ciphertext, extracts the prepended IV,
// and decrypts using AES-256-CBC.
func (c *aesCryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("ciphertext is required")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	if len(data)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of block size")
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	iv := data[:aes.BlockSize]
	encrypted := data[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	plain := make([]byte, len(encrypted))
	mode.CryptBlocks(plain, encrypted)

	unpadded, err := pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	return string(unpadded), nil
}

// pkcs7Pad pads data to a multiple of blockSize using PKCS7.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}

// pkcs7Unpad removes PKCS7 padding from data.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	if len(data)%blockSize != 0 {
		return nil, errors.New("invalid padding size")
	}

	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize {
		return nil, errors.New("invalid padding")
	}

	for i := 0; i < pad; i++ {
		if data[len(data)-1-i] != byte(pad) {
			return nil, errors.New("invalid padding bytes")
		}
	}

	return data[:len(data)-pad], nil
}
