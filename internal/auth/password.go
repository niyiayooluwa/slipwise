package auth

import "golang.org/x/crypto/bcrypt"

// HashSecret hashes a secret (password or OTP code) using bcrypt at the default cost (10).
//
// Why Bcrypt?
// Bcrypt uses a slow, memory-hard key derivation function with an automatic random 128-bit salt.
// This ensures that rainbow table attacks are impossible, and attackers cannot efficiently use
// GPUs or ASICs to crack guessed passwords.
func HashSecret(secret string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return string(b), err
}

// CheckSecret verifies whether plaintext secret matches a stored bcrypt hash.
//
// Security Note:
// Bcrypt comparisons are constant-time to eliminate side-channel timing attacks.
// We return a simple boolean so handlers don't accidentally leak detailed error messages
// (like "user exists but password wrong" vs "user does not exist") to prospective attackers.
func CheckSecret(hash, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}
