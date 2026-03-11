package shims

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// SecureRandomHex generates a random hex string.
// n is the number of random bytes (default 16); the hex output is 2*n chars.
func SecureRandomHex(n ...int) string {
	size := 16
	if len(n) > 0 {
		size = n[0]
	}
	b := make([]byte, size)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SecureRandomBytes generates n random bytes.
func SecureRandomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// SecureRandomBase64 generates a random base64 string.
// n is the number of random bytes (default 16).
func SecureRandomBase64(n ...int) string {
	size := 16
	if len(n) > 0 {
		size = n[0]
	}
	b := make([]byte, size)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// SecureRandomURLSafeBase64 generates a random URL-safe base64 string.
// n is the number of random bytes (default 16).
func SecureRandomURLSafeBase64(n ...int) string {
	size := 16
	if len(n) > 0 {
		size = n[0]
	}
	b := make([]byte, size)
	rand.Read(b)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)
}

// SecureRandomUUID generates a random v4 UUID string.
func SecureRandomUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// SecureRandomNumber generates a random number.
// With no args: returns a float in [0, 1).
// With an int arg: returns an int in [0, n).
// With a float arg: returns a float in [0, n).
func SecureRandomNumber(n ...int) int {
	if len(n) == 0 {
		return 0
	}
	max := big.NewInt(int64(n[0]))
	val, _ := rand.Int(rand.Reader, max)
	return int(val.Int64())
}

// SecureRandomAlphanumeric generates a random alphanumeric string.
// n is the length (default 16).
func SecureRandomAlphanumeric(n ...int) string {
	size := 16
	if len(n) > 0 {
		size = n[0]
	}
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var sb strings.Builder
	for i := 0; i < size; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		sb.WriteByte(charset[idx.Int64()])
	}
	return sb.String()
}
