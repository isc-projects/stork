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

var (
	_ RawConfigAccessor = (*Config)(nil)
	_ RawConfigAccessor = (*SettableConfig)(nil)
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

// A structure holding partial configuration for a single Kea server.
// It exposes functions to set selected configuration parameters. It
// also implements the RawConfigAccessor interface that can be used to
// merge this partial configuration into the full server configuration.
type SettableConfig struct {
	*SettableCtrlAgentConfig `json:"Control-agent,omitempty"`
	*SettableD2Config        `json:"DhcpDdns,omitempty"`
	*SettableDHCPv4Config    `json:"Dhcp4,omitempty"`
	*SettableDHCPv6Config    `json:"Dhcp6,omitempty"`
}

// An interface providing a function returning raw configuration. It
// must be implemented by both Config and SettableConfig.
type RawConfigAccessor interface {
	GetRawConfig() (RawConfig, error)
}

// Raw configuration type for a Kea server.
type RawConfig = map[string]any

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

// Custom unmarshaller parsing configuration into the dedicated structures
// held in the Config object. It is called internally by the custom unmarshaller
// performing two passes and by the Merge function.
func (c *Config) unmarshalIntoAccessibleConfig(data []byte) error {
	type ct Config
	c.CtrlAgentConfig = nil
	c.D2Config = nil
	c.DHCPv4Config = nil
	c.DHCPv6Config = nil
	err := jsonc.Unmarshal(data, (*ct)(c))
	return errors.Wrapf(err, "cannot unmarshal the data into an accessible config")
}

// Custom unmarshaller making two passes. The first pass parses the configuration
// into the dedicated structures. The second pass parses the entire configuration
// into the raw map.
func (c *Config) UnmarshalJSON(data []byte) error {
	// First pass.
	if err := c.unmarshalIntoAccessibleConfig(data); err != nil {
		return err
	}
	// Second pass.
	type rt map[string]any
	err := jsonc.Unmarshal(data, (*rt)(&c.Raw))
	return errors.Wrapf(err, "cannot unmarshal the data into a raw config")
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

// Returns raw configuration.
func (c *Config) GetRawConfig() (RawConfig, error) {
	return c.Raw, nil
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

// Returns DHCP DDNS connectivity parameters.
func (c *Config) GetDHCPDDNSParameters() (parameters *DHCPDDNS) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().DHCPDDNS
	}
	return
}

// Returns parameters pertaining to lease expiration processing.
func (c *Config) GetExpiredLeasesProcessingParameters() (parameters *ExpiredLeasesProcessing) {
	if accessor := c.getDHCPConfigAccessor(); accessor != nil {
		parameters = accessor.GetCommonDHCPConfig().ExpiredLeasesProcessing
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
	hideSensitiveData(&c.Raw)
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

// Merges raw configuration into current configuration.
func (c *Config) Merge(source RawConfigAccessor) error {
	// Get source and destination raw configurations. The merge is performed
	// at the raw configuration levels.
	destConfig, _ := c.GetRawConfig()
	sourceConfig, err := source.GetRawConfig()
	if err != nil {
		return errors.Wrap(err, "problem getting raw configuration during Kea configurations merge")
	}
	// Merge the source config into destination config. The resulting
	// configuration is stored in the Config.Raw field. However, the
	// server-specific configurations (e.g., Config.DHCPv4Config) have
	// not been updated at this point.
	c.Raw = merge(destConfig, sourceConfig).(RawConfig)
	// After the merge, the hash is no longer valid, so let's delete it.
	delete(c.Raw, "hash")
	// In order to update the server-specific configuration structures
	// we need to serialize the raw configuration and then unmarshal this
	// configuration.
	data, err := json.Marshal(c.Raw)
	if err != nil {
		return errors.Wrap(err, "problem serializing merged Kea configuration")
	}
	// This call only performs only one pass which unmarshals the configuration
	// into server-specific structures. It does not unmarshal into the raw
	// configuration because it has been already updated by the call to Merge().
	err = c.unmarshalIntoAccessibleConfig(data)
	return errors.WithMessage(err, "problem parsing merged Kea configuration")
}

// Merges branches of the two configurations. If the branches are maps
// the values of the maps are merged into one. Otherwise, the merged value
// is preferred and replaces the value in destination. Explicit null values
// in the source config designate the corresponding values to be removed
// from the target.
func merge(c1, c2 any) any {
	switch c1 := c1.(type) {
	case RawConfig:
		c2, ok := c2.(RawConfig)
		if !ok {
			return c1
		}
		for k, v2 := range c2 {
			if v1, ok := c1[k]; ok {
				if v := merge(v1, v2); v != nil {
					c1[k] = v
				} else {
					delete(c1, k)
				}
			} else {
				if v2 != nil {
					c1[k] = v2
				}
			}
			// Delete any explicit null values in the target config.
			// These values are merely used in the source config to
			// indicate which fields should be removed.
			if c, ok := c1[k].(RawConfig); ok {
				for k := range c {
					if c[k] == nil {
						delete(c, k)
					}
				}
			}
		}
	default:
		return c2
	}
	return c1
}

// Creates new settable Control Agent configuration instance.
func NewSettableCtrlAgentConfig() *SettableConfig {
	return &SettableConfig{
		SettableCtrlAgentConfig: &SettableCtrlAgentConfig{},
	}
}

// Creates new settable D2 configuration instance.
func NewSettableD2Config() *SettableConfig {
	return &SettableConfig{
		SettableD2Config: &SettableD2Config{},
	}
}

// Creates new settable DHCPv4 configuration instance.
func NewSettableDHCPv4Config() *SettableConfig {
	return &SettableConfig{
		SettableDHCPv4Config: &SettableDHCPv4Config{},
	}
}

// Creates new settable DHCPv6 configuration instance.
func NewSettableDHCPv6Config() *SettableConfig {
	return &SettableConfig{
		SettableDHCPv6Config: &SettableDHCPv6Config{},
	}
}

// Returns raw settable configuration. The raw configuration is created
// upon the call to this function from the already configured values.
func (c *SettableConfig) GetRawConfig() (RawConfig, error) {
	serializedConfig, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "problem getting raw settable config")
	}
	var mapConfig RawConfig
	err = json.Unmarshal(serializedConfig, &mapConfig)
	return mapConfig, errors.Wrap(err, "problem getting raw settable config")
}

// Returns settable configuration as string.
func (c *SettableConfig) GetSerializedConfig() (string, error) {
	serializedConfig, err := json.Marshal(c)
	return string(serializedConfig), errors.Wrap(err, "problem getting serialized settable config")
}

// Returns true if the Config holds the DHCPv4 server's configuration.
func (c *SettableConfig) IsDHCPv4() bool {
	return c.SettableDHCPv4Config != nil
}

// Returns true if the Config holds the DHCPv6 server's configuration.
func (c *SettableConfig) IsDHCPv6() bool {
	return c.SettableDHCPv6Config != nil
}

// Convenience function used by different exported functions returning
// an interface to setting the DHCP-specific data.
func (c *SettableConfig) getDHCPConfigModifier() dhcpConfigModifier {
	switch {
	case c.IsDHCPv4():
		return c.SettableDHCPv4Config
	case c.IsDHCPv6():
		return c.SettableDHCPv6Config
	default:
		return nil
	}
}

// Convenience function used by different exported functions returning
// an interface to setting the DHCPv4-specific data.
func (c *SettableConfig) getDHCPv4ConfigModifier() dhcp4ConfigModifier {
	switch {
	case c.IsDHCPv4():
		return c.SettableDHCPv4Config
	default:
		return nil
	}
}

// Convenience function used by different exported functions returning
// an interface to setting the DHCPv6-specific data.
func (c *SettableConfig) getDHCPv6ConfigModifier() dhcp6ConfigModifier {
	switch {
	case c.IsDHCPv6():
		return c.SettableDHCPv6Config
	default:
		return nil
	}
}

// Sets a DHCPv4 or DHCPv6 parameter using the provided setter or returns an
// error when such a parameter is not supported by the DHCP servers.
func (c *SettableConfig) setDHCPParameter(setter func(modifier dhcpConfigModifier), parameterName string) (err error) {
	if modifier := c.getDHCPConfigModifier(); modifier != nil {
		setter(modifier)
	} else {
		err = NewUnsupportedConfigParameter(parameterName)
	}
	return
}

// Sets a DHCPv4 parameter using the provided setter or returns an error when such a
// parameter is not supported by the DHCPv4 servers.
func (c *SettableConfig) setDHCPv4Parameter(setter func(modifier dhcp4ConfigModifier), parameterName string) (err error) {
	if modifier := c.getDHCPv4ConfigModifier(); modifier != nil {
		setter(modifier)
	} else {
		err = NewUnsupportedConfigParameter(parameterName)
	}
	return
}

// Sets a DHCPv6 parameter using the provided setter or returns an error when such a
// parameter is not supported by the DHCPv6 servers.
func (c *SettableConfig) setDHCPv6Parameter(setter func(modifier dhcp6ConfigModifier), parameterName string) (err error) {
	if modifier := c.getDHCPv6ConfigModifier(); modifier != nil {
		setter(modifier)
	} else {
		err = NewUnsupportedConfigParameter(parameterName)
	}
	return
}

// Sets an allocator.
func (c *SettableConfig) SetAllocator(allocator *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetAllocator(allocator)
	}, "allocator")
}

