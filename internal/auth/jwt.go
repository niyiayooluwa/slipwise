package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenTTL is the lifetime of an issued access token. It's kept short
// (15 minutes) because access tokens are not revocable. A compromised
// token is only ever valid for this window. Long-lived sessions are handled
// via refresh tokens instead (see tokens.go).
const AccessTokenTTL = 15 * time.Minute

// ErrInvalidToken is returned by Verify when a token is malformed, expired,
// or signed with an unexpected algorithm. Callers should treat this as a
// 401, not a 500 — it does not distinguish between "expired" and "forged"
// on purpose, to avoid leaking which case applies to an attacker.
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims is the JWT payload embedded in every access token issued by
// JWTIssuer. It carries the authenticated user's ID alongside the standard
// registered claims (exp, iat).
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTIssuer issues and verifies HS256-signed access tokens using a shared
// secret. It holds no state beyond the signing key, so a single instance
// is safe to reuse (and share) across requests.
type JWTIssuer struct {
	secret []byte
}

// NewJWTIssuer creates a JWTIssuer that signs and verifies tokens using secret.
// secret should be loaded from config/env, never hardcoded — anyone with it
// can mint valid tokens for any user ID.
func NewJWTIssuer(secret string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret)}
}

// Issue creates a signed access token for userID, valid for AccessTokenTTL
// from now.
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

// Verify parses and validates tokenStr, returning its Claims if the
// signature and expiry check out. It rejects any token not signed with
// HMAC (SigningMethodHMAC), which guards against algorithm-confusion
// attacks where a forged token claims a different signing method (e.g. "none").
// Any failure — bad signature, expired, wrong method — collapses to
// ErrInvalidToken.
func (j *JWTIssuer) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
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
