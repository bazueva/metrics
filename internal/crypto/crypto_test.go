package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		encrypted, err := Encrypt(&privateKey.PublicKey, original)
		require.NoError(t, err)

		assert.NotEqual(t, original, encrypted)

		decrypted, err := Decrypt(privateKey, encrypted)
		require.NoError(t, err)

		assert.Equal(t, original, decrypted)
	})

	t.Run("wrong private key", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		anotherPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		encrypted, err := Encrypt(&privateKey.PublicKey, original)
		require.NoError(t, err)

		decrypted, err := Decrypt(anotherPrivateKey, encrypted)
		require.Error(t, err)

		assert.Nil(t, decrypted)
	})

	t.Run("corrupted encrypted data", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		encrypted, err := Encrypt(&privateKey.PublicKey, original)
		require.NoError(t, err)

		encrypted[len(encrypted)-1] ^= 0xFF

		decrypted, err := Decrypt(privateKey, encrypted)
		require.Error(t, err)

		assert.Nil(t, decrypted)
	})
}
