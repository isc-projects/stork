package agent

// Store the agent credentials to Kea CA.
// The data are read from a dedicated JSON file.
//
// The file structure is flexible to allow for future extensions,
// for example:
//
// - Credentials may be assigned per network instead of IP/port.
// - Store may contain different kinds of credentials

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// Kea CA location in the network. It is a key of the credentials store.
// It is the internal structure of the credentials store.
type location struct {
	IP   string
	Port int64
}

// Basic authentication credentials.
type BasicAuthCredentials struct {
	User     string
	Password string
}

// Credentials store with an API to add/update/delete the content.
type CredentialsStore struct {
	basicAuthCredentials map[location]*BasicAuthCredentials
}

// Structure of the credentials JSON file.
type CredentialsStoreContent struct {
	BasicAuth []CredentialsStoreContentBasicAuthEntry `json:"basic_auth"`
}

// Single Basic Auth item of the credentials JSON file.
type CredentialsStoreContentBasicAuthEntry struct {
	IP       *string
	Port     *int64
	User     *string
	Password *string
}

// Constructor of the credentials store.
func NewCredentialsStore() *CredentialsStore {
	return &CredentialsStore{
		basicAuthCredentials: make(map[location]*BasicAuthCredentials),
	}
}

// Constructor of the Basic Auth credentials.
func NewBasicAuthCredentials(user, password string) *BasicAuthCredentials {
	return &BasicAuthCredentials{
		User:     user,
		Password: password,
	}
}

// Get Basic Auth credentials by URL
// The Basic Auth is often used during HTTP calls. It is helper function
// to retrieve the credentials based on the request URL. The URL contains
// a protocol, URL segments and the query parameters.
func (cs *CredentialsStore) GetBasicAuthByURL(url string) (*BasicAuthCredentials, bool) {
	address, port, _ := storkutil.ParseURL(url)
	return cs.GetBasicAuth(address, port)
}

// Get Basic Auth credentials by the network location (IP address and port).
func (cs *CredentialsStore) GetBasicAuth(address string, port int64) (*BasicAuthCredentials, bool) {
	location, err := newLocation(address, port)
	if err != nil {
		return nil, false
	}
	item, ok := cs.basicAuthCredentials[location]
	return item, ok
}

// Add or update the Basic Auth credentials by the network location (IP address and port).
// If the credentials already exist in the store then they will be override.
func (cs *CredentialsStore) AddOrUpdateBasicAuth(address string, port int64, credentials *BasicAuthCredentials) error {
	location, err := newLocation(address, port)
	if err != nil {
		return err
	}
	cs.basicAuthCredentials[location] = credentials
	return nil
}

// Remove the Basic Auth credentials by the network location (IP address and port).
// If the credentials don't exist then this function does nothing.
func (cs *CredentialsStore) RemoveBasicAuth(address string, port int64) {
	location, err := newLocation(address, port)
	if err != nil {
		return
	}
	delete(cs.basicAuthCredentials, location)
}

// Read the credentials store content from reader.
// The file may contain IP addresses in the different forms,
// they will be converted to canonical forms.
func (cs *CredentialsStore) Read(reader io.Reader) error {
	rawContent, err := io.ReadAll(reader)
	if err != nil {
		return errors.Wrap(err, "Cannot read the credentials")
	}
	var content CredentialsStoreContent
	err = json.Unmarshal(rawContent, &content)
	if err != nil {
		return errors.Wrap(err, "Cannot parse the credentials")
	}
	return cs.loadContent(&content)
}

// Constructor of the network location (IP address and port).
func newLocation(address string, port int64) (location, error) {
	ip := storkutil.ParseIP(address)
	if ip == nil {
		return location{}, errors.Errorf("invalid IP address: %s", address)
	}

	return location{
		IP:   ip.NetworkAddress,
		Port: port,
	}, nil
}

// Load the content from JSON file to the credentials store.
func (cs *CredentialsStore) loadContent(content *CredentialsStoreContent) error {
	for _, entry := range content.BasicAuth {
		// Check required fields
		if entry.IP == nil {
			return errors.New("missing IP address")
		}
		if entry.Port == nil {
			return errors.New("missing port")
		}
		if entry.User == nil {
			return errors.New("missing user")
		}
		if entry.Password == nil {
			return errors.New("missing password")
		}

		credentials := NewBasicAuthCredentials(*entry.User, *entry.Password)
		err := cs.AddOrUpdateBasicAuth(*entry.IP, *entry.Port, credentials)
		if err != nil {
			return err
		}
	}
	return nil
}
