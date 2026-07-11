package auth

import "golang.org/x/crypto/bcrypt"

// HashSecret hashes secret using bcrypt at the default cost. Used for
// passwords and OTP codes as both are low-entropy, human-facing secrets,
// so we want bcrypt's slow, salted hash to resist brute force. This is
// deliberately different from HashToken in tokens.go, which uses a fast
// deterministic hash for high-entropy refresh tokens.
func HashSecret(secret string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return string(b), err
}

// CheckSecret reports whether secret matches the given bcrypt hash.
// It returns false for any error (malformed hash, mismatch, etc.) —
// callers should treat every false as "invalid credentials" and not
// distinguish reasons, to avoid leaking info to an attacker.
func CheckSecret(hash, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}
