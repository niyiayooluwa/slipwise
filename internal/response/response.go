// Package response holds the shared HTTP response helpers every
// domain's handler package uses (WriteJSON/WriteError). It is
// deliberately a leaf package — it imports apitypes and nothing else
// project-internal, and nothing project-internal should ever import a
// handler package from here. This keeps the dependency direction
// clean: handler packages import response; response never imports a
// handler package. (It used to live inside internal/httpserver, which
// created a cycle: httpserver's router imports auth/handler to mount
// routes, and auth/handler imported httpserver for these helpers.)
package response

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

// WriteError writes an apitypes.ErrorResponse with the given status
// code. Every @Failure across every handler in the codebase should
// point at apitypes.ErrorResponse, and every handler should produce
// its errors through this function, so that promise stays true.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, apitypes.ErrorResponse{Error: msg})
}
