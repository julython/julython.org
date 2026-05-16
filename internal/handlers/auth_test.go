package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- generateRandomString ---

func TestGenerateRandomString(t *testing.T) {
	t.Run("returns string of expected length", func(t *testing.T) {
		s, err := generateRandomString(16)
		require.NoError(t, err)
		// base64 raw url encoding of 16 bytes = 22 chars
		assert.Len(t, s, 22)
	})

	t.Run("returns string for length 1", func(t *testing.T) {
		s, err := generateRandomString(1)
		require.NoError(t, err)
		// base64 raw url encoding of 1 byte = 2 chars
		assert.Len(t, s, 2)
	})

	t.Run("returns string for length 64", func(t *testing.T) {
		s, err := generateRandomString(64)
		require.NoError(t, err)
		// base64 raw url encoding of 64 bytes = 86 chars
		assert.Len(t, s, 86)
	})

	t.Run("produces unique strings", func(t *testing.T) {
		strings := make(map[string]bool, 20)
		for i := 0; i < 20; i++ {
			s, err := generateRandomString(16)
			require.NoError(t, err)
			require.False(t, strings[s], "duplicate random string at iteration %d", i)
			strings[s] = true
		}
	})
}

func TestGenerateRandomStringZeroLength(t *testing.T) {
	s, err := generateRandomString(0)
	require.NoError(t, err)
	assert.Equal(t, "", s)
}

func TestGenerateRandomStringOutputFormat(t *testing.T) {
	// Verify the output uses URL-safe base64 characters (no padding)
	s, err := generateRandomString(32)
	require.NoError(t, err)
	for _, ch := range s {
		assert.True(t,
			(ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
				(ch >= '0' && ch <= '9') || ch == '-' || ch == '_',
			"unexpected character in base64url output: %q", ch,
		)
	}
	// No padding
	assert.NotContains(t, s, "=")
}
