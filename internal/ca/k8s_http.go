package ca

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"
)

func clientWithCA(caPEM []byte) (*http.Client, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errInvalidK8sCA
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}, nil
}

var errInvalidK8sCA = &k8sCAError{msg: "kubernetes cluster CA could not be parsed"}

type k8sCAError struct{ msg string }

func (e *k8sCAError) Error() string { return e.msg }
