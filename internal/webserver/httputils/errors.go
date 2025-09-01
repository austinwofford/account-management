package httputils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
)

type ErrorResponse struct {
	// The error, explained (for humans).
	Message string `json:"message,omitempty"`
	// The type of error (for computers). Document this and keep it stable
	// so that clients can reliably handle it.
	Type string `json:"type,omitempty"`
	// The response status code. This isn't marshaled in the response body
	// since it is in the header. Defaults to 500.
	StatusCode int `json:"-"`
	// The text description of the status code. This does get marshaled in the
	// response body. You do not need to set this as it will be derived from the status code.
	// Defaults to Internal Server Error.
	Status string `json:"http_status"`
	// We find this on the request context and set it so that clients can give us some info
	// when there's an error.
	RequestID string `json:"request_id,omitempty"`
}

// WriteErrorResponse writes a standard error response body. Failures to JSON encode the body
// will be logged and otherwise ignored. Status codes will still be written.
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, httpErr ErrorResponse) {
	if httpErr.StatusCode == 0 {
		httpErr.StatusCode = 500
	}

	httpErr.Status = http.StatusText(httpErr.StatusCode)
	httpErr.RequestID = fmt.Sprint(r.Context().Value(middleware.RequestIDKey))

	if err := json.NewEncoder(w).Encode(httpErr); err != nil {
		ctx := r.Context()
		slog.ErrorContext(ctx, "failed to encode JSON error response", "error", err.Error())
	}

	w.WriteHeader(httpErr.StatusCode)
}
