package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCalculateHMAC(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  string
	}{
		{
			name: "simple data and key",
			data: []byte("test-data"),
			key:  "secret",
		},
		{
			name: "empty data",
			data: []byte(""),
			key:  "secret",
		},
		{
			name: "empty key",
			data: []byte("test-data"),
			key:  "",
		},
		{
			name: "unicode data",
			data: []byte("привет мир"),
			key:  "ключ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hmac.New(sha256.New, []byte(tt.key))
			h.Write(tt.data)
			expected := hex.EncodeToString(h.Sum(nil))

			result := CalculateHMAC(tt.data, tt.key)

			if result != expected {
				t.Errorf(
					"CalculateHMAC() = %s, want %s",
					result,
					expected,
				)
			}
		})
	}
}
