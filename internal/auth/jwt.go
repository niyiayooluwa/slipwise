// Package auth implements foundational security primitives:
// - Stateless HS256 JWT access tokens (15-min lifetime)
// - Cryptographically secure refresh tokens with SHA-256 hash lookups
// - Salted bcrypt password & OTP verification
// - Bearer token middleware with Echo request store injection
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenTTL is the lifetime of an issued access token.
//
// Why 15 minutes?
// JWTs are stateless: once signed, they CANNOT be revoked until they expire without introducing
// a centralized token-blocklist in Redis/DB (which defeats the performance benefit of stateless JWTs).
// Keeping the TTL short ensures that if an access token is intercepted, the attacker's window of
// opportunity is tiny. For long sessions, the client silently exchanges its 30-day Refresh Token.
const AccessTokenTTL = 15 * time.Minute

// ErrInvalidToken is returned when a token is malformed, expired, forged, or has an invalid signature.
// Security Note: We intentionally collapse all validation errors (expired, bad signature, wrong algo)
// into a single generic 401 error. This prevents timing attacks and information leakage where an attacker
// probes whether a stolen token is expired vs forged.
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims is the custom payload embedded inside every issued access token.
// It carries the authenticated user's UUID alongside standard RFC 7519 registered claims (exp, iat).
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTIssuer is the stateless token mint. It signs and verifies access tokens using HS256 HMAC.
type JWTIssuer struct {
	secret []byte
}

// NewJWTIssuer creates an issuer with the provided secret.
// Always load the secret from environment variables — never commit secrets to Git!
func NewJWTIssuer(secret string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret)}
}

// Issue generates a cryptographically signed HS256 JWT for the given user ID.
func (j *JWTIssuer) Issue(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

// Verify decodes and validates a raw Bearer token string.
//
// Critical Security Defense (Algorithm Confusion):
// We strictly enforce `t.Method.(*jwt.SigningMethodHMAC)`.
// Without this check, a malicious actor could forge a token with header `{"alg": "none"}` or `"RS256"`
// using our public key as the HMAC secret, bypassing authentication completely.
func (j *JWTIssuer) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// Enforce that the token was signed with HMAC (HS256), rejecting "none" or asymmetric algorithms
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
