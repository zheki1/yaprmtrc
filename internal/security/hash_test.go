package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
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

func TestEncryptDecryptHybrid(t *testing.T) {
	// Generate test keys
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := &privKey.PublicKey

	// Test data
	data := []byte("test data for encryption")

	// Encrypt
	encrypted, err := EncryptHybrid(data, pubKey)
	if err != nil {
		t.Fatal(err)
	}

	// Decrypt
	decrypted, err := DecryptHybrid(encrypted, privKey)
	if err != nil {
		t.Fatal(err)
	}

	// Check
	if string(decrypted) != string(data) {
		t.Errorf("Decrypted data does not match original: got %s, want %s", decrypted, data)
	}
}

func TestLoadKeys(t *testing.T) {
	// Generate test keys
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	// Encode private key
	privDER := x509.MarshalPKCS1PrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})

	// Encode public key
	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	// Write to temp files
	privFile, err := os.CreateTemp("", "privkey.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(privFile.Name())
	privFile.Write(privPEM)
	privFile.Close()

	pubFile, err := os.CreateTemp("", "pubkey.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pubFile.Name())
	pubFile.Write(pubPEM)
	pubFile.Close()

	// Load keys
	loadedPub, err := LoadPublicKey(pubFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	loadedPriv, err := LoadPrivateKey(privFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Test encryption/decryption with loaded keys
	data := []byte("test data")
	encrypted, err := EncryptHybrid(data, loadedPub)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := DecryptHybrid(encrypted, loadedPriv)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(data) {
		t.Errorf("Decrypted data does not match original")
	}
}
