package acmeprotocol

import stepacme "github.com/smallstep/certificates/acme"

// StatusProcessing is the ACME challenge status while validation is in progress (RFC 8555 §7.1.6).
const StatusProcessing = stepacme.Status("processing")

func rootedName(name string) string {
	if stepacme.StrictFQDN {
		if name == "" || name[len(name)-1] != '.' {
			return name + "."
		}
	}
	return name
}
