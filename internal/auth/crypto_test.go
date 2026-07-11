package auth_test

import (
	"testing"
	"time"

	"sportloga/internal/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- JWTIssuer ---

func TestJWTIssuer_IssueThenVerify_RoundTrips(t *testing.T) {
	issuer := auth.NewJWTIssuer("test-secret")
	userID := uuid.New()

	token, err := issuer.Issue(userID)
	if err != nil {
		t.Fatalf("expected no error issuing token, got %v", err)
	}

	claims, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("expected no error verifying token, got %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
}

func TestJWTIssuer_Verify_RejectsTamperedToken(t *testing.T) {
	issuer := auth.NewJWTIssuer("test-secret")
	token, _ := issuer.Issue(uuid.New())

	// Flip the last character — cheap way to corrupt the signature
	// without needing to understand JWT's internal structure.
	tampered := token[:len(token)-1] + "x"

	_, err := issuer.Verify(tampered)
	if err == nil {
		t.Fatal("expected an error verifying a tampered token, got none")
	}
}

func TestJWTIssuer_Verify_RejectsWrongSecret(t *testing.T) {
	issuedBy := auth.NewJWTIssuer("secret-a")
	verifiedBy := auth.NewJWTIssuer("secret-b")

	token, _ := issuedBy.Issue(uuid.New())

	// This is the actual security property: a token signed with one
	// secret must never validate against a different one. If this
	// test ever passes when it shouldn't, that's a critical bug, not
	// a cosmetic one.
	_, err := verifiedBy.Verify(token)
	if err == nil {
		t.Fatal("expected an error verifying a token signed with a different secret, got none")
	}
}

func TestJWTIssuer_Verify_RejectsExpiredToken(t *testing.T) {
	secret := "test-secret"
	issuer := auth.NewJWTIssuer(secret)

	// auth.Issue always sets a 15-minute expiry, so we can't produce
	// an expired token through the public API — we have to build one
	// by hand, signed with the same secret, but with ExpiresAt in the
	// past.
	claims := auth.Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-16 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to build test token: %v", err)
	}

	_, err = issuer.Verify(expiredTokenStr)
	if err == nil {
		t.Fatal("expected an error verifying an expired token, got none")
	}
}

// --- password / OTP hashing (bcrypt) ---

func TestHashSecret_CheckSecret_RoundTrips(t *testing.T) {
	hash, err := auth.HashSecret("correct-password")
	if err != nil {
		t.Fatalf("expected no error hashing, got %v", err)
	}
	if !auth.CheckSecret(hash, "correct-password") {
		t.Error("expected CheckSecret to succeed with the correct secret")
	}
}

func TestCheckSecret_RejectsWrongSecret(t *testing.T) {
	hash, _ := auth.HashSecret("correct-password")

	if auth.CheckSecret(hash, "wrong-password") {
		t.Error("expected CheckSecret to fail with an incorrect secret")
	}
}

func TestHashSecret_IsSalted(t *testing.T) {
	// Two hashes of the *same* input should differ — this is what
	// makes bcrypt resistant to rainbow-table attacks. If this ever
	// starts failing, someone accidentally swapped in a non-salted
	// hash function.
	hashA, _ := auth.HashSecret("same-password")
	hashB, _ := auth.HashSecret("same-password")

	if hashA == hashB {
		t.Error("expected two hashes of the same input to differ (salting), got identical hashes")
	}
	// Both must still independently verify against the original input.
	if !auth.CheckSecret(hashA, "same-password") || !auth.CheckSecret(hashB, "same-password") {
		t.Error("expected both salted hashes to still verify against the original input")
	}
}

// --- refresh token generation / hashing ---

func TestGenerateRefreshToken_HashMatchesHashToken(t *testing.T) {
	raw, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// The hash returned alongside raw must be exactly what HashToken
	// would independently compute for that same raw value — this is
	// what repository.GetRefreshToken relies on to look tokens up by
	// hash.
	if got := auth.HashToken(raw); got != hash {
		t.Errorf("HashToken(raw) = %q, want %q (mismatch with what GenerateRefreshToken returned)", got, hash)
	}
}

func TestGenerateRefreshToken_ProducesUniqueTokens(t *testing.T) {
	rawA, _, _ := auth.GenerateRefreshToken()
	rawB, _, _ := auth.GenerateRefreshToken()

	if rawA == rawB {
		t.Error("expected two generated refresh tokens to differ, got identical values")
	}
}

// --- OTP generation ---

func TestGenerateOTP_IsSixDigits(t *testing.T) {
	code, err := auth.GenerateOTP()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6 characters, got %d (%q)", len(code), code)
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			t.Fatalf("expected all digits, got %q", code)
		}
	}
}
