package security

import (
	"crypto/sha256"
	"encoding/hex"
)

func CalcHash(data []byte, key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
