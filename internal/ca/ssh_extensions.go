package ca

import (
	"golang.org/x/crypto/ssh"

	"github.com/smallstep/certificates/authority/provisioner"
)

// sshUserStandardExtensions ensures common interactive SSH user extensions are present.
type sshUserStandardExtensions struct{}

func (sshUserStandardExtensions) Modify(cert *ssh.Certificate, _ provisioner.SignSSHOptions) error {
	if cert == nil || cert.CertType != ssh.UserCert {
		return nil
	}
	if cert.Extensions == nil {
		cert.Extensions = make(map[string]string, 2)
	}
	cert.Extensions["permit-pty"] = ""
	cert.Extensions["permit-port-forwarding"] = ""
	return nil
}
