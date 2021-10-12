package agent

import (
	"io/ioutil"
	"os"
	"path"
)

// Helper function to store and defer restore
// original paths of: certificates, secrets and credentials.
func RememberPaths() func() {
	originalKeyPEMFile := KeyPEMFile
	originalCertPEMFile := CertPEMFile
	originalRootCAFile := RootCAFile
	originalAgentTokenFile := AgentTokenFile
	originalCredentialsFile := CredentialsFile

	return func() {
		KeyPEMFile = originalKeyPEMFile
		CertPEMFile = originalCertPEMFile
		RootCAFile = originalRootCAFile
		AgentTokenFile = originalAgentTokenFile
		CredentialsFile = originalCredentialsFile
	}
}

// Helper function that creates the temporary,
// self-signed certificates. Return the cleanup function
// and generation error. This function always creates
// the files with the same content.
func GenerateSelfSignedCerts() (func(), error) {
	restoreCerts := RememberPaths()
	tmpDir, err := ioutil.TempDir("", "reg")
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
		restoreCerts()
	}

	err = os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	if err != nil {
		cleanup()
		return nil, err
	}
	err = os.Mkdir(path.Join(tmpDir, "tokens"), 0755)
	if err != nil {
		cleanup()
		return nil, err
	}

	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// store proper content
	err = ioutil.WriteFile(KeyPEMFile, getKeyPEMContent(), 0600)
	if err != nil {
		cleanup()
		return nil, err
	}

	err = ioutil.WriteFile(CertPEMFile, getCertPEMContent(), 0600)
	if err != nil {
		cleanup()
		return nil, err
	}

	err = ioutil.WriteFile(RootCAFile, getCACertPEMContent(), 0600)
	if err != nil {
		cleanup()
		return nil, err
	}

	return cleanup, nil
}

// This function always return the same content. It is intended
// to avoid breaking IDE syntax highlighting (problem occurs on vscode).
func getCACertPEMContent() []byte {
	return []byte(`-----BEGIN CERTIFICATE-----
MIIFFjCCAv6gAwIBAgIBATANBgkqhkiG9w0BAQsFADAzMQswCQYDVQQGEwJVUzES
MBAGA1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMB4XDTIwMTIwODA4
MDc1M1oXDTMwMTIwODA4MDgwM1owMzELMAkGA1UEBhMCVVMxEjAQBgNVBAoTCUlT
QyBTdG9yazEQMA4GA1UEAxMHUm9vdCBDQTCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBALgcYkndHQGFmLk8yi8+yetaCBI1cLG/nm+hwjh5C2rh3lqqDziG
qRmcITxkEbCFujbxJzlaXop1MeXwg2YJMky3WM1GWomVKv3jOVR+GkQG70pp0qpt
JmU2CuXoNhwMFA0H22CG8pPRiilUGPI7RLXaLWpA8D+AslfPHR9TG00HbJ86Bi3g
m4/uPiGdcHS6Q+wmKQRsKs6wAKSmlCrvmaKfmVOkxpuKyuKgjmIKoCwY3gYL1T8L
idvVePvbP/Z2SRQOVbSV8eMaYuk+uFwGKq8thLHs8bIEKhrIGlzDss6ZlPotTi2V
I6e6lb06oFuCSfhBaiHPw2sldwYvE/I8MkWUAuWtBgNvVE/e64FgJb1lGIzJpYMj
5jUp9Z13INsXy9zA8nKyZAK4fI6vlQGRg3bERn+S4Q6HXQor9Ma8QWxsqbdiC9dt
pxpzyx11tWg0jEgzCEBfk9IZjlGqyCdX5Z9pshHkQZ9VeK+DG0s6tYEm7BO1ssQD
+qbJS2PJq4Cwe82a6gO+lDz8A+xiXk8dJeTb8hf/c1NY192rqSLewI8oaHOLKEQg
XNSPEEkQqtIqn92Y5oKhLYKmYkwfOgldpj0XQQ3YwUnsOCfy2wRVNRg6VYnbjca8
rSy58t2MfovKWz9UcKhpnXefSdMgR7VhGv0ekDddGIfONn153uyjN/LpAgMBAAGj
NTAzMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFILkrDPZAlboeF+nav7C
Rf7nN1W+MA0GCSqGSIb3DQEBCwUAA4ICAQCDfvIgo70Y0Mi+Rs0mF6114z2gGQ7a
7/VnxV9w9uIjuaARq42E2DemFs5e72tPIfT9UWncgs5ZfyO5w2tjRpUOaVCSS5VY
93qzXBfTsqgkrkVRwec4qqZxpNqpsL9u2ZIfsSJ3BJWFV3Zq/3cOrDulfR5bk0G4
hYo/GDyLHjNalBFpetJSIk7l0VOkr2CBUvxKBOP0U1IQGXd+NL/8zW6UB6OitqNL
/tO+JztOpjo6ZYKJGZvxyL/3FUsiHmd8UwqAjnFjQRd3w0gseyqWDgILXQaDXQ5D
vs2oK+HheJv4h6CdrcIdWlWRKoZP3odZyWB0l31kpMbgYC/tMPYebG6mjPx+/S4m
7L+K27zmm2wItUaWI12ky2FPgeW78ALoKDYWmQ+CnpBNE1iFUf4qRzmypu77DmmM
bLgLFj8Bb50j0/zciPO7+D1h6hCPxwXdfQk0tnWBqjImmK3enbkEsw77kF8MkNjr
Hka0EeTt0hyEFKGgJ7jVdbjLFnRzre63q1GuQbLkOibyjf9WS/1ljv1Ps82aWeE+
rh78iXtpm8c/2IqrI37sLbAIs08iPj8ULV57RbcZI7iTYFIjKwPlWL8O2U1mopYP
RXkm1+W4cMzZS14MLfmacBHnI7Z4mRKvc+zEdco/l4omlszafmUXxnCOmqZlhqbm
/p0vFt1oteWWSQ==
-----END CERTIFICATE-----`)
}

// This function always return the same content. It is intended
// to avoid breaking IDE syntax highlighting (problem occurs on vscode).
func getKeyPEMContent() []byte {
	return []byte(`-----BEGIN PRIVATE KEY-----
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
H/Njg8CqtOWDeTVICuUq60wkbEkxYg==
-----END PRIVATE KEY-----`)
}

// This function always return the same content. It is intended
// to avoid breaking IDE syntax highlighting (problem occurs on vscode).
func getCertPEMContent() []byte {
	return []byte(`-----BEGIN CERTIFICATE-----
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
wQ==
-----END CERTIFICATE-----`)
}
