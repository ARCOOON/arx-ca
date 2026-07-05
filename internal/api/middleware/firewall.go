package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

// FirewallRuleSet holds the active IP allowlist evaluated by the firewall middleware.
type FirewallRuleSet struct {
	Enabled   bool
	Networks  []*net.IPNet
	Addresses []net.IP
}

// Firewall holds a concurrently swappable allowlist for runtime hot-reload.
type Firewall struct {
	mu    sync.RWMutex
	rules FirewallRuleSet
}

// NewFirewall constructs a firewall with an empty disabled ruleset.
func NewFirewall() *Firewall {
	return &Firewall{}
}

// Update replaces the active allowlist atomically.
func (f *Firewall) Update(rules FirewallRuleSet) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.rules = rules
	f.mu.Unlock()
}

// Rules returns a snapshot of the current ruleset.
func (f *Firewall) Rules() FirewallRuleSet {
	if f == nil {
		return FirewallRuleSet{}
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.rules
}

// FirewallFromAllowlist parses CIDR strings and single IP addresses into a ruleset.
func FirewallFromAllowlist(enabled bool, entries []string) (FirewallRuleSet, error) {
	rules := FirewallRuleSet{Enabled: enabled}
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return FirewallRuleSet{}, err
			}
			rules.Networks = append(rules.Networks, network)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			return FirewallRuleSet{}, &net.AddrError{Err: "invalid IP or CIDR", Addr: entry}
		}
		rules.Addresses = append(rules.Addresses, ip)
	}
	return rules, nil
}

// IPAllowed reports whether ip matches the ruleset. When disabled, all IPs are allowed.
func (r FirewallRuleSet) IPAllowed(ip net.IP) bool {
	if !r.Enabled {
		return true
	}
	if ip == nil {
		return false
	}
	for _, allowed := range r.Addresses {
		if allowed.Equal(ip) {
			return true
		}
	}
	for _, network := range r.Networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// FirewallMiddleware blocks requests when the firewall is enabled and the client IP is not allowlisted.
func FirewallMiddleware(firewall *Firewall, next http.Handler) http.Handler {
	if firewall == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rules := firewall.Rules()
		if !rules.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ip := net.ParseIP(clientIP(r))
		if rules.IPAllowed(ip) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"client IP is not allowlisted","data":null}`))
	})
}
