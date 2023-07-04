package agent

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/pkg/errors"
	"isc.org/stork/pki"
	storkutil "isc.org/stork/util"
)

type CertStore struct {
	fileManager *AgentFileManager
}

func NewCertStore(fileManager *AgentFileManager) *CertStore {
	return &CertStore{fileManager: fileManager}
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
	content, err := s.fileManager.Read(s.fileManager.TokenPath)
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
	ca, err := s.fileManager.Read(s.fileManager.RootCAPAth)
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
	keyPEM, err := s.fileManager.Read(s.fileManager.KeyPEMPath)
	if err != nil {
		return nil, err
	}
	certPEM, err := s.fileManager.Read(s.fileManager.CertPEMPath)
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
	err = s.fileManager.Write(s.fileManager.KeyPEMPath, keyPEM)
	if err != nil {
		return errors.Wrapf(err, "cannot write key file: %s", keyPEM)
	}

	// Invalidate cert, root CA and token.
	if err = s.fileManager.RemoveIfExist(s.fileManager.CertPEMPath); err != nil {
		return err
	}
	if err = s.fileManager.RemoveIfExist(s.fileManager.RootCAPAth); err != nil {
		return err
	}
	if err = s.fileManager.RemoveIfExist(s.fileManager.TokenPath); err != nil {
		return err
	}
	return nil
}

func (s *CertStore) GenerateCSR(agentAddress string) (csrPEM []byte, fingerprint [32]byte, err error) {
	agentIPs, agentNames := s.resolveAddress(agentAddress)
	keyPEM, err := s.fileManager.Read(s.fileManager.KeyPEMPath)
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
	return s.fileManager.Write(s.fileManager.TokenPath, []byte(fingerprintStr))
}

func (s *CertStore) WriteRootCAPEM(rootCAPEM []byte) error {
	if err := s.validCertValidator(rootCAPEM); err != nil {
		return err
	}
	return s.fileManager.Write(s.fileManager.RootCAPAth, rootCAPEM)
}

func (s *CertStore) WriteCertPEM(certPEM []byte) error {
	if err := s.validCertValidator(certPEM); err != nil {
		return err
	}
	return s.fileManager.Write(s.fileManager.CertPEMPath, certPEM)
}

func (s *CertStore) IsValid() error {
	var validationErrors []error
	content, err := s.fileManager.Read(s.fileManager.CertPEMPath)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.validCertValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.fileManager.Read(s.fileManager.RootCAPAth)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.validCertValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.fileManager.Read(s.fileManager.KeyPEMPath)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.validPrivateKeyValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.fileManager.Read(s.fileManager.TokenPath)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.tokenValidator(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	return storkutil.CombineErrors("cert manager is not valid", validationErrors)
}

func (s *CertStore) IsEmpty() (bool, error) {
	if ok, err := s.fileManager.IsExist(s.fileManager.KeyPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.fileManager.IsExist(s.fileManager.CertPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.fileManager.IsExist(s.fileManager.RootCAPAth); ok || err != nil {
		return false, err
	}
	if ok, err := s.fileManager.IsExist(s.fileManager.TokenPath); ok || err != nil {
		return false, err
	}
	return true, nil
}
