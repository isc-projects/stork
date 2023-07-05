package agent

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"isc.org/stork/pki"
	storkutil "isc.org/stork/util"
)

// An utility structure to perform actions on the GRPC/TLS certificate files.
type CertStore struct{}

// Constructs a new cert store instance. It has a short lifetime, may
// be constructed on demand.
func NewCertStore() *CertStore {
	return &CertStore{}
}

// Checks if the file content is a valid certificate in a PEM format.
func (*CertStore) isValidCert(content []byte) error {
	_, err := pki.ParseCert(content)
	return err
}

// Checks if the file content is a valid private key in a PEM format.
func (*CertStore) isValidPrivateKey(content []byte) error {
	_, err := pki.ParsePrivateKey(content)
	return err
}

// Checks if the file content is a valid agent token.
func (*CertStore) isValidToken(content []byte) error {
	if len(content) != 0 {
		return errors.New("file cannot be empty")
	}
	return nil
}

// Reads the file content.
func (*CertStore) read(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read the file: %s", path)
	}
	return content, nil
}

// Writes the content to a file. If the file exists, it is overwritten.
// The directory tree is created if needed.
func (s *CertStore) write(path string, content []byte) error {
	if err := s.removeIfExist(path); err != nil {
		return err
	}

	if err := s.createDirectoryTree(path); err != nil {
		return err
	}

	err := os.WriteFile(path, content, 0o600)
	if err != nil {
		return errors.Wrapf(err, "cannot write the file: %s", path)
	}
	return nil
}

// Checks if a given path exists on a filesystem.
// Returns an error if there is a problem with access to the path.
func (*CertStore) isExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, errors.Wrapf(err, "cannot stat the file: %s", path)
	}
}

// Removes a file if it exists.
func (s *CertStore) removeIfExist(path string) error {
	ok, err := s.isExist(path)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err = os.Remove(path); err != nil {
		return errors.Wrapf(err, "cannot remove the file: %s", path)
	}

	return nil
}

// Creates a directory structure to a given file.
func (s *CertStore) createDirectoryTree(path string) error {
	directory := filepath.Dir(path)
	ok, err := s.isExist(directory)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	if err := os.MkdirAll(directory, 0o700); err != nil {
		return errors.Wrapf(err, "cannot create a directory tree: %s", directory)
	}
	return nil
}

// Parses provided address and return either an IP address as a one
// element list or a DNS name as one element list. The arrays are
// returned as then it is easy to pass these returned elements to the
// functions that generates CSR (Certificate Signing Request).
func (*CertStore) resolveAddress(address string) ([]net.IP, []string) {
	ipAddress := net.ParseIP(address)
	if ipAddress != nil {
		return []net.IP{ipAddress}, []string{}
	}
	return []net.IP{}, []string{address}
}

// Reads the content of the agent token file.
// Returns an error if the file is not available or the content is invalid.
func (s *CertStore) ReadToken() ([]byte, error) {
	content, err := s.read(AgentTokenFile)
	if err != nil {
		return nil, err
	}
	if err = s.isValidToken(content); err != nil {
		return nil, err
	}
	return content, nil
}

// Reads and parses the root CA file.
func (s *CertStore) ReadRootCA() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	ca, err := s.read(RootCAFile)
	if err != nil {
		return nil, err
	}

	// Append the client certificates from the CA.
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		err = errors.New("failed to append client root CA certificate")
		return nil, err
	}
	return certPool, nil
}

// Reads and parses the certificate and private key files. Combines them into
// a single TLS certificate.
func (s *CertStore) ReadTLSCert() (*tls.Certificate, error) {
	keyPEM, err := s.read(KeyPEMFile)
	if err != nil {
		return nil, err
	}
	certPEM, err := s.read(CertPEMFile)
	if err != nil {
		return nil, err
	}
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		err = errors.Wrapf(err, "could not setup TLS key pair")
		return nil, err
	}
	return &certificate, nil
}

// Generates a new private key and writes it to the file.
// After that, it removes the certificate, root CA, and agent token files,
// so they must be requested again.
func (s *CertStore) CreateKey() error {
	keyPEM, err := pki.GenKey()
	if err != nil {
		return err
	}
	err = s.write(KeyPEMFile, keyPEM)
	if err != nil {
		return errors.Wrapf(err, "cannot write key file: %s", keyPEM)
	}

	// Invalidate cert, root CA and token.
	for _, path := range []string{CertPEMFile, RootCAFile, AgentTokenFile} {
		if err = s.removeIfExist(path); err != nil {
			return err
		}
	}
	return nil
}

// Generates the CSR (Certificate Signing Request) for a given IP address or
// hostname. Returns CSR serialized to the PEM format, fingerprint of CSR or
// error.
func (s *CertStore) GenerateCSR(agentAddress string) (csrPEM []byte, fingerprint [32]byte, err error) {
	agentIPs, agentNames := s.resolveAddress(agentAddress)
	keyPEM, err := s.read(KeyPEMFile)
	if err != nil {
		return
	}

	csrPEM, fingerprint, err = pki.GenCSRUsingKey("agent", agentNames, agentIPs, keyPEM)
	if err != nil {
		err = errors.WithMessagef(err, "cannot generate CSR and private key for '%s' address", agentAddress)
		return
	}

	return
}

// Writes the provided fingerprint to the agent token file.
// The fingerprint is saved as hex bytes.
func (s *CertStore) WriteFingerprintAsToken(fingerprint [32]byte) error {
	if err := s.isValidToken(fingerprint[:]); err != nil {
		return err
	}
	fingerprintStr := storkutil.BytesToHex(fingerprint[:])
	return s.write(AgentTokenFile, []byte(fingerprintStr))
}

// Writes the given root CA in the PEM format to a file.
// It fails if the provided content is not a valid certificate file.
func (s *CertStore) WriteRootCAPEM(rootCAPEM []byte) error {
	if err := s.isValidCert(rootCAPEM); err != nil {
		return err
	}
	return s.write(RootCAFile, rootCAPEM)
}

// Writes the given cert in the PEM format to a file.
// It fails if the provided content is not a valid certificate file.
func (s *CertStore) WriteCertPEM(certPEM []byte) error {
	if err := s.isValidCert(certPEM); err != nil {
		return err
	}
	return s.write(CertPEMFile, certPEM)
}

// Checks if all files managed by the cert store are valid (they exist and
// have proper contents). Returns an error that describes all occurred
// problems. Returns nil if all is OK.
func (s *CertStore) IsValid() error {
	var validationErrors []error
	content, err := s.read(CertPEMFile)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidCert(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.read(RootCAFile)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidCert(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.read(KeyPEMFile)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidPrivateKey(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.read(AgentTokenFile)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidToken(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	return storkutil.CombineErrors("cert manager is not valid", validationErrors)
}

// Check if all files managed by the cert store are missing.
func (s *CertStore) IsEmpty() (bool, error) {
	if ok, err := s.isExist(KeyPEMFile); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(CertPEMFile); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(RootCAFile); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(AgentTokenFile); ok || err != nil {
		return false, err
	}
	return true, nil
}
