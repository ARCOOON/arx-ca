package server

import (
	"net/http"

	"github.com/ARCOOON/arx-ca/internal/ca"
)

// injectForwardedClientCert sets X-Forwarded-Client-Cert when the WebUI listener received a client certificate.
func injectForwardedClientCert(r *http.Request) {
	if r == nil || r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return
	}
	value, err := ca.FormatForwardedClientCert(r.TLS.PeerCertificates[0])
	if err != nil {
		return
	}
	r.Header.Set(ca.HeaderForwardedClientCert, value)
}

// wrapAPIProxyHandler forwards API traffic and propagates presented client certificates.
func wrapAPIProxyHandler(proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			r2 := r.Clone(r.Context())
			injectForwardedClientCert(r2)
			proxy.ServeHTTP(w, r2)
			return
		}
		proxy.ServeHTTP(w, r)
	})
}
