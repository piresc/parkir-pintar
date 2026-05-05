// Best practices applied from Go testing guidelines:
// - Descriptive test names using ShouldXXX_WhenYYY pattern
// - AAA (Arrange-Act-Assert) structure
// - Table-driven tests for multiple scenarios
// - Comprehensive coverage of success and error cases
// - testify assertions (assert, require)

package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AES Cryptor Tests ---

func TestAESEncryptDecrypt_ShouldRoundtrip_WhenValidInput(t *testing.T) {
	// Arrange
	cryptor := NewCryptor("my-secret-password")
	plaintext := "hello world, this is a test message"

	// Act
	encrypted, err := cryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := cryptor.Decrypt(encrypted)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, plaintext, decrypted)
	assert.NotEqual(t, plaintext, encrypted)
}

func TestAESEncrypt_ShouldProduceDifferentCiphertext_WhenCalledTwice(t *testing.T) {
	// Arrange — random IV means each encryption produces different output
	cryptor := NewCryptor("my-secret-password")
	plaintext := "same input"

	// Act
	enc1, err := cryptor.Encrypt(plaintext)
	require.NoError(t, err)

	enc2, err := cryptor.Encrypt(plaintext)
	require.NoError(t, err)

	// Assert
	assert.NotEqual(t, enc1, enc2)
}