// Sets cache threshold.
func (c *SettableConfig) SetCacheThreshold(cacheThreshold *float32) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetCacheThreshold(cacheThreshold)
	}, "cache-threshold")
}

// Sets boolean flag indicating if DDNS updates should be sent.
func (c *SettableConfig) SetDDNSSendUpdates(ddnsSendUpdates *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSSendUpdates(ddnsSendUpdates)
	}, "ddns-send-updates")
}

// Sets boolean flag indicating whether the DHCP server should override the
// client's wish to not update the DNS.
func (c *SettableConfig) SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSOverrideNoUpdate(ddnsOverrideNoUpdate)
	}, "ddns-override-no-update")
}

// Sets the boolean flag indicating whether the DHCP server should ignore the
// client's wish to update the DNS on its own.
func (c *SettableConfig) SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSOverrideClientUpdate(ddnsOverrideClientUpdate)
	}, "ddns-override-client-update")
}

// Sets the enumeration specifying whether the server should honor
// the hostname or Client FQDN sent by the client or replace this name.
func (c *SettableConfig) SetDDNSReplaceClientName(ddnsReplaceClientName *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSReplaceClientName(ddnsReplaceClientName)
	}, "ddns-replace-client-name")
}

// Sets a prefix to be prepended to the generated Client FQDN.
func (c *SettableConfig) SetDDNSGeneratedPrefix(ddnsGeneratedPrefix *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSGeneratedPrefix(ddnsGeneratedPrefix)
	}, "ddns-generated-prefix")
}

