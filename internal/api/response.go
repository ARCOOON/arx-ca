package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/your-org/arx-ca/internal/models"
)

// WriteJSON encodes payload as JSON with the given HTTP status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("api: failed to encode JSON response: %v", err)
	}
}

// WriteSuccess writes a standardized successful API envelope.
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, models.NewSuccessResponse(data))
}

// WriteError writes a standardized error API envelope.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, models.NewErrorResponse(message))
}
