package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/yabeye/gebeta_api_mvp/common/apperrors"
)

// Response is the standard JSON envelope for every API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
}

type ErrorBody struct {
	Message string `json:"message"`
}

// WriteJSON writes a successful response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

// WriteError maps err to an HTTP status via apperrors and writes it.
// The real error is logged server-side; only a safe message goes to
// the client — never leak internal error strings (DB errors, stack
// traces, etc.) to callers.
func WriteError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status := apperrors.ToHTTPStatus(err)

	clientMsg := "something went wrong"
	if status < http.StatusInternalServerError {
		clientMsg = err.Error()
	} else {
		logger.Error("internal server error", "error", err, "path", r.URL.Path)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   &ErrorBody{Message: clientMsg},
	})
}
