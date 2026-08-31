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

// GenerateRefreshToken generates a 256-bit (32-byte) cryptographically secure random token.
//
// Why SHA-256 instead of Bcrypt here?
// 1. High Entropy: A 256-bit CSPRNG output has 2^256 possibilities. It is mathematically impossible
//    to brute force, so it does not need bcrypt's intentional CPU-throttling work factor.
// 2. Deterministic Hash Lookup: Storing `sha256(raw)` in the DB allows the database to do an O(1)
//    indexed lookup (`WHERE token_hash = $1`). With bcrypt, the salt is non-deterministic,
//    meaning you'd have to scan every user row and run bcrypt.Compare on each one!
func GenerateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken deterministically hashes a raw refresh token using SHA-256.
// NEVER use this function for passwords or OTPs — low entropy strings MUST use bcrypt.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// GenerateOTP generates a cryptographically random 6-digit numeric OTP (e.g. "049182").
//
// Security Warning:
// A 6-digit OTP has only 1,000,000 possibilities (low entropy!). A fast computer can brute-force
// this in milliseconds. Therefore, the service layer MUST enforce:
// 1. Strict 10-minute expiry (TTL)
// 2. Maximum 3 incorrect guess attempts before the OTP is killed
// 3. Rate limiting on the resend endpoint
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
