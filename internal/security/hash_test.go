package security

import (
	"testing"
)

func TestCalcHash(t *testing.T) {
	body := []byte("test body")
	key := "test key"

	hash := CalcHash(body, key)

	if len(hash) != 64 {
		t.Errorf("CalcHash() returned hash with length %d; expected 64 characters (SHA-256)", len(hash))
	}

	if hash == "" {
		t.Errorf("CalcHash() returned empty string")
	}
}
