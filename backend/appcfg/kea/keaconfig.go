// Package keaconfig implements functions to parse and manage the Kea
// daemons' configurations.
package keaconfig

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
	"muzzammil.xyz/jsonc"
)

// An interface to the common data for all daemons. It must be implemented
// for all configuration types (all daemon types).
type commonConfigAccessor interface {
	GetHookLibraries() HookLibraries
	GetLoggers() []Logger
}

// A structure holding a configuration of a single Kea server.
// It can hold configurations of all Kea daemon types supported by Stork.
// The Raw field keeps the complete raw configuration as a map of strings.
// It is marshalled and unmarshalled for storing the Kea daemon configuration
// in the Stork database. It can also be used to construct the config-set
// commands. The other embedded structures hold parsed configurations for
// the respective daemons. For instance, for the Kea CA configuration the
// CtrlAgentConfig field is set and the parsed data are returned from this
// structure. All other structures are nil.
type Config struct {
	Raw              RawConfig `json:"-"`
	*CtrlAgentConfig `json:"Control-agent,omitempty"`
	*D2Config        `json:"DhcpDdns,omitempty"`
	*DHCPv4Config    `json:"Dhcp4,omitempty"`
	*DHCPv6Config    `json:"Dhcp6,omitempty"`
}

// Raw configuration type for a Kea server.
type RawConfig map[string]any

// Convenience function used by different exported functions returning
// an interface to the DHCP-specific data.
func (c *Config) getDHCPConfigAccessor() dhcpConfigAccessor {
	switch {
	case c.IsDHCPv4():
		return c.DHCPv4Config
	case c.IsDHCPv6():
		return c.DHCPv6Config
	default:
		return nil
	}
}

// Convenience function used by different exported functions returning
// an interface to the data common for all supported servers.
func (c *Config) getCommonConfigAccessor() commonConfigAccessor {
	switch {
	case c.IsCtrlAgent():
		return c.CtrlAgentConfig
	case c.IsD2():
		return c.D2Config
	case c.IsDHCPv4():
		return c.DHCPv4Config
	case c.IsDHCPv6():
		return c.DHCPv6Config
	default:
		return nil
	}
}

// Custom unmarshaller making two passes. The first pass parses the configuration
// into the dedicated structures. The second pass parses the entire configuration
// into the raw map.
func (c *Config) UnmarshalJSON(data []byte) error {
	// First pass.
	type ct Config
	if err := jsonc.Unmarshal(data, (*ct)(c)); err != nil {
		return err
	}
	// Second pass.
	type rt map[string]any
	if err := jsonc.Unmarshal(data, (*rt)(&c.Raw)); err != nil {
		return err
	}
	return nil
}

// Converts the configuration to the JSON form.
func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Raw)
}

// Creates a new configuration instance from a JSON string. It uses a custom
// unmarshaller supporting comments in JSON.
func NewConfig(raw string) (*Config, error) {
	var config Config
	err := jsonc.Unmarshal([]byte(raw), &config)
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing Kea configuration: %s", err)
	}
	return &config, nil
}

// Creates a new configuration instance from a map. This function is here for
// historical reasons. Part of the Stork code parses the received configuration
// structures into the maps (e.g., the code that receives Kea responses over the
// command channel). No new code should use this function, and the existing code
// should be eventually refactored to not use this function.
func NewConfigFromMap(rawCfg *map[string]any) *Config {
	// Turn the JSON back into the string, so it can be unmarshalled into the
	// proper structure. Yes, it is inefficient, but it is temporary.
	marshalled, err := json.Marshal(rawCfg)
	if err != nil {
		return nil
	}
	newCfg, err := NewConfig(string(marshalled))
	if err != nil {
		return nil
	}
	return newCfg
}

// Returns true if the Config holds the control agent's configuration.
func (c *Config) IsCtrlAgent() bool {
	return c.CtrlAgentConfig != nil
}

// Returns true if the Config holds the D2's configuration.
func (c *Config) IsD2() bool {
	return c.D2Config != nil
}

// Returns true if the Config holds the DHCPv4 server's configuration.
func (c *Config) IsDHCPv4() bool {
	return c.DHCPv4Config != nil
}

// Returns true if the Config holds the DHCPv6 server's configuration.
func (c *Config) IsDHCPv6() bool {
	return c.DHCPv6Config != nil
}

// Returns multi-threading configuration for a DHCP server.
func (c *Config) GetMultiThreading() (mt *MultiThreading) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		mt = accessor.GetCommonDHCPConfig().MultiThreading
	}
	return
}

