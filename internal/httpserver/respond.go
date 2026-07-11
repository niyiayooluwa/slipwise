// Package httpserver holds cross-domain HTTP plumbing: the chi router
// wiring and shared response helpers. Individual domains (auth,
// realtime, notifications, ...) import WriteJSON/WriteError rather
// than each defining their own, so every handler in the codebase
// writes errors in exactly the shape model.ErrorResponse promises in
// swagger.
package httpserver

import (
	"encoding/json"
	"net/http"

	"sportloga/internal/apitypes"
)

// WriteJSON writes v as a JSON body with the given status code and the
// correct Content-Type header. v is typically one of the DTOs in a
// domain's model package.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a model.ErrorResponse with the given status code.
// Every @Failure across every handler in the codebase should point at
// model.ErrorResponse, and every handler should produce its errors
// through this function, so that promise stays true.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, apitypes.ErrorResponse{Error: msg})
}
