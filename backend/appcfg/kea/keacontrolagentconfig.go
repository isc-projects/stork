package keaconfig

import (
	"github.com/pkg/errors"
	"muzzammil.xyz/jsonc"
)

// Kea Control Agent JSON configuration - root wrapper.
type keaControlAgentConfigWrapper struct {
	ControlAgent *KeaControlAgentConfig
}

// Kea Control Agent JSON configuration - root node.
type KeaControlAgentConfig struct {
	HTTPHost     string
	HTTPPort     int64
	TrustAnchor  string
	CertFile     string
	KeyFile      string
	CertRequired bool
}

func NewKeaControlAgentFromJSON(raw []byte) (*KeaControlAgentConfig, error) {
	var data map[string]interface{}
	err := jsonc.Unmarshal(raw, &data)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse Kea Control Agent config JSON")
	}
	var root keaControlAgentConfigWrapper
	err = decode(data, &root)
	if err != nil {
		return nil, errors.WithMessage(err, "the JSON content isn't a valid Kea Control Agent configuration")
	}

	if root.ControlAgent == nil {
		return nil, errors.New("invalid JSON content")
	}

	return root.ControlAgent, nil
}
