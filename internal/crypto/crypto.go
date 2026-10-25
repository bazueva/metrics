package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func Encrypt(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("generate AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	encryptedKey, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		aesKey,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("encrypt AES key: %w", err)
	}

	if len(encryptedKey) > 65535 {
		return nil, fmt.Errorf("encrypted key too large")
	}

	result := make([]byte, 2, 2+len(encryptedKey)+len(nonce)+len(ciphertext))
	binary.BigEndian.PutUint16(result, uint16(len(encryptedKey)))

	result = append(result, encryptedKey...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func Decrypt(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("invalid encrypted data")
	}

	keyLength := int(binary.BigEndian.Uint16(data[:2]))

	if keyLength == 0 || len(data) < 2+keyLength {
		return nil, fmt.Errorf("invalid encrypted key length")
	}

	encryptedKey := data[2 : 2+keyLength]
	payload := data[2+keyLength:]

	aesKey, err := rsa.DecryptOAEP(
		sha256.New(),
		nil,
		privateKey,
		encryptedKey,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypt AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(payload) < nonceSize+gcm.Overhead() {
		return nil, fmt.Errorf("invalid encrypted payload")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	return plaintext, nil
}
