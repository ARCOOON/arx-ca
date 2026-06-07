package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// StatsHandler serves aggregate inventory metrics for certificates and SSH CAs.
type StatsHandler struct {
	engine   *ca.PKIEngine
	sshStore *db.SSHCertificateStore
}

// NewStatsHandler constructs a StatsHandler.
func NewStatsHandler(engine *ca.PKIEngine, sshStore *db.SSHCertificateStore) *StatsHandler {
	return &StatsHandler{
		engine:   engine,
		sshStore: sshStore,
	}
}

// CertificateStats handles GET /api/v1/certificates/stats.
func (h *StatsHandler) CertificateStats() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		stats, err := h.computeCertificateStats(r.Context())
		if err != nil {
			log.Printf("stats: certificate stats: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to compute certificate statistics")
			return
		}

		api.WriteSuccess(w, http.StatusOK, stats)
	})
}

// SSHStats handles GET /api/v1/ssh/stats.
func (h *StatsHandler) SSHStats() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if h.sshStore == nil {
			api.WriteError(w, http.StatusInternalServerError, "ssh certificate store is unavailable")
			return
		}

		raw, err := h.sshStore.Stats(r.Context())
		if err != nil {
			log.Printf("stats: ssh stats: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to compute SSH statistics")
			return
		}

		api.WriteSuccess(w, http.StatusOK, models.SSHStatsResponse{
			TotalUserCerts: raw.TotalUserCerts,
			TotalHostCerts: raw.TotalHostCerts,
			ActiveNow:      raw.ActiveNow,
		})
	})
}

func (h *StatsHandler) computeCertificateStats(ctx context.Context) (models.CertificateStatsResponse, error) {
	if h.engine == nil {
		return models.CertificateStatsResponse{}, errors.New("CA engine is not initialized")
	}

	list, err := h.engine.ListCertificates(ctx, models.CertificateListFilter{})
	if err != nil {
		return models.CertificateStatsResponse{}, err
	}

	now := time.Now().UTC()
	expiringCutoff := now.Add(30 * 24 * time.Hour)

	var stats models.CertificateStatsResponse
	stats.TotalIssued = list.Total

	for _, cert := range list.Certificates {
		if cert.Revoked {
			stats.TotalRevoked++
			continue
		}
		if cert.NotAfter.After(now) && !cert.NotAfter.After(expiringCutoff) {
			stats.Expiring30d++
		}
	}

	return stats, nil
}
