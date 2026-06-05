package handlers

import (
	"log"
	"net/http"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// CAHandler serves public CA certificate endpoints.
type CAHandler struct {
	engine *ca.PKIEngine
}

// NewCAHandler constructs a CA handler bound to the PKI engine.
func NewCAHandler(engine *ca.PKIEngine) *CAHandler {
	return &CAHandler{engine: engine}
}

// RootCert handles GET /api/v1/ca/root and returns the Root CA certificate in PEM format.
func (h *CAHandler) RootCert() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		pemBytes := h.engine.RootCertPEM()
		if len(pemBytes) == 0 {
			api.WriteError(w, http.StatusInternalServerError, "root certificate is unavailable")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.RootCertResponse{
			PEM: string(pemBytes),
		})
	})
}

// Info handles GET /api/v1/ca/info and returns parsed Root and Intermediate CA metadata.
func (h *CAHandler) Info() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		info, err := h.engine.CAInfo()
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, "CA certificate information is unavailable")
			return
		}

		api.WriteSuccess(w, http.StatusOK, info)
	})
}

// Chain handles GET /api/v1/ca/chain and returns a ZIP archive with Root, Intermediate, and chain PEM/CRT files.
func (h *CAHandler) Chain() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		zipBytes, err := buildCABundleZip(caBundleInput{
			IntermediatePEM: string(h.engine.IntermediateCertPEM()),
			RootPEM:         string(h.engine.RootCertPEM()),
		})
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, "CA bundle is unavailable")
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="ca-bundle.zip"`)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(zipBytes); err != nil {
			log.Printf("ca: write chain bundle: %v", err)
		}
	})
}

// Provisioners handles GET /api/v1/ca/provisioners and returns sanitized provisioners from ca.json.
func (h *CAHandler) Provisioners() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp, err := h.engine.CAProvisioners()
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, "CA provisioner configuration is unavailable")
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}
