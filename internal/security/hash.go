// Package security содержит функции для подписи и проверки целостности данных.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"os"
)

// CalcHash вычисляет SHA-256 хеш тела body с применением ключа key.
// Возвращает hex-кодированную строку.
func CalcHash(body []byte, key string) string {
	h := sha256.New()
	h.Write(body)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

// LoadPublicKey загружает публичный RSA ключ из файла.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if rsaPub, ok := pub.(*rsa.PublicKey); ok {
		return rsaPub, nil
	}
	return nil, errors.New("not RSA public key")
}

// LoadPrivateKey загружает приватный RSA ключ из файла.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid private key")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

// EncryptHybrid шифрует данные гибридно: AES для данных, RSA для ключа.
func EncryptHybrid(data []byte, pub *rsa.PublicKey) ([]byte, error) {
	// Генерируем случайный AES ключ
	aesKey := make([]byte, 32) // 256 бит
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, err
	}

	// Шифруем данные AES
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Шифруем AES ключ RSA
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, err
	}

	// Возвращаем encryptedKey + ciphertext
	result := make([]byte, len(encryptedKey)+len(ciphertext))
	copy(result, encryptedKey)
	copy(result[len(encryptedKey):], ciphertext)
	return result, nil
}

// DecryptHybrid расшифровывает данные гибридно.
func DecryptHybrid(data []byte, priv *rsa.PrivateKey) ([]byte, error) {
	// Предполагаем, что первые 256 байт - зашифрованный ключ (для RSA 2048)
	keySize := priv.Size()
	if len(data) < keySize {
		return nil, errors.New("data too short")
	}
	encryptedKey := data[:keySize]
	ciphertext := data[keySize:]

	// Расшифровываем AES ключ
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encryptedKey, nil)
	if err != nil {
		return nil, err
	}

	// Расшифровываем данные AES
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
