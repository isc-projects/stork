package oidc

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"isc.org/stork/server/authdata"
)

// OIDC authentication method Metadata.
type Metadata struct {
	settings Settings
}

// GetDescription implements [authdata.AuthenticationMetadata].
func (m *Metadata) GetDescription() string {
	provider := m.settings.IdentityProviderName
	if provider == "OpenID Connect" {
		// In case of default setting, use more appropriate provider name.
		provider = "OpenID Provider"
	}
	return fmt.Sprintf("OAuth2/OIDC authentication. You will get redirected to %s to authenticate and authorize your access to Stork.", provider)
}

// GetID implements [authdata.AuthenticationMetadata].
func (m *Metadata) GetID() string {
	return "oidc"
}

// GetIcon implements [authdata.AuthenticationMetadata].
func (m *Metadata) GetIcon() (io.ReadCloser, error) {
	// While OIDC auth method is implemented in server core code, we don't need to provide
	// the icon using AuthenticationMetadata interface.
	return nil, errors.New("no icon provided by OIDC auth method implemented in server core")
}

// GetName implements [authdata.AuthenticationMetadata].
func (m *Metadata) GetName() string {
	return fmt.Sprintf("Log in with %s", m.settings.IdentityProviderName)
}

var _ authdata.AuthenticationMetadata = (*Metadata)(nil)