// Checks if the multi threading has been enabled in the Kea configuration.
// Versions earlier than 2.3.5 have MT disabled by default. Other versions
// have MT enabled by default.
func (c *Config) IsMultiThreadingEnabled(keaVersion storkutil.SemanticVersion) bool {
	var mt *MultiThreading
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		mt = accessor.GetCommonDHCPConfig().MultiThreading
	}
	if keaVersion.GreaterThanOrEqual(storkutil.NewSemanticVersion(2, 3, 5)) {
		return mt == nil || mt.EnableMultiThreading == nil || *mt.EnableMultiThreading
	}
	return mt != nil && mt.EnableMultiThreading != nil && *mt.EnableMultiThreading
}

// It returns all database backend configurations found in the DHCP configuration.
// It includes lease-database, host-database or hosts-databases, config-databases
// and the database used by the Legal Log hooks library. It is safe to call for
// non-DHCP daemons but it always returns an empty structure in this case.
func (c *Config) GetAllDatabases() (databases Databases) {
	accessor := c.getDHCPConfigAccessor()
	if accessor == nil {
		// Not a DHCP server.
		return
	}
	dhcpConfig := accessor.GetCommonDHCPConfig()
	// Lease database.
	databases.Lease = dhcpConfig.LeaseDatabase
	// Host database can be specified in two different structures (i.e., host-database
	// or host-databases).
	if dhcpConfig.HostsDatabase != nil {
		databases.Hosts = append(databases.Hosts, *dhcpConfig.HostsDatabase)
	} else if len(dhcpConfig.HostsDatabases) > 0 {
		databases.Hosts = dhcpConfig.HostsDatabases
	}
	if dhcpConfig.ConfigControl != nil {
		databases.Config = dhcpConfig.ConfigControl.ConfigDatabases
	}
	// Legal logging.
	if _, params, ok := c.GetHookLibraries().GetLegalLogHookLibrary(); ok {
		databases.Forensic = &params.Database
	}
	return
}

// Returns DHCP cache parameters.
func (c *Config) GetCacheParameters() (parameters CacheParameters) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().CacheParameters
	}
	return
}

// Returns DHCP client classes. It returns an empty slice when there are
// no client classes or the configuration is not associated with a DHCP
// server.
func (c *Config) GetClientClasses() (clientClasses []ClientClass) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		clientClasses = accessor.GetCommonDHCPConfig().ClientClasses
	}
	return
}

// Returns DHCP DDNS parameters.
func (c *Config) GetDDNSParameters() (parameters DDNSParameters) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().DDNSParameters
	}
	return
}

// Returns DHCP hostname char parameters.
func (c *Config) GetHostnameCharParameters() (parameters HostnameCharParameters) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().HostnameCharParameters
	}
	return
}

// Returns DHCP timer parameters.
func (c *Config) GetTimerParameters() (parameters TimerParameters) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().TimerParameters
	}
	return
}

// Returns DHCPv6 preferred lifetime parameters.
func (c *Config) GetPreferredLifetimeParameters() (parameters PreferredLifetimeParameters) {
	if c.IsDHCPv6() {
		parameters = c.DHCPv6Config.PreferredLifetimeParameters
	}
	return
}

// Returns DHCP valid lifetime parameters.
func (c *Config) GetValidLifetimeParameters() (parameters ValidLifetimeParameters) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().ValidLifetimeParameters
	}
	return
}

// Returns DHCP allocator.
func (c *Config) GetAllocator() (allocator *string) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		allocator = accessor.GetCommonDHCPConfig().Allocator
	}
	return
}

// Returns prefix delegation allocator.
func (c *Config) GetPDAllocator() (allocator *string) {
	if c.IsDHCPv6() {
		allocator = c.DHCPv6Config.PDAllocator
	}
	return
}

// Returns DHCPv4 authoritative flag.
func (c *Config) GetAuthoritative() (authoritative *bool) {
	if c.IsDHCPv4() {
		authoritative = c.DHCPv4Config.Authoritative
	}
	return
}

// Returns DHCPv4 boot file name.
func (c *Config) GetBootFileName() (bootFileName *string) {
	if c.IsDHCPv4() {
		bootFileName = c.DHCPv4Config.BootFileName
	}
	return
}

// Returns DHCPv4 match client ID.
func (c *Config) GetMatchClientID() (matchClientID *bool) {
	if c.IsDHCPv4() {
		matchClientID = c.DHCPv4Config.MatchClientID
	}
	return
}

// Returns DHCPv4 next server.
func (c *Config) GetNextServer() (nextServer *string) {
	if c.IsDHCPv4() {
		nextServer = c.DHCPv4Config.NextServer
	}
	return
}

