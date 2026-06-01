package local

import (
	"fmt"
	"strings"
)

// ListOptions filters installed certificate enumeration.
type ListOptions struct {
	Stores []StoreKind
}

// List returns installed certificates from the requested stores.
func List(opts ListOptions) ([]InstalledCertificate, error) {
	stores := opts.Stores
	if len(stores) == 0 {
		stores = []StoreKind{StoreSystem, StoreUser, StoreBrowser}
	}

	var out []InstalledCertificate
	seen := make(map[string]struct{})

	for _, store := range stores {
		certs, err := listStore(store)
		if err != nil {
			return nil, fmt.Errorf("%s store: %w", store, err)
		}
		for _, cert := range certs {
			key := string(cert.Store) + "|" + cert.Thumbprint
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, cert)
		}
	}

	return out, nil
}

// Get returns a single installed certificate by ID (thumbprint or serial).
func Get(id string) (*InstalledCertificate, error) {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return nil, fmt.Errorf("certificate id is required")
	}

	certs, err := List(ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range certs {
		c := &certs[i]
		if strings.EqualFold(c.ID, id) ||
			strings.EqualFold(c.Thumbprint, id) ||
			strings.EqualFold(c.Serial, id) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("certificate %q not found in local stores", id)
}

// ParseStoreKinds converts CLI store names into StoreKind values.
func ParseStoreKinds(names []string) ([]StoreKind, error) {
	if len(names) == 0 {
		return nil, nil
	}
	out := make([]StoreKind, 0, len(names))
	for _, name := range names {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "system":
			out = append(out, StoreSystem)
		case "user":
			out = append(out, StoreUser)
		case "browser":
			out = append(out, StoreBrowser)
		default:
			return nil, fmt.Errorf("unknown store %q (use system, user, or browser)", name)
		}
	}
	return out, nil
}
