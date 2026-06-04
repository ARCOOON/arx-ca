package middleware

import (
	"net/http"
	"strings"
)

// CORSOptions holds Cross-Origin Resource Sharing policy applied by CORS middleware.
type CORSOptions struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORS returns middleware that applies configured CORS headers and handles OPTIONS preflight.
func CORS(opts CORSOptions, next http.Handler) http.Handler {
	origins := append([]string(nil), opts.AllowedOrigins...)
	methods := expandCORSMethods(opts.AllowedMethods)
	headers := expandCORSHeaders(opts.AllowedHeaders)
	allowMethods := strings.Join(methods, ", ")
	allowHeaders := strings.Join(headers, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && corsOriginAllowed(origins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", corsAllowOriginValue(origins, origin))
			w.Header().Add("Vary", "Origin")
		} else if corsAllowsAnyOrigin(origins) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		if acrh := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers")); acrh != "" && corsHeadersAllowWildcard(opts.AllowedHeaders) {
			w.Header().Set("Access-Control-Allow-Headers", acrh)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		}
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func expandCORSMethods(methods []string) []string {
	if len(methods) == 0 {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if corsSliceContainsWildcard(methods) {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	}
	out := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m == "" || m == "*" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	if len(out) == 0 {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	return out
}

func expandCORSHeaders(headers []string) []string {
	if len(headers) == 0 {
		return []string{"Authorization", "Content-Type", "Accept", "X-API-Key"}
	}
	if corsHeadersAllowWildcard(headers) {
		return []string{"Authorization", "Content-Type", "Accept", "X-API-Key", "*"}
	}
	out := make([]string, 0, len(headers))
	seen := make(map[string]struct{}, len(headers))
	for _, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" || h == "*" {
			continue
		}
		key := strings.ToLower(h)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, h)
	}
	if len(out) == 0 {
		return []string{"Authorization", "Content-Type", "Accept", "X-API-Key"}
	}
	return out
}

func corsSliceContainsWildcard(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "*" {
			return true
		}
	}
	return false
}

func corsHeadersAllowWildcard(headers []string) bool {
	return corsSliceContainsWildcard(headers)
}

func corsAllowsAnyOrigin(origins []string) bool {
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}

func corsOriginAllowed(origins []string, requestOrigin string) bool {
	if corsAllowsAnyOrigin(origins) {
		return true
	}
	for _, o := range origins {
		if strings.EqualFold(strings.TrimSpace(o), requestOrigin) {
			return true
		}
	}
	return false
}

func corsAllowOriginValue(origins []string, requestOrigin string) string {
	if corsAllowsAnyOrigin(origins) {
		return "*"
	}
	return requestOrigin
}
