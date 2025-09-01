package httputils

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSONResponse writes the provided status header and marshals the provided response struct to JSON.
// Failures to JSON encode the body will be logged and otherwise ignored. Status codes will still be written.
func WriteJSONResponse(w http.ResponseWriter, r *http.Request, status int, responseBody any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if responseBody == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		ctx := r.Context()
		slog.ErrorContext(ctx, "failed to encode JSON response", "error", err.Error())
	}
}
