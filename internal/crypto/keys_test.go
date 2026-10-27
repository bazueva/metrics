package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPublicKey(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		data, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)

		path := writePEMFile(t, "PUBLIC KEY", data)

		publicKey, err := LoadPublicKey(path)
		require.NoError(t, err)
		require.NotNil(t, publicKey)

		assert.Equal(t, privateKey.PublicKey.N, publicKey.N)
		assert.Equal(t, privateKey.PublicKey.E, publicKey.E)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "missing.pem")

		publicKey, err := LoadPublicKey(path)

		require.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, publicKey)
	})

	t.Run("invalid PEM", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "public.pem")
		require.NoError(t, os.WriteFile(path, []byte("invalid PEM"), 0600))

		publicKey, err := LoadPublicKey(path)

		require.EqualError(t, err, "invalid public key PEM")
		assert.Nil(t, publicKey)
	})

	t.Run("invalid public key", func(t *testing.T) {
		t.Parallel()

		path := writePEMFile(t, "PUBLIC KEY", []byte("invalid key"))

		publicKey, err := LoadPublicKey(path)

		require.ErrorContains(t, err, "parse public key:")
		assert.Nil(t, publicKey)
	})

	t.Run("public key is not RSA", func(t *testing.T) {
		t.Parallel()

		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		data, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)

		path := writePEMFile(t, "PUBLIC KEY", data)

		publicKey, err := LoadPublicKey(path)

		require.EqualError(t, err, "public key is not RSA")
		assert.Nil(t, publicKey)
	})
}

func TestLoadPrivateKey(t *testing.T) {
	t.Parallel()

	t.Run("success PKCS1", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		data := x509.MarshalPKCS1PrivateKey(privateKey)
		path := writePEMFile(t, "RSA PRIVATE KEY", data)

		key, err := LoadPrivateKey(path)
		require.NoError(t, err)
		require.NotNil(t, key)

		assert.Equal(t, privateKey.N, key.N)
		assert.Equal(t, privateKey.D, key.D)
	})

	t.Run("success PKCS8", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		data, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)

		path := writePEMFile(t, "PRIVATE KEY", data)

		key, err := LoadPrivateKey(path)
		require.NoError(t, err)
		require.NotNil(t, key)

		assert.Equal(t, privateKey.N, key.N)
		assert.Equal(t, privateKey.D, key.D)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "missing.pem")

		key, err := LoadPrivateKey(path)

		require.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, key)
	})

	t.Run("invalid PEM", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "private.pem")
		require.NoError(t, os.WriteFile(path, []byte("invalid PEM"), 0600))

		key, err := LoadPrivateKey(path)

		require.EqualError(t, err, "invalid private key PEM")
		assert.Nil(t, key)
	})

	t.Run("invalid PKCS1", func(t *testing.T) {
		t.Parallel()

		path := writePEMFile(t, "RSA PRIVATE KEY", []byte("invalid key"))

		key, err := LoadPrivateKey(path)

		require.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("invalid PKCS8", func(t *testing.T) {
		t.Parallel()

		path := writePEMFile(t, "PRIVATE KEY", []byte("invalid key"))

		key, err := LoadPrivateKey(path)

		require.ErrorContains(t, err, "parse private key:")
		assert.Nil(t, key)
	})

	t.Run("private key is not RSA", func(t *testing.T) {
		t.Parallel()

		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		data, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)

		path := writePEMFile(t, "PRIVATE KEY", data)

		key, err := LoadPrivateKey(path)

		require.EqualError(t, err, "private key is not RSA")
		assert.Nil(t, key)
	})

	t.Run("unsupported private key type", func(t *testing.T) {
		t.Parallel()

		path := writePEMFile(t, "EC PRIVATE KEY", []byte("test"))

		key, err := LoadPrivateKey(path)

		require.EqualError(t, err, "unsupported private key type: EC PRIVATE KEY")
		assert.Nil(t, key)
	})
}

func writePEMFile(t *testing.T, blockType string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "key.pem")

	content := pem.EncodeToMemory(&pem.Block{
		Type:  blockType,
		Bytes: data,
	})

	require.NoError(t, os.WriteFile(path, content, 0600))

	return path
}
