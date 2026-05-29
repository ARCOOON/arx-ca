//go:build windows

package local

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	certStoreProvSystem         = 10
	certSystemStoreLocalMachine = 0x00020000
	certSystemStoreCurrentUser  = 0x00010000
	cryptEncodePKCS7Asn1        = 0x00000001
)

var (
	modCrypt32             = windows.NewLazySystemDLL("crypt32.dll")
	procCertOpenStore      = modCrypt32.NewProc("CertOpenStore")
	procCertCloseStore     = modCrypt32.NewProc("CertCloseStore")
	procCertEnumCertsStore = modCrypt32.NewProc("CertEnumCertificatesInStore")
)

type windowsStore struct {
	kind      StoreKind
	storeName string
	location  uintptr
}

func listStore(kind StoreKind) ([]InstalledCertificate, error) {
	switch kind {
	case StoreSystem:
		return listWindowsStores([]windowsStore{
			{StoreSystem, "ROOT", certSystemStoreLocalMachine},
			{StoreSystem, "CA", certSystemStoreLocalMachine},
			{StoreSystem, "MY", certSystemStoreLocalMachine},
		})
	case StoreUser:
		return listWindowsStores([]windowsStore{
			{StoreUser, "ROOT", certSystemStoreCurrentUser},
			{StoreUser, "CA", certSystemStoreCurrentUser},
			{StoreUser, "MY", certSystemStoreCurrentUser},
		})
	case StoreBrowser:
		// Chromium-based browsers use the Windows certificate store (covered by system/user).
		return listFirefoxCerts()
	default:
		return nil, fmt.Errorf("unsupported store %q", kind)
	}
}

func listWindowsStores(stores []windowsStore) ([]InstalledCertificate, error) {
	var out []InstalledCertificate
	for _, st := range stores {
		certs, err := enumerateCertStore(st)
		if err != nil {
			return nil, err
		}
		out = append(out, certs...)
	}
	return out, nil
}

func enumerateCertStore(st windowsStore) ([]InstalledCertificate, error) {
	namePtr, err := windows.UTF16PtrFromString(st.storeName)
	if err != nil {
		return nil, err
	}

	store, _, err := procCertOpenStore.Call(
		uintptr(certStoreProvSystem),
		0,
		0,
		st.location|uintptr(cryptEncodePKCS7Asn1),
		uintptr(unsafe.Pointer(namePtr)),
	)
	if store == 0 {
		return nil, fmt.Errorf("open store %s: %w", st.storeName, err)
	}
	defer procCertCloseStore.Call(store, 0)

	var out []InstalledCertificate
	var prev uintptr
	for {
		certCtx, _, err := procCertEnumCertsStore.Call(store, prev)
		if certCtx == 0 {
			if err != syscall.Errno(0) && err != nil {
				return nil, fmt.Errorf("enumerate certificates: %w", err)
			}
			break
		}
		prev = certCtx

		cert, parseErr := parseCertContext(certCtx)
		if parseErr != nil {
			continue
		}
		cert.Store = st.kind
		cert.StoreName = st.storeName
		out = append(out, *cert)
	}
	return out, nil
}

type certContext struct {
	EncodingType uint32
	_            uint32
	EncodedCert  *byte
	Length       uint32
}

func parseCertContext(ctx uintptr) (*InstalledCertificate, error) {
	cc := (*certContext)(unsafe.Pointer(ctx))
	if cc.EncodedCert == nil || cc.Length == 0 {
		return nil, fmt.Errorf("empty certificate context")
	}

	raw := unsafe.Slice(cc.EncodedCert, cc.Length)
	cert, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(cert.Raw)
	thumb := hex.EncodeToString(sum[:])

	return &InstalledCertificate{
		ID:         thumb,
		Subject:    cert.Subject.String(),
		Issuer:     cert.Issuer.String(),
		Serial:     cert.SerialNumber.String(),
		Thumbprint: thumb,
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		DNSNames:   append([]string(nil), cert.DNSNames...),
		IsCA:       cert.IsCA,
	}, nil
}