// Sets a suffix appended to the partial name sent to the DNS.
func (c *SettableConfig) SetDDNSQualifyingSuffix(ddnsQualifyingSuffix *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSQualifyingSuffix(ddnsQualifyingSuffix)
	}, "ddns-qualifying-suffix")
}

// Sets a boolean flag, which when true instructs the server to always
// update DNS when leases are renewed, even if the DNS information
// has not changed.
func (c *SettableConfig) SetDDNSUpdateOnRenew(ddnsUpdateOnRenew *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSUpdateOnRenew(ddnsUpdateOnRenew)
	}, "ddns-update-on-renew")
}

// Sets a boolean flag which is passed to kea-dhcp-ddns with each DDNS
// update request, to indicate whether DNS update conflict
// resolution as described in RFC 4703 should be employed for the
// given update request.
func (c *SettableConfig) SetDDNSUseConflictResolution(ddnsUseConflictResolution *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSUseConflictResolution(ddnsUseConflictResolution)
	}, "ddns-use-conflict-resolution")
}

// Sets the DDNS conflict resolution mode.
func (c *SettableConfig) SetDDNSConflictResolutionMode(ddnsConflictResolutionMode *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSConflictResolutionMode(ddnsConflictResolutionMode)
	}, "ddns-conflict-resolution-mode")
}

// Sets the the percent of the lease's lifetime to use for the DNS TTL.
func (c *SettableConfig) SetDDNSTTLPercent(ddnsTTLPercent *float32) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDDNSTTLPercent(ddnsTTLPercent)
	}, "ddns-ttl-percent")
}

// Enables connectivity with the DHCP DDNS daemon and sending DNS updates.
func (c *SettableConfig) SetDHCPDDNSEnableUpdates(enableUpdates *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSEnableUpdates(enableUpdates)
	}, "enable-updates")
}

// Sets the IP address on which D2 listens for requests.
func (c *SettableConfig) SetDHCPDDNSServerIP(serverIP *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSServerIP(serverIP)
	}, "server-ip")
}

// Sets the port on which D2 listens for requests.
func (c *SettableConfig) SetDHCPDDNSServerPort(serverPort *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSServerPort(serverPort)
	}, "server-port")
}

// Sets the IP address which DHCP server uses to send requests to D2.
func (c *SettableConfig) SetDHCPDDNSSenderIP(senderIP *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSSenderIP(senderIP)
	}, "sender-ip")
}

// Sets the port which DHCP server uses to send requests to D2.
func (c *SettableConfig) SetDHCPDDNSSenderPort(senderPort *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSSenderPort(senderPort)
	}, "sender-port")
}

// Sets the maximum number of requests allowed to queue while waiting to be sent to D2.
func (c *SettableConfig) SetDHCPDDNSMaxQueueSize(maxQueueSize *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSMaxQueueSize(maxQueueSize)
	}, "max-queue-size")
}

// Sets the socket protocol to use when sending requests to D2.
func (c *SettableConfig) SetDHCPDDNSNCRProtocol(ncrProtocol *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSNCRProtocol(ncrProtocol)
	}, "ncr-protocol")
}

// Sets the packet format to use when sending requests to D2.
func (c *SettableConfig) SetDHCPDDNSNCRFormat(ncrFormat *string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNSNCRFormat(ncrFormat)
	}, "ncr-format")
}

