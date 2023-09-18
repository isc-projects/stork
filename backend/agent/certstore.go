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

// Paths pointing to agent's key and cert, and CA cert from server,
// and agent token generated by agent.
// They are being modified by tests so need to be writable.
var (
	KeyPEMFile     = "/var/lib/stork-agent/certs/key.pem"          //nolint:gochecknoglobals
	CertPEMFile    = "/var/lib/stork-agent/certs/cert.pem"         //nolint:gochecknoglobals
	RootCAFile     = "/var/lib/stork-agent/certs/ca.pem"           //nolint:gochecknoglobals
	AgentTokenFile = "/var/lib/stork-agent/tokens/agent-token.txt" //nolint:gochecknoglobals,gosec
)

// An utility structure to perform actions on the GRPC/TLS certificate files.
type CertStore struct {
	keyPEMPath     string
	certPEMPath    string
	rootCAPEMPath  string
	agentTokenPath string
}

// Constructs a new cert store instance. Uses the paths where the GRPC
// certificates, obtained from the server, are located.
func NewCertStoreDefault() *CertStore {
	return &CertStore{
		keyPEMPath:     KeyPEMFile,
		certPEMPath:    CertPEMFile,
		rootCAPEMPath:  RootCAFile,
		agentTokenPath: AgentTokenFile,
	}
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
	if len(content) == 0 {
		return errors.New("file could not be empty")
	}
	return nil
}

// Reads the file content.
func (*CertStore) read(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read the file: %s", path)
	}
	return content, nil
}

// Reads the agent token.
func (s *CertStore) readAgentTokenFile() ([]byte, error) {
	content, err := s.read(s.agentTokenPath)
	err = errors.WithMessage(err, "could not read the agent token")
	return content, err
}

// Reads the cert file.
func (s *CertStore) readCert() ([]byte, error) {
	content, err := s.read(s.certPEMPath)
	err = errors.WithMessage(err, "could not read the cert")
	return content, err
}

// Reads the root CA file.
func (s *CertStore) readRootCA() ([]byte, error) {
	content, err := s.read(s.rootCAPEMPath)
	err = errors.WithMessage(err, "could not read the root CA")
	return content, err
}

// Reads the private key file.
func (s *CertStore) readPrivateKey() ([]byte, error) {
	content, err := s.read(s.keyPEMPath)
	err = errors.WithMessage(err, "could not read the private key")
	return content, err
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
		return errors.Wrapf(err, "could not write the file: %s", path)
	}
	return nil
}

// Writes the agent token.
func (s *CertStore) writeAgentToken(content []byte) error {
	err := s.write(s.agentTokenPath, content)
	return errors.WithMessage(err, "could not write the agent token")
}

// Writes the cert file.
func (s *CertStore) writeCert(content []byte) error {
	err := s.write(s.certPEMPath, content)
	return errors.WithMessage(err, "could not write the cert")
}

// Writes the root CA.
func (s *CertStore) writeRootCA(content []byte) error {
	err := s.write(s.rootCAPEMPath, content)
	return errors.WithMessage(err, "could not write the root CA")
}

// Writes the private key.
func (s *CertStore) writePrivateKey(content []byte) error {
	err := s.write(s.keyPEMPath, content)
	return errors.WithMessage(err, "could not write the private key")
}

// Checks if a given path exists on a filesystem.
// Returns an error if there is a problem with access to the path.
func (*CertStore) isExist(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, errors.Wrapf(err, "could not stat the file: %s", path)
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
		return errors.Wrapf(err, "could not remove the file: %s", path)
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
		return errors.Wrapf(err, "could not create a directory tree: %s", directory)
	}
	return nil
}

// Parses provided address and returns either an IP address as a one
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
func (s *CertStore) ReadToken() (string, error) {
	content, err := s.readAgentTokenFile()
	if err != nil {
		return "", err
	}
	if err = s.isValidToken(content); err != nil {
		return "", err
	}
	return string(content), nil
}

// Reads and parses the root CA file.
func (s *CertStore) ReadRootCA() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	ca, err := s.readRootCA()
	if err != nil {
		return nil, err
	}

	// Append the client certificates from the CA.
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		err = errors.New("failed to append root CA")
		return nil, err
	}
	return certPool, nil
}

// Reads and parses the certificate and private key files. Combines them into
// a single TLS certificate.
func (s *CertStore) ReadTLSCert() (*tls.Certificate, error) {
	keyPEM, err := s.readPrivateKey()
	if err != nil {
		return nil, err
	}
	certPEM, err := s.readCert()
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
	err = s.writePrivateKey(keyPEM)
	if err != nil {
		return err
	}

	// Invalidate cert, root CA and token.
	for _, path := range []string{s.certPEMPath, s.rootCAPEMPath, s.agentTokenPath} {
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
	keyPEM, err := s.readPrivateKey()
	if err != nil {
		err = errors.WithMessage(err, "could not read the private key")
		return
	}

	csrPEM, fingerprint, err = pki.GenCSRUsingKey("agent", agentNames, agentIPs, keyPEM)
	if err != nil {
		err = errors.WithMessagef(err, "could not generate CSR and private key for '%s' address", agentAddress)
		return
	}

	return
}

// Writes the provided fingerprint to the agent token file.
// The fingerprint is saved as hex bytes.
func (s *CertStore) WriteFingerprintAsToken(fingerprint [32]byte) error {
	fingerprintHex := []byte(storkutil.BytesToHex(fingerprint[:]))
	if err := s.isValidToken(fingerprintHex); err != nil {
		return err
	}
	return s.writeAgentToken(fingerprintHex)
}

// Writes the given root CA in the PEM format to a file.
// It fails if the provided content is not a valid certificate file.
func (s *CertStore) WriteRootCAPEM(rootCAPEM []byte) error {
	if err := s.isValidCert(rootCAPEM); err != nil {
		err = errors.WithMessage(err, "the provided root CA PEM content is invalid")
		return err
	}
	return s.writeRootCA(rootCAPEM)
}

// Writes the given cert in the PEM format to a file.
// It fails if the provided content is not a valid certificate file.
func (s *CertStore) WriteCertPEM(certPEM []byte) error {
	if err := s.isValidCert(certPEM); err != nil {
		err = errors.WithMessage(err, "the provided TLS cert content is invalid")
		return err
	}
	return s.writeCert(certPEM)
}

// Checks if all files managed by the cert store are valid (they exist and
// have proper contents). Returns an error that describes all occurred
// problems. Returns nil if all is OK.
func (s *CertStore) IsValid() error {
	var validationErrors []error
	content, err := s.readCert()
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidCert(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.readRootCA()
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidCert(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.readPrivateKey()
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidPrivateKey(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	content, err = s.readAgentTokenFile()
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else if err = s.isValidToken(content); err != nil {
		validationErrors = append(validationErrors, err)
	}

	return storkutil.CombineErrors("cert store is not valid", validationErrors)
}

// Check if all files managed by the cert store are missing.
func (s *CertStore) IsEmpty() (bool, error) {
	if ok, err := s.isExist(s.keyPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(s.certPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(s.rootCAPEMPath); ok || err != nil {
		return false, err
	}
	if ok, err := s.isExist(s.agentTokenPath); ok || err != nil {
		return false, err
	}
	return true, nil
}
