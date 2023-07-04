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

type CertStore struct {
	paths AgentPaths
}

func NewCertStore(paths AgentPaths) *CertStore {
	return &CertStore{paths: paths}
}

func (*CertStore) validCertValidator(content []byte) error {
	_, err := pki.ParseCert(content)
	return err
}

func (*CertStore) validPrivateKeyValidator(content []byte) error {
	_, err := pki.ParsePrivateKey(content)
	return err
}

func (*CertStore) tokenValidator(content []byte) error {
	if len(content) != 0 {
		return errors.New("file cannot be empty")
	}
	return nil
}

func (*CertStore) read(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read the file: %s", path)
	}
	return content, nil
}

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

// Parse provided address and return either an IP address as a one
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

func (s *CertStore) ReadToken() ([]byte, error) {
	content, err := s.read(s.paths.TokenPath)
	if err != nil {
		return nil, err
	}
	if err = s.tokenValidator(content); err != nil {
		return nil, err
	}
	return content, nil
}

func (s *CertStore) GetRootCA() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	ca, err := s.read(s.paths.RootCAPAth)
	if err != nil {
		return nil, err
	}

	// append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		err = errors.New("failed to append client root CA certificate")
		return nil, err
	}
	return certPool, nil
}

func (s *CertStore) GetTLSCert() (*tls.Certificate, error) {
	keyPEM, err := s.read(s.paths.KeyPEMPath)
	if err != nil {
		return nil, err
	}
	certPEM, err := s.read(s.paths.CertPEMPath)
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

func (s *CertStore) CreateKey() error {
	keyPEM, err := pki.GenKey()
	if err != nil {
		return err
	}
	err = s.write(s.paths.KeyPEMPath, keyPEM)
	if err != nil {
		return errors.Wrapf(err, "cannot write key file: %s", keyPEM)
	}

	// Invalidate cert, root CA and token.
	if err = s.removeIfExist(s.paths.CertPEMPath); err != nil {
		return err
	}
	if err = s.removeIfExist(s.paths.RootCAPAth); err != nil {
		return err
	}
	if err = s.removeIfExist(s.paths.TokenPath); err != nil {
		return err
	}
	return nil
}

func (s *CertStore) GenerateCSR(agentAddress string) (csrPEM []byte, fingerprint [32]byte, err error) {
	agentIPs, agentNames := s.resolveAddress(agentAddress)
	keyPEM, err := s.read(s.paths.KeyPEMPath)
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

func (s *CertStore) WriteFingerprintAsToken(fingerprint [32]byte) error {
	if err := s.tokenValidator(fingerprint[:]); err != nil {
		return err
	}
	fingerprintStr := storkutil.BytesToHex(fingerprint[:])
	return s.write(s.paths.TokenPath, []byte(fingerprintStr))
}

func (s *CertStore) WriteRootCAPEM(rootCAPEM []byte) error {
	if err := s.validCertValidator(rootCAPEM); err != nil {
		return err
	}
	return s.write(s.paths.RootCAPAth, rootCAPEM)
}

func (s *CertStore) WriteCertPEM(certPEM []byte) error {
	if err := s.validCertValidator(certPEM); err != nil {
		return err
	}
	return s.write(s.paths.CertPEMPath, certPEM)
}

func (s *CertStore) IsValid() error {
	var validationErrors []error
	content, err := s.read(s.paths.CertPEMPath)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.validCertValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.read(s.paths.RootCAPAth)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.validCertValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.read(s.paths.KeyPEMPath)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.validPrivateKeyValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.read(s.paths.TokenPath)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.tokenValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	return storkutil.CombineErrors("cert manager is not valid", validationErrors)
}

func (s *CertStore) IsEmpty() (bool, error) {
	if ok, err := s.isExist(s.paths.KeyPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(s.paths.CertPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(s.paths.RootCAPAth); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(s.paths.TokenPath); ok || err != nil {
		return false, err
	}
	return true, nil
}
