package security

import (
	"testing"
)

func BenchmarkCalcHash(b *testing.B) {
	body := []byte(`{"id":"Alloc","type":"gauge","value":123.456}`)
	key := "secret-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalcHash(body, key)
	}
}

func BenchmarkCalcHash_LargeBody(b *testing.B) {
	body := make([]byte, 4096)
	for i := range body {
		body[i] = byte(i % 256)
	}
	key := "secret-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalcHash(body, key)
	}
}
