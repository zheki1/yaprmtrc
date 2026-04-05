// Package security содержит функции для подписи и проверки целостности данных.
package security

import (
	"crypto/sha256"
	"encoding/hex"
)

// CalcHash вычисляет SHA-256 хеш тела body с применением ключа key.
// Возвращает hex-кодированную строку.
func CalcHash(body []byte, key string) string {
	h := sha256.New()
	h.Write(body)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
