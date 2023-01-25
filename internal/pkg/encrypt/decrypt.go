package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

func Decrypt(encrypted string, salt string) (string, error) {
	encryptionKey, err := DeriveEncryptionKey(salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	cipherText, err := base64.RawStdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	decryptedPassword, err := aesgcm.Open(nil, encryptionKey[:12], cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(decryptedPassword), nil
}
