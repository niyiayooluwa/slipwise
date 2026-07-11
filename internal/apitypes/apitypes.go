// Package apitypes holds response shapes that are common across every
// domain (auth, realtime, notifications, ...) rather than specific to
// any one of them. If a type belongs on the wire for exactly one
// domain, it goes in that domain's model package instead — this
// package is only for genuinely cross-cutting shapes.
package apitypes

// ErrorResponse is the shape of every non-2xx JSON body returned by
// this API, regardless of which domain's handler produced it. Every
// @Failure annotation across the codebase should reference this type
// so error shapes are predictable for frontend/QA.
type ErrorResponse struct {
	// Error is a short, user-displayable reason the request failed.
	Error string `json:"error" example:"invalid email or password"`
} // @name ErrorResponse

// MessageResponse is a generic human-readable confirmation, used
// wherever a handler has nothing structured to return beyond
// "it worked."
type MessageResponse struct {
	// Message is a human-readable confirmation for the frontend to display.
	Message string `json:"message" example:"logged out"`
} // @name MessageResponse
