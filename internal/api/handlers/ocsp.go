package handlers

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/ca"
)

const maxOCSPBodyBytes = 64 << 10 // 64 KiB

// OCSPHandler serves RFC 6960 OCSP responder endpoints.
type OCSPHandler struct {
	engine *ca.PKIEngine
}

// NewOCSPHandler constructs an OCSP handler bound to the PKI engine.
func NewOCSPHandler(engine *ca.PKIEngine) *OCSPHandler {
	return &OCSPHandler{engine: engine}
}

// Post handles POST /ocsp with an application/ocsp-request body.
func (h *OCSPHandler) Post() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeOCSPError(w, http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxOCSPBodyBytes))
		if err != nil {
			writeOCSPError(w, http.StatusBadRequest)
			return
		}

		h.serveOCSP(w, r, body)
	})
}

// Get handles GET /ocsp/{base64url} where the path segment is a URL-safe base64 OCSP request.
func (h *OCSPHandler) Get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeOCSPError(w, http.StatusMethodNotAllowed)
			return
		}

		encoded := strings.TrimSpace(r.PathValue("request"))
		if encoded == "" {
			writeOCSPError(w, http.StatusBadRequest)
			return
		}

		requestDER, err := decodeOCSPPath(encoded)
		if err != nil {
			writeOCSPError(w, http.StatusBadRequest)
			return
		}

		h.serveOCSP(w, r, requestDER)
	})
}

func (h *OCSPHandler) serveOCSP(w http.ResponseWriter, r *http.Request, requestDER []byte) {
	respDER, err := h.engine.RespondOCSP(r.Context(), requestDER)
	if err != nil {
		log.Printf("ocsp: respond: %v", err)
		writeOCSPError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/ocsp-response")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(respDER); err != nil {
		log.Printf("ocsp: write response: %v", err)
	}
}

func decodeOCSPPath(encoded string) ([]byte, error) {
	if raw, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
		return raw, nil
	}
	if raw, err := base64.URLEncoding.DecodeString(encoded); err == nil {
		return raw, nil
	}
	return base64.StdEncoding.DecodeString(encoded)
}

func writeOCSPError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/ocsp-response")
	w.WriteHeader(status)
}
