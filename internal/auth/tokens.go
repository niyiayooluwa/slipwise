package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// RefreshTokenTTLDays is the lifetime, in days, of an issued refresh token.
const RefreshTokenTTLDays = 30

// GenerateRefreshToken returns a high-entropy random string.
// We hash it with sha256 (not bcrypt) before storing — bcrypt is for
// low-entropy secrets like passwords/OTPs where you need slow hashing
// to resist brute force. A 256-bit random token is already unguessable,
// so a fast, deterministic hash is correct here (and lets us look it up
// by exact match instead of comparing against every row).
func GenerateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken deterministically hashes raw with sha256 for storage/lookup.
// Deterministic is the point here: it lets us find a refresh token by
// exact hash match on login instead of scanning every stored hash. Do
// not reuse this for passwords or OTPs — those need bcrypt (slow,
// salted) since they're low-entropy and guessable by brute force.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// GenerateOTP returns a 6-digit numeric code as a string, zero-padded.
// OTPs are intentionally low-entropy (1 in a million) for usability, so
// callers must enforce rate limiting and expiry on verification — this
// function only generates the code, it doesn't protect against brute force.
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
