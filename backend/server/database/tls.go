package dbops

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
)

// Returns tls.Config structure based on the specified connection parameters.
// This implementation origins from the similar logic from lib/pq.
// See: https://github.com/lib/pq/blob/master/ssl.go.
// It strives to establish secure connections over the go-pg in the same
// way as lib/pq package because this package used by the session manager
// (github.com/alexedwards/scs/postgresstore). Note that the lib/pq was
// based on the libpq - C library.
func GetTLSConfig(sslMode, host, sslCert, sslKey, sslRootCert string) (*tls.Config, error) {
	verifyCAOnly := false
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	switch sslMode {
	case "require":
		// We must skip TLS's own verification since it requires full
		// verification since Go 1.3.
		tlsConfig.InsecureSkipVerify = true

		// See note in: http://www.postgresql.org/docs/current/static/libpq-ssl.html
		if len(sslRootCert) > 0 {
			if _, err := os.Stat(sslRootCert); err == nil {
				verifyCAOnly = true
			}
		} else {
			sslRootCert = ""
		}

	case "verify-ca":
		// We must skip TLS's own verification since it requires full
		// verification since Go 1.3.
		tlsConfig.InsecureSkipVerify = true
		verifyCAOnly = true

	case "verify-full":
		tlsConfig.ServerName = host

	case "", "disable":
		return nil, nil

	default:
		return nil, pkgerrors.Errorf("unsupported sslmode value %s", sslMode)
	}

	if verifyCAOnly {
		// Run our own verification for verify-ca and require cases.
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				DNSName:       cs.ServerName,
				Intermediates: x509.NewCertPool(),
				Roots:         tlsConfig.RootCAs,
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		}
	}

	err := setClientCertificates(tlsConfig, sslCert, sslKey)
	if err != nil {
		return nil, err
	}

	err = setCertificateAuthority(tlsConfig, sslRootCert)
	if err != nil {
		return nil, err
	}

	// Accept renegotiation requests initiated by the backend.
	//
	// Renegotiation was deprecated then removed from PostgreSQL 9.5, but
	// the default configuration of older versions has it enabled. Redshift
	// also initiates renegotiations and cannot be reconfigured.
	tlsConfig.Renegotiation = tls.RenegotiateFreelyAsClient

	return tlsConfig, nil
}

// Adds the certificate and key settings, or if they aren't set, from the
// .postgresql directory in the user's home directory. The configured
// files must exist and have the correct permissions.
func setClientCertificates(tlsConfig *tls.Config, sslCert, sslKey string) error {
	user, _ := user.Current()

	// In libpq, the client certificate is only loaded if the setting is not blank.
	if len(sslCert) == 0 && user != nil {
		sslCert = filepath.Join(user.HomeDir, ".postgresql", "postgresql.crt")
	}
	if len(sslCert) == 0 {
		return nil
	}

	if _, err := os.Stat(sslCert); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return pkgerrors.Wrapf(err, "failed to stat the certificate file %s", sslCert)
	}

	// In libpq, the ssl key is only loaded if the setting is not blank.
	if len(sslKey) == 0 && user != nil {
		sslKey = filepath.Join(user.HomeDir, ".postgresql", "postgresql.key")
	}
	if len(sslKey) > 0 {
		sslKeyInfo, err := os.Stat(sslKey)
		if err != nil {
			return pkgerrors.Wrapf(err, "failed to state the key file %s", sslKey)
		}
		if sslKeyInfo.Mode().Perm()&0077 != 0 {
			return pkgerrors.Errorf("key file %s has too large permissions", sslKey)
		}
	}

	cert, err := tls.LoadX509KeyPair(sslCert, sslKey)
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	tlsConfig.Certificates = []tls.Certificate{cert}
	return nil
}

// Adds the RootCA from the specified file.
func setCertificateAuthority(tlsConfig *tls.Config, sslRootCert string) error {
	if len(sslRootCert) > 0 {
		tlsConfig.RootCAs = x509.NewCertPool()

		rootCert, err := ioutil.ReadFile(sslRootCert)
		if err != nil {
			return pkgerrors.Wrapf(err, "failed to read root CA certificate file %s", sslRootCert)
		}

		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCert) {
			return pkgerrors.Wrapf(err, "unable to parse root CA certificate %s", sslRootCert)
		}
	}
	return nil
}
