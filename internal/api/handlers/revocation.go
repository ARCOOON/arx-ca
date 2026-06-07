package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/ARCOOON/arx-ca/internal/ca"
)

// CRL handles GET /api/v1/ca/crl and returns the current CRL in DER or PEM format.
func (h *CAHandler) CRL() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		crlInfo, err := h.engine.GetCRL(r.Context())
		if err != nil {
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("crl: get: %v", err)
				message = "certificate revocation list is unavailable"
			}
			http.Error(w, message, status)
			return
		}

		expires := crlInfo.ExpiresAt
		if expires.IsZero() {
			expires = time.Now().UTC()
		}
		w.Header().Set("Expires", expires.Format(time.RFC1123))
		w.Header().Set("Cache-Control", "public, max-age=3600")

		_, formatAsPEM := r.URL.Query()["pem"]
		if formatAsPEM {
			w.Header().Set("Content-Type", "application/x-pem-file")
			w.Header().Set("Content-Disposition", `attachment; filename="crl.pem"`)
			if _, err := w.Write(ca.EncodeCRLPEM(crlInfo.Data)); err != nil {
				log.Printf("crl: write pem: %v", err)
			}
			return
		}

		w.Header().Set("Content-Type", "application/pkix-crl")
		w.Header().Set("Content-Disposition", `attachment; filename="crl.crl"`)
		if _, err := w.Write(crlInfo.Data); err != nil {
			log.Printf("crl: write der: %v", err)
		}
	})
}
