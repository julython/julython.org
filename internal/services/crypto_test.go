package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef") // 16-byte AES key

	t.Run("encrypt and decrypt same value", func(t *testing.T) {
		plaintext := "hello, world"
		ciphertext, err := encrypt(key, plaintext)
		require.NoError(t, err)
		require.NotEmpty(t, ciphertext)
		require.NotEqual(t, plaintext, ciphertext)

		decrypted, err := decrypt(key, ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt produces different ciphertext each time", func(t *testing.T) {
		plaintext := "same value"
		c1, err := encrypt(key, plaintext)
		require.NoError(t, err)
		c2, err := encrypt(key, plaintext)
		require.NoError(t, err)
		// GCM with random nonce should produce different ciphertexts
		assert.NotEqual(t, c1, c2)
	})

	t.Run("decrypted value matches after multiple encryptions", func(t *testing.T) {
		plaintext := "encrypted multiple times"
		c1, _ := encrypt(key, plaintext)
		c2, _ := encrypt(key, plaintext)

		d1, err := decrypt(key, c1)
		require.NoError(t, err)
		d2, err := decrypt(key, c2)
		require.NoError(t, err)

		assert.Equal(t, plaintext, d1)
		assert.Equal(t, plaintext, d2)
	})
}

func TestEncryptDecryptLargeInput(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := string(make([]byte, 10000))
	ciphertext, err := encrypt(key, plaintext)
	require.NoError(t, err)
	decrypted, err := decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := ""
	ciphertext, err := encrypt(key, plaintext)
	require.NoError(t, err)
	decrypted, err := decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptUnicode(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := "hello 世界 🌍"
	ciphertext, err := encrypt(key, plaintext)
	require.NoError(t, err)
	decrypted, err := decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptWithDifferentKeys(t *testing.T) {
	key1 := []byte("0123456789abcdef")
	key2 := []byte("fedcba9876543210")
	plaintext := "secret"

	ciphertext, err := encrypt(key1, plaintext)
	require.NoError(t, err)

	// Decrypt with wrong key should fail
	_, err = decrypt(key2, ciphertext)
	require.Error(t, err)
}

func TestDecryptInvalidBase64(t *testing.T) {
	key := []byte("0123456789abcdef")
	_, err := decrypt(key, "not-valid-base64!!!")
	require.Error(t, err)
}

func TestDecryptTooShort(t *testing.T) {
	key := []byte("0123456789abcdef")
	// Base64 of 11 bytes (less than GCM nonce size of 12)
	// "shortbase64" = MTIzNDU2Nzg5MA == 11 bytes
	_, err := decrypt(key, "MTIzNDU2Nzg5MA==")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := "original"
	ciphertext, err := encrypt(key, plaintext)
	require.NoError(t, err)

	// Tamper with ciphertext by flipping a character
	tampered := ciphertext[:len(ciphertext)-2] + "XX"
	_, err = decrypt(key, tampered)
	require.Error(t, err)
}