// Returns DHCPv4 server hostname.
func (c *Config) GetServerHostname() (serverHostname *string) {
	if c.IsDHCPv4() {
		serverHostname = c.DHCPv4Config.ServerHostname
	}
	return
}

// Returns DHCPv6 rapid commit flag.
func (c *Config) GetRapidCommit() (rapidCommit *bool) {
	if c.IsDHCPv6() {
		rapidCommit = c.DHCPv6Config.RapidCommit
	}
	return
}

// Returns a set of parameters specifying the global DHCP host reservation
// modes. It is safe to call for non-DHCP daemons but it always returns an
// empty structure in this case.
func (c *Config) GetGlobalReservationParameters() (parameters ReservationParameters) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().ReservationParameters
	}
	return
}

// Returns DHCP host reservations. It is safe to call it for non-DHCP daemons
// but it always returns an empty slice in this case.
func (c *Config) GetReservations() (reservations []Reservation) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		reservations = accessor.GetCommonDHCPConfig().Reservations
	}
	return
}

// Returns all configured hook libraries.
func (c *Config) GetHookLibraries() (hooks HookLibraries) {
	if accessor := c.getCommonConfigAccessor(); accessor != nil {
		hooks = accessor.GetHookLibraries()
	}
	return
}

// Returns a hook library with a matching name. For example, to find a configured
// lease_cmds hook library, call this function with the 'libdhcp_lease_cmds' name.
func (c *Config) GetHookLibrary(name string) (path string, params map[string]any, ok bool) {
	if accessor := c.getCommonConfigAccessor(); accessor != nil {
		path, params, ok = accessor.GetHookLibraries().GetHookLibrary(name)
	}
	return
}

// Returns configured loggers.
func (c *Config) GetLoggers() (loggers []Logger) {
	if accessor := c.getCommonConfigAccessor(); accessor != nil {
		loggers = accessor.GetLoggers()
	}
	return
}

// Finds and returns a subnet (i.e., Subnet4 or Subnet6) having the specified
// prefix. The type of the returned object behind the interface depends on
// the type of the configured DHCP server. It always returns a nil interface
// for non-DHCP servers.
func (c *Config) GetSubnetByPrefix(prefix string) (subnet Subnet) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		subnet = accessor.GetSubnetByPrefix(prefix)
	}
	return
}

// Returns all shared networks (i.e., SharedNetwork4 or SharedNetwork6)
// configured in the DHCP server. It is safe to call it for a non-DHCP
// server but it always returns an empty slice in this case.
func (c *Config) GetSharedNetworks(includeRootSubnets bool) (sharedNetworks []SharedNetwork) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		sharedNetworks = accessor.GetSharedNetworks(includeRootSubnets)
	}
	return
}

// Returns all subnets (i.e., Subnet4 or Subnet6) configured in the DHCP
// server. It is safe to call it for non-DHCP server but it always returns
// an empty slice in this case.
func (c *Config) GetSubnets() (subnets []Subnet) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		subnets = accessor.GetSubnets()
	}
	return
}

// Returns DHCP extended info flag.
func (c *Config) GetStoreExtendedInfo() (storeExtendedInfo *bool) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		storeExtendedInfo = accessor.GetCommonDHCPConfig().StoreExtendedInfo
	}
	return
}

// Returns a slice of the global DHCP option data.
func (c *Config) GetDHCPOptions() (options []SingleOptionData) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		options = accessor.GetDHCPOptions()
	}
	return
}

// Recursively hides sensitive data in the configuration. It traverses the raw
// configuration and nullifies the values for the following keys: password,
// secret, token. It doesn't modify the parsed configuration.
func (c *Config) HideSensitiveData() {
	hideSensitiveData((*map[string]any)(&c.Raw))
}

// Hides the sensitive data in the configuration map. It traverses the raw
// configuration and nullifies the values for the following keys: password,
// secret, token.
func hideSensitiveData(obj *map[string]any) {
	for entryKey, entryValue := range *obj {
		// Check if the value holds sensitive data.
		entryKeyNormalized := strings.ToLower(entryKey)
		if entryKeyNormalized == "password" || entryKeyNormalized == "secret" || entryKeyNormalized == "token" {
			(*obj)[entryKey] = nil
			continue
		}
		// Check if it is an array.
		array, ok := entryValue.([]interface{})
		if ok {
			for _, arrayItemValue := range array {
				// Check if it is a subobject (or array).
				subobject, ok := arrayItemValue.(map[string]any)
				if ok {
					hideSensitiveData(&subobject)
				}
			}
			continue
		}
		// Check if it is a subobject (but not array).
		subobject, ok := entryValue.(map[string]any)
		if ok {
			hideSensitiveData(&subobject)
		}
	}
}