func TestAESEncrypt_ShouldReturnError_WhenPlaintextEmpty(t *testing.T) {
	// Arrange
	cryptor := NewCryptor("key")

	// Act
	result, err := cryptor.Encrypt("")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestAESDecrypt_ShouldReturnError_WhenCiphertextEmpty(t *testing.T) {
	// Arrange
	cryptor := NewCryptor("key")

	// Act
	result, err := cryptor.Decrypt("")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestAESDecrypt_ShouldReturnError_WhenInvalidBase64(t *testing.T) {
	// Arrange
	cryptor := NewCryptor("key")

	// Act
	result, err := cryptor.Decrypt("not-valid-base64!!!")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestAESDecrypt_ShouldReturnError_WhenWrongKey(t *testing.T) {
	// Arrange
	cryptor1 := NewCryptor("key-one")
	cryptor2 := NewCryptor("key-two")

	encrypted, err := cryptor1.Encrypt("secret data")
	require.NoError(t, err)

	// Act
	result, err := cryptor2.Decrypt(encrypted)

	// Assert — decryption with wrong key should fail (bad padding)
	assert.Error(t, err)
	assert.Empty(t, result)
}

// --- RSA Tests ---

func generateTestRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privKey, &privKey.PublicKey
}

func TestRSAEncryptDecrypt_ShouldRoundtrip_WhenValidKeys(t *testing.T) {
	// Arrange
	privKey, pubKey := generateTestRSAKeys(t)
	plaintext := []byte("RSA test message")

	// Act
	ciphertext, err := EncryptRSA(pubKey, plaintext)
	require.NoError(t, err)

	decrypted, err := DecryptRSA(privKey, ciphertext)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptRSA_ShouldReturnError_WhenPublicKeyNil(t *testing.T) {
	// Act
	result, err := EncryptRSA(nil, []byte("data"))

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestEncryptRSA_ShouldReturnError_WhenPlaintextEmpty(t *testing.T) {
	// Arrange
	_, pubKey := generateTestRSAKeys(t)

	// Act
	result, err := EncryptRSA(pubKey, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDecryptRSA_ShouldReturnError_WhenPrivateKeyNil(t *testing.T) {
	// Act
	result, err := DecryptRSA(nil, []byte("data"))

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseRSAPublicKey_ShouldReturnKey_WhenValidPEM(t *testing.T) {
	// Arrange
	privKey, _ := generateTestRSAKeys(t)
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	require.NoError(t, err)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	// Act
	key, err := ParseRSAPublicKey(pemData)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestParseRSAPublicKey_ShouldReturnError_WhenInvalidPEM(t *testing.T) {
	// Act
	key, err := ParseRSAPublicKey([]byte("not a pem"))

	// Assert
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestParseRSAPrivateKey_ShouldReturnKey_WhenValidPKCS1PEM(t *testing.T) {
	// Arrange
	privKey, _ := generateTestRSAKeys(t)
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})

	// Act
	key, err := ParseRSAPrivateKey(pemData)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestParseRSAPrivateKey_ShouldReturnKey_WhenValidPKCS8PEM(t *testing.T) {
	// Arrange
	privKey, _ := generateTestRSAKeys(t)
	privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	require.NoError(t, err)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	// Act
	key, err := ParseRSAPrivateKey(pemData)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestParseRSAPrivateKey_ShouldReturnError_WhenInvalidPEM(t *testing.T) {
	// Act
	key, err := ParseRSAPrivateKey([]byte("not a pem"))

	// Assert
	assert.Error(t, err)
	assert.Nil(t, key)
}

// --- HMAC Signature Tests ---

func TestGenerateSignature_ShouldReturnHexString_WhenValidInput(t *testing.T) {
	// Arrange
	payload := "test-payload"
	secret := "test-secret"

	// Act
	sig := GenerateSignature(payload, secret)

	// Assert
	assert.NotEmpty(t, sig)
	assert.Len(t, sig, 64) // SHA-256 produces 32 bytes = 64 hex chars
}

func TestGenerateSignature_ShouldBeDeterministic_WhenSameInput(t *testing.T) {
	// Arrange
	payload := "same-payload"
	secret := "same-secret"

	// Act
	sig1 := GenerateSignature(payload, secret)
	sig2 := GenerateSignature(payload, secret)

	// Assert
	assert.Equal(t, sig1, sig2)
}

func TestGenerateSignature_ShouldDiffer_WhenDifferentPayload(t *testing.T) {
	// Arrange
	secret := "same-secret"

	// Act
	sig1 := GenerateSignature("payload-1", secret)
	sig2 := GenerateSignature("payload-2", secret)

	// Assert
	assert.NotEqual(t, sig1, sig2)
}

func TestValidateSignature_ShouldReturnNil_WhenSignatureValid(t *testing.T) {
	// Arrange
	payload := "validate-me"
	secret := "my-secret"
	timestamp := time.Now().Unix()
	sig := GenerateSignature(payload, secret)

	// Act
	err := ValidateSignature(sig, payload, secret, 5*time.Minute, timestamp)

	// Assert
	assert.NoError(t, err)
}

func TestValidateSignature_ShouldReturnError_WhenSignatureExpired(t *testing.T) {
	// Arrange
	payload := "expired-payload"
	secret := "my-secret"
	timestamp := time.Now().Add(-10 * time.Minute).Unix()
	sig := GenerateSignature(payload, secret)

	// Act
	err := ValidateSignature(sig, payload, secret, 5*time.Minute, timestamp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature expired")
}

func TestValidateSignature_ShouldReturnError_WhenSignatureMismatch(t *testing.T) {
	// Arrange
	payload := "my-payload"
	secret := "my-secret"
	timestamp := time.Now().Unix()

	// Act
	err := ValidateSignature("bad-signature-hex", payload, secret, 5*time.Minute, timestamp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestValidateSignature_ShouldReturnError_WhenSignatureEmpty(t *testing.T) {
	// Act
	err := ValidateSignature("", "payload", "secret", 5*time.Minute, time.Now().Unix())

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature is required")
}

func TestValidateSignature_ShouldReturnError_WhenPayloadEmpty(t *testing.T) {
	// Act
	err := ValidateSignature("sig", "", "secret", 5*time.Minute, time.Now().Unix())

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload is required")
}

func TestValidateSignature_ShouldReturnError_WhenSecretEmpty(t *testing.T) {
	// Act
	err := ValidateSignature("sig", "payload", "", 5*time.Minute, time.Now().Unix())

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret is required")
}
