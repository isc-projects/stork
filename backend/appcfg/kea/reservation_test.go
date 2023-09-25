package keaconfig_test

import (
	"testing"

	"github.com/pkg/errors"
	require "github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
)

// Test host returning static values and implementing Host interface.
type testHost struct {
	identifiers []struct {
		Type  string
		Value []byte
	}
	subnetIDTuple struct {
		id  int64
		err error
	}
	clientClasses  []string
	nextServer     string
	serverHostname string
	bootFileName   string
}

// Creates a test host with default values.
func createDefaultTestHost() *testHost {
	return &testHost{
		identifiers: []struct {
			Type  string
			Value []byte
		}{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "duid",
				Value: []byte{2, 2, 2, 2, 2, 2},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1, 1},
			},
			{
				Type:  "client-id",
				Value: []byte{1, 2, 3, 4},
			},
			{
				Type:  "flex-id",
				Value: []byte{9, 9, 9, 9},
			},
		},
		subnetIDTuple: struct {
			id  int64
			err error
		}{
			id:  123,
			err: nil,
		},
		clientClasses: []string{
			"foo", "bar",
		},
		nextServer:     "192.2.2.2",
		serverHostname: "my-server-hostname",
		bootFileName:   "/tmp/bootfile",
	}
}

// Returns static host identifiers.
func (host testHost) GetHostIdentifiers() []struct {
	Type  string
	Value []byte
} {
	return host.identifiers
}

// Returns static IP reservation of various kinds.
func (host testHost) GetIPReservations() []string {
	return []string{"2001:db8:1::1", "3000::/16", "2001:db8:2::2", "3001::/16", "192.0.2.1", "10.0.0.1"}
}

// Returns static hostname.
func (host testHost) GetHostname() string {
	return "hostname.example.org"
}

// Returns static subnet ID.
func (host testHost) GetSubnetID(int64) (int64, error) {
	return host.subnetIDTuple.id, host.subnetIDTuple.err
}

// Returns static client classes.
func (host testHost) GetClientClasses(int64) []string {
	return host.clientClasses
}

// Returns next server.
func (host testHost) GetNextServer(int64) string {
	return host.nextServer
}

// Returns server hostname.
func (host testHost) GetServerHostname(int64) string {
	return host.serverHostname
}

// Returns boot file name.
func (host testHost) GetBootFileName(int64) string {
	return host.bootFileName
}

// Returns static DHCP options.
func (host testHost) GetDHCPOptions(int64) (options []dhcpmodel.DHCPOptionAccessor) {
	testOptions := []testDHCPOption{
		{
			code:        5,
			encapsulate: "dhcp4",
			fields: []testDHCPOptionField{
				*newTestDHCPOptionField("ipv4-address", "192.0.2.1"),
			},
		},
		{
			code:        7,
			encapsulate: "dhcp4",
			fields: []testDHCPOptionField{
				*newTestDHCPOptionField("ipv4-address", "10.0.0.1"),
			},
		},
	}
	for _, to := range testOptions {
		options = append(options, to)
	}
	return
}

// Test conversion of the host to Kea reservation.
func TestCreateReservation(t *testing.T) {
	host := createDefaultTestHost()
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).AnyTimes().Return(false)
	reservation, err := keaconfig.CreateReservation(1, lookup, host)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	require.Equal(t, "010203040506", reservation.HWAddress)
	require.Equal(t, "020202020202", reservation.DUID)
	require.Equal(t, "010101010101", reservation.CircuitID)
	require.Equal(t, "01020304", reservation.ClientID)
	require.Equal(t, "09090909", reservation.FlexID)
	require.Equal(t, "192.0.2.1", reservation.IPAddress)
	require.Len(t, reservation.IPAddresses, 2)
	require.Equal(t, "2001:db8:1::1", reservation.IPAddresses[0])
	require.Equal(t, "2001:db8:2::2", reservation.IPAddresses[1])
	require.Len(t, reservation.Prefixes, 2)
	require.Equal(t, "3000::/16", reservation.Prefixes[0])
	require.Equal(t, "3001::/16", reservation.Prefixes[1])
	require.Equal(t, "hostname.example.org", reservation.Hostname)
	require.Len(t, reservation.ClientClasses, 2)
	require.Equal(t, "foo", reservation.ClientClasses[0])
	require.Equal(t, "bar", reservation.ClientClasses[1])
	require.Len(t, reservation.OptionData, 2)
}

// Test conversion of the host to Kea reservation that can be used
// in host_cmds command.
func TestCreateHostCmdsReservation(t *testing.T) {
	host := createDefaultTestHost()
	controller := gomock.NewController(t)
	lookup := NewMockDHCPOptionDefinitionLookup(controller)
	lookup.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).AnyTimes().Return(false)
	reservation, err := keaconfig.CreateHostCmdsReservation(1, lookup, host)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	require.Equal(t, "010203040506", reservation.HWAddress)
	require.Equal(t, "192.0.2.1", reservation.IPAddress)
	require.Len(t, reservation.IPAddresses, 2)
	require.Equal(t, "2001:db8:1::1", reservation.IPAddresses[0])
	require.Equal(t, "2001:db8:2::2", reservation.IPAddresses[1])
	require.Len(t, reservation.Prefixes, 2)
	require.Equal(t, "3000::/16", reservation.Prefixes[0])
	require.Equal(t, "3001::/16", reservation.Prefixes[1])
	require.Equal(t, "hostname.example.org", reservation.Hostname)
	require.Len(t, reservation.ClientClasses, 2)
	require.Equal(t, "foo", reservation.ClientClasses[0])
	require.Equal(t, "bar", reservation.ClientClasses[1])
	require.Equal(t, "192.2.2.2", reservation.NextServer)
	require.Equal(t, "my-server-hostname", reservation.ServerHostname)
	require.Equal(t, "/tmp/bootfile", reservation.BootFileName)
	require.EqualValues(t, 123, reservation.SubnetID)
}

// Test conversion of the host to a structure used when deleting the
// reservation from Kea.
func TestCreateHostCmdsDeletedReservation(t *testing.T) {
	host := createDefaultTestHost()
	reservation, err := keaconfig.CreateHostCmdsDeletedReservation(1, host)
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Use the first identifier to delete the reservation.
	require.Equal(t, "hw-address", reservation.IdentifierType)
	require.Equal(t, "010203040506", reservation.Identifier)
	require.EqualValues(t, 123, reservation.SubnetID)
}

// Test that conversion error is returned when the host has no
// identifiers.
func TestCreateHostCmdsDeletedReservationNoIdentfiers(t *testing.T) {
	host := createDefaultTestHost()
	host.identifiers = []struct {
		Type  string
		Value []byte
	}{}
	reservation, err := keaconfig.CreateHostCmdsDeletedReservation(1, host)
	require.Error(t, err)
	require.Nil(t, reservation)
}

// Test that conversion error is returned when getting a subnet
// ID fails.
func TestCreateHostCmdsDeletedReservationSubnetIDError(t *testing.T) {
	host := createDefaultTestHost()
	host.subnetIDTuple.err = errors.New("error getting subnet ID")
	reservation, err := keaconfig.CreateHostCmdsDeletedReservation(1, host)
	require.Error(t, err)
	require.Nil(t, reservation)
}