// Sets the DHCP DDNS structure.
func (c *SettableConfig) SetDHCPDDNS(dhcpDDNS *SettableDHCPDDNS) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPDDNS(dhcpDDNS)
	}, "dhcp-ddns")
}

// Sets the number of seconds since the last removal of the expired
// leases, when the next removal should occur.
func (c *SettableConfig) SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetELPFlushReclaimedTimerWaitTime(flushReclaimedTimerWaitTime)
	}, "flush-reclaimed-timer-wait-time")
}

// Sets the length of time in seconds to keep expired leases in the
// lease database (lease affinity).
func (c *SettableConfig) SetELPHoldReclaimedTime(holdReclaimedTime *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetELPHoldReclaimedTime(holdReclaimedTime)
	}, "hold-reclaimed-time")
}

// Sets the maximum number of expired leases that can be processed in
// a single attempt to clean up expired leases from the lease database.
func (c *SettableConfig) SetELPMaxReclaimLeases(maxReclaimedLeases *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetELPMaxReclaimLeases(maxReclaimedLeases)
	}, "max-reclaimed-leases")
}

// Sets the maximum time in milliseconds that a single attempt to clean
// up expired leases from the lease database may take.
func (c *SettableConfig) SetELPMaxReclaimTime(maxReclaimTime *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetELPMaxReclaimTime(maxReclaimTime)
	}, "max-reclaim-time")
}

// Sets the length of time in seconds since the last attempt to process
// expired leases before initiating the next attempt.
func (c *SettableConfig) SetELPReclaimTimerWaitTime(reclaimTimerWaitTime *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetELPReclaimTimerWaitTime(reclaimTimerWaitTime)
	}, "reclaim-timer-wait-time")
}

// Sets the maximum number of expired lease-processing cycles which didn't
// result in full cleanup of the expired leases from the lease database,
// after which a warning message is issued.
func (c *SettableConfig) SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetELPUnwarnedReclaimCycles(unwarnedReclaimCycles)
	}, "unwarned-reclaim-cycles")
}

// Sets the expired leases processing structure.
func (c *SettableConfig) SetExpiredLeasesProcessing(expiredLeasesProcessing *SettableExpiredLeasesProcessing) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetExpiredLeasesProcessing(expiredLeasesProcessing)
	}, "expired-leases-processing")
}

// Sets the boolean flag enabling global reservations.
func (c *SettableConfig) SetReservationsGlobal(reservationsGlobal *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetReservationsGlobal(reservationsGlobal)
	}, "reservations-global")
}

// Sets the boolean flag enabling in-subnet reservations.
func (c *SettableConfig) SetReservationsInSubnet(reservationsInSubnet *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetReservationsInSubnet(reservationsInSubnet)
	}, "reservations-in-subnet")
}

// Sets the boolean flag enabling out-of-pool reservations.
func (c *SettableConfig) SetReservationsOutOfPool(reservationsOutOfPool *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetReservationsOutOfPool(reservationsOutOfPool)
	}, "reservations-out-of-pool")
}

// Sets a boolean flag enabling an early global reservations lookup.
func (c *SettableConfig) SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup *bool) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetEarlyGlobalReservationsLookup(earlyGlobalReservationsLookup)
	}, "early-global-reservations-lookup")
}

// Sets host reservation identifiers to be used for host reservation lookup.
func (c *SettableConfig) SetHostReservationIdentifiers(hostReservationIdentifiers []string) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetHostReservationIdentifiers(hostReservationIdentifiers)
	}, "host-reservation-identifiers")
}

// Sets valid lifetime in the configuration.
func (c *SettableConfig) SetValidLifetime(validLifetime *int64) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetValidLifetime(validLifetime)
	}, "valid-lifetime")
}

// Sets DHCP option data in the configuration.
func (c *SettableConfig) SetDHCPOptions(options []SingleOptionData) error {
	return c.setDHCPParameter(func(modifier dhcpConfigModifier) {
		modifier.SetDHCPOptions(options)
	}, "option-data")
}

// Sets a boolean flag indicating whether the server is authoritative.
func (c *SettableConfig) SetAuthoritative(authoritative *bool) error {
	return c.setDHCPv4Parameter(func(modifier dhcp4ConfigModifier) {
		modifier.SetAuthoritative(authoritative)
	}, "authoritative")
}

// Sets a boolean flag indicating whether the server should return client
// ID in its responses.
func (c *SettableConfig) SetEchoClientID(echoClientID *bool) error {
	return c.setDHCPv4Parameter(func(modifier dhcp4ConfigModifier) {
		modifier.SetEchoClientID(echoClientID)
	}, "authoritative")
}

// Sets allocator for prefix delegation.
func (c *SettableConfig) SetPDAllocator(pdAllocator *string) error {
	return c.setDHCPv6Parameter(func(modifier dhcp6ConfigModifier) {
		modifier.SetPDAllocator(pdAllocator)
	}, "authoritative")
}
