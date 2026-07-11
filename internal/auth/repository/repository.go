// Package repository is the auth domain's only point of contact with
// the database. It wraps the sqlc-generated *db.Queries and exposes
// methods in terms the service layer cares about, translating
// pgx/sql-level errors (e.g. pgx.ErrNoRows) into the sentinel errors
// in errors.go. Nothing above this layer should import "internal/db"
// or know that Postgres/sqlc/pgx exist.
package repository

import (
	"context"
	"errors"
	db "sportloga/internal/db/generated"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AuthRepository is the persistence contract the auth service depends
// on. Defined as an interface so the service can be tested against a
// fake without spinning up Postgres.
type AuthRepository interface {
	CreateUser(ctx context.Context, firstName, lastName, email, passwordHash string) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error

	CreateOTP(ctx context.Context, email, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error)
	GetLatestOTP(ctx context.Context, email, purpose string) (db.OtpCode, error)
	IncrementOTPAttempts(ctx context.Context, otpID uuid.UUID) error
	MarkOTPUsed(ctx context.Context, otpID uuid.UUID) error

	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
}

// repo is the concrete AuthRepository backed by sqlc/pgx.
type repo struct {
	q *db.Queries
}

// NewAuthRepository builds an AuthRepository over the given generated
// Queries. q is expected to already be bound to a live pgx pool/conn.
func NewAuthRepository(q *db.Queries) AuthRepository {
	return &repo{q: q}
}

func (r *repo) CreateUser(ctx context.Context, firstName, lastName, email, passwordHash string) (db.User, error) {
	u, err := r.q.CreateUser(ctx, db.CreateUserParams{
		FirstName:    &firstName,
		LastName:     &lastName,
		Email:        email,
		PasswordHash: passwordHash,
	})
	return u, wrap(err)
}

func (r *repo) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	return u, wrap(err)
}

func (r *repo) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	return u, wrap(err)
}

func (r *repo) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return wrap(r.q.MarkEmailVerified(ctx, userID))
}

func (r *repo) CreateOTP(ctx context.Context, email, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
	o, err := r.q.CreateOTP(ctx, db.CreateOTPParams{
		Email:     email,
		CodeHash:  codeHash,
		Purpose:   purpose,
		ExpiresAt: expiresAt,
	})
	return o, wrap(err)
}

func (r *repo) GetLatestOTP(ctx context.Context, email, purpose string) (db.OtpCode, error) {
	o, err := r.q.GetLatestOTP(ctx, db.GetLatestOTPParams{Email: email, Purpose: purpose})
	return o, wrap(err)
}

func (r *repo) IncrementOTPAttempts(ctx context.Context, otpID uuid.UUID) error {
	return wrap(r.q.IncrementOTPAttempts(ctx, otpID))
}

func (r *repo) MarkOTPUsed(ctx context.Context, otpID uuid.UUID) error {
	return wrap(r.q.MarkOTPUsed(ctx, otpID))
}

func (r *repo) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	t, err := r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	return t, wrap(err)
}

func (r *repo) GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	t, err := r.q.GetRefreshToken(ctx, tokenHash)
	return t, wrap(err)
}

func (r *repo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	return wrap(r.q.RevokeRefreshToken(ctx, id))
}

// ErrNotFound is returned when a row doesn't exist — the service layer
// checks against this rather than against pgx.ErrNoRows directly, so it
// never needs to import pgx.
var ErrNotFound = errors.New("not found")

// wrap normalizes pgx.ErrNoRows into the package's own ErrNotFound and
// passes every other error through unchanged. Keeps pgx out of every
// layer above this one.
func wrap(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
