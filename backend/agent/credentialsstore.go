package agent

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// Type for store the agent credentials to Kea CA.
//
// It isn't just a map, because
// I predict that it may change a lot in the future.
// For example:
//
// - Credentials may be assigned not to exact IP/port, but
//   to subnetwork.
// - Store may contains different kinds of credentials

// Location CA in the network.
type Location struct {
	IP   string
	Port int64
}

// Basic authentication credentials.
type BasicAuthCredentials struct {
	Login    string
	Password string
}

type CredentialsStore struct {
	basicAuthCredentials map[Location]*BasicAuthCredentials
}

type CredentialsStoreConfigurationBasicAuthEntry struct {
	Location
	BasicAuthCredentials
}

type CredentialsStoreConfiguration struct {
	Basic []CredentialsStoreConfigurationBasicAuthEntry
}

func NewCredentialsStore() *CredentialsStore {
	return &CredentialsStore{}
}

func NewBasicAuthCredentials(login, password string) *BasicAuthCredentials {
	return &BasicAuthCredentials{
		Login:    login,
		Password: password,
	}
}

func (cs *CredentialsStore) GetBasicAuthByURL(url string) (*BasicAuthCredentials, bool) {
	address, port, _ := storkutil.ParseURL(url)
	return cs.GetBasicAuth(address, port)
}

func (cs *CredentialsStore) GetBasicAuth(address string, port int64) (*BasicAuthCredentials, bool) {
	location := newLocation(address, port)
	item, ok := cs.basicAuthCredentials[location]
	return item, ok
}

func (cs *CredentialsStore) AddOrUpdateBasicAuth(address string, port int64, credentials *BasicAuthCredentials) {
	location := newLocation(address, port)
	cs.basicAuthCredentials[location] = credentials
}

func (cs *CredentialsStore) RemoveBasicAuth(address string, port int64) {
	location := newLocation(address, port)
	delete(cs.basicAuthCredentials, location)
}

func (cs *CredentialsStore) ReadFromFile(path string) error {
	jsonFile, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "cannot open a credentials file")
	}
	defer jsonFile.Close()
	content, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return errors.Wrap(err, "cannot read a credentials file")
	}
	var configuration CredentialsStoreConfiguration
	err = json.Unmarshal(content, &configuration)
	if err != nil {
		return errors.Wrap(err, "cannot parse a credentials file")
	}
	cs.loadConfiguration(&configuration)
	return nil
}

func newLocation(address string, port int64) Location {
	return Location{
		IP:   address,
		Port: port,
	}
}

func (cs *CredentialsStore) loadConfiguration(configuration *CredentialsStoreConfiguration) {
	nextBasicAuthCredentials := make(map[Location]*BasicAuthCredentials, len(configuration.Basic))

	for _, entry := range configuration.Basic {
		credentials := NewBasicAuthCredentials(entry.Login, entry.Password)
		nextBasicAuthCredentials[newLocation(entry.IP, entry.Port)] = credentials
	}

	cs.basicAuthCredentials = nextBasicAuthCredentials
}
