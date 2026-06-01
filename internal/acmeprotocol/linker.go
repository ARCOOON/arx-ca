package acmeprotocol

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/smallstep/certificates/acme"
	"github.com/smallstep/certificates/api/render"
	"github.com/smallstep/certificates/authority"
	"github.com/smallstep/certificates/authority/provisioner"
)

const mountPrefix = "/acme"

// FlatLinker generates RFC 8555 resource URLs under /acme without a provisioner path segment.
type FlatLinker struct {
	dns             string
	provisionerName string
	auth            *authority.Authority
}

// NewFlatLinker constructs a linker for the public ACME directory layout.
func NewFlatLinker(dns, provisionerName string, auth *authority.Authority) *FlatLinker {
	return &FlatLinker{
		dns:             normalizeDNS(dns),
		provisionerName: provisionerName,
		auth:            auth,
	}
}

func normalizeDNS(dns string) string {
	_, _, err := net.SplitHostPort(dns)
	if err != nil && strings.Contains(err.Error(), "too many colons in address") {
		lastIndex := strings.LastIndex(dns, ":")
		hostPart, portPart := dns[:lastIndex], dns[lastIndex+1:]
		if ip := net.ParseIP(hostPart); ip != nil {
			return "[" + hostPart + "]:" + portPart
		}
		if ip := net.ParseIP(dns); ip != nil {
			return "[" + dns + "]"
		}
	}
	return dns
}

func flatPathSuffix(typ acme.LinkType, inputs ...string) string {
	switch typ {
	case acme.NewNonceLinkType:
		return "/new-nonce"
	case acme.DirectoryLinkType:
		return "/directory"
	case acme.NewAccountLinkType:
		return "/new-account"
	case acme.NewOrderLinkType:
		return "/new-order"
	case acme.RevokeCertLinkType:
		return "/revoke-cert"
	case acme.KeyChangeLinkType:
		return "/key-change"
	case acme.AccountLinkType:
		return "/account/" + inputs[0]
	case acme.OrderLinkType:
		return "/order/" + inputs[0]
	case acme.FinalizeLinkType:
		return "/order/" + inputs[0] + "/finalize"
	case acme.AuthzLinkType:
		return "/authz/" + inputs[0]
	case acme.ChallengeLinkType:
		return "/challenge/" + inputs[0] + "/" + inputs[1]
	case acme.CertificateLinkType:
		return "/certificate/" + inputs[0]
	case acme.OrdersByAccountLinkType:
		return "/account/" + inputs[0] + "/orders"
	default:
		return ""
	}
}

// GetLink builds an absolute ACME resource URL.
func (l *FlatLinker) GetLink(ctx context.Context, typ acme.LinkType, inputs ...string) string {
	var u url.URL
	if baseURL := baseURLFromContext(ctx); baseURL != nil {
		u = *baseURL
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Host == "" {
		u.Host = l.dns
	}
	u.Path = mountPrefix + flatPathSuffix(typ, inputs...)
	return u.String()
}

// Middleware injects the ACME provisioner and request base URL into the context.
func (l *FlatLinker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := newBaseURLContext(r.Context(), r)

		prov, err := l.auth.LoadProvisionerByName(l.provisionerName)
		if err != nil {
			render.Error(w, r, err)
			return
		}

		acmeProv, ok := prov.(*provisioner.ACME)
		if !ok {
			render.Error(w, r, acme.NewDetailedError(acme.ErrorUnauthorizedType, "provisioner must be of type ACME"))
			return
		}

		ctx = acme.NewProvisionerContext(ctx, acme.Provisioner(acmeProv))
		ctx = acme.NewLinkerContext(ctx, l)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LinkOrder sets order resource URLs on the order object.
func (l *FlatLinker) LinkOrder(ctx context.Context, o *acme.Order) {
	o.AuthorizationURLs = make([]string, len(o.AuthorizationIDs))
	for i, azID := range o.AuthorizationIDs {
		o.AuthorizationURLs[i] = l.GetLink(ctx, acme.AuthzLinkType, azID)
	}
	o.FinalizeURL = l.GetLink(ctx, acme.FinalizeLinkType, o.ID)
	if o.CertificateID != "" {
		o.CertificateURL = l.GetLink(ctx, acme.CertificateLinkType, o.CertificateID)
	}
}

// LinkAccount sets account resource URLs.
func (l *FlatLinker) LinkAccount(ctx context.Context, acc *acme.Account) {
	acc.OrdersURL = l.GetLink(ctx, acme.OrdersByAccountLinkType, acc.ID)
}

// LinkChallenge sets the challenge resource URL.
func (l *FlatLinker) LinkChallenge(ctx context.Context, ch *acme.Challenge, azID string) {
	ch.URL = l.GetLink(ctx, acme.ChallengeLinkType, azID, ch.ID)
}

// LinkAuthorization links challenges on an authorization.
func (l *FlatLinker) LinkAuthorization(ctx context.Context, az *acme.Authorization) {
	for _, ch := range az.Challenges {
		l.LinkChallenge(ctx, ch, az.ID)
	}
}

// LinkOrdersByAccountID converts order IDs to resource URLs.
func (l *FlatLinker) LinkOrdersByAccountID(ctx context.Context, orders []string) {
	for i, id := range orders {
		orders[i] = l.GetLink(ctx, acme.OrderLinkType, id)
	}
}

type baseURLKey struct{}

func newBaseURLContext(ctx context.Context, r *http.Request) context.Context {
	var u *url.URL
	if r.Host != "" {
		u = &url.URL{Scheme: "https", Host: r.Host}
	}
	return context.WithValue(ctx, baseURLKey{}, u)
}

func baseURLFromContext(ctx context.Context) *url.URL {
	if u, ok := ctx.Value(baseURLKey{}).(*url.URL); ok {
		return u
	}
	return nil
}
