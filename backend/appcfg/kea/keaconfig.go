package keaconfig

import (
	"bytes"
	"net"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"muzzammil.xyz/jsonc"
)

const (
	localhost string = "localhost"
	// Root name for DHCPv4.
	RootNameDHCPv4 string = "Dhcp4"
	// Root name for DHCPv6.
	RootNameDHCPv6 string = "Dhcp6"
)

// Kea daemon configuration map. It comprises a set of functions
// which retrieve complex data structures from the configuration.
type Map map[string]interface{}

// Groups the functions used to read data from the top Kea config.
type TopConfig interface {
	// Returns name of the root configuration node, e.g. Dhcp4.
	// The second returned value designates whether the root node
	// name was successfully found or not.
	GetRootName() (string, bool)
	// Returns a list found at the top level of the configuration under
	// a given name. If the given parameter does not exist or it is
	// not a list, the ok value returned is set to false.
	GetTopLevelList(name string) (list []interface{}, ok bool)
	// Returns a map found at the top level of the configuration under a
	// given name. If the given parameter does not exist or it is not
	// a map, the ok value returned is set to false.
	GetTopLevelMap(name string) (m map[string]interface{}, ok bool)
}

// Get the database sections from the configuration.
type DatabaseConfig interface {
	GetAllDatabases() Databases
}

// Structure representing a configuration of the single hooks library.
type HooksLibrary struct {
	Library    string
	Parameters map[string]interface{}
}

// Structure representing output_options for a logger.
type LoggerOutputOptions struct {
	Output string
}

// Structure representing a single logger configuration.
type Logger struct {
	Name          string
	OutputOptions []LoggerOutputOptions `mapstructure:"output_options"`
	Severity      string
	DebugLevel    int `mapstructure:"debuglevel"`
}

// Structure representing a configuration of a HA peer.
type Peer struct {
	Name         *string
	URL          *string
	Role         *string
	AutoFailover *bool `mapstructure:"auto-failover"`
}

// Structure representing a multi-threading configuration of the HA hooks
// library.
type HAMultiThreading struct {
	EnableMultiThreading  *bool `mapstructure:"enable-multi-threading"`
	HTTPDedicatedListener *bool `mapstructure:"http-dedicated-listener"`
	HTTPListenerThreads   *int  `mapstructure:"http-listener-threads"`
	HTTPClientThreads     *int  `mapstructure:"http-client-threads"`
}

// Structure representing a configuration of the HA hooks library.
type HA struct {
	ThisServerName    *string `mapstructure:"this-server-name"`
	Mode              *string
	HeartbeatDelay    *int `mapstructure:"heartbeat-delay"`
	MaxResponseDelay  *int `mapstructure:"max-response-delay"`
	MaxAckDelay       *int `mapstructure:"max-ack-delay"`
	MaxUnackedClients *int `mapstructure:"max-unacked-clients"`
	Peers             []Peer
	MultiThreading    *HAMultiThreading `mapstructure:"multi-threading"`
}

// Structure representing a configuration of the control socket in the
// Kea Control Agent.
type ControlSocket struct {
	SocketName string `mapstructure:"socket-name"`
	SocketType string `mapstructure:"socket-type"`
}

// Structure representing configuration of multiple control sockets in
// in the Kea Control Agent.
type ControlSockets struct {
	D2      *ControlSocket
	Dhcp4   *ControlSocket
	Dhcp6   *ControlSocket
	NetConf *ControlSocket
}

// Structure representing database connection parameters. It is common
// for all supported backend types.
type Database struct {
	Path string `mapstructure:"path"`
	Type string `mapstructure:"type"`
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
}

// Structure holding all possible database configurations in the Kea
// configuration structure, i.e. lease database, hosts databases,
// config backend and forensic logging database. If some of the
// configurations are not present, nil values or empty slices are
// returned for them. This structure is returned by functions parsing
// Kea configurations to find present database configurations.
type Databases struct {
	Lease    *Database
	Hosts    []Database
	Config   []Database
	Forensic *Database
}

// A structure comprising host reservation modes at the particular
// configuration level. This structure can be embedded in the
// structures for decoding subnets and shared networks. In that
// case, the reservation modes configured at the subnet or shared
// network level will be decoded into the embedded structure.
// Do not read the decoded modes directly from the structure.
// Call appropriate functions on this structure to test the
// decoded modes. The Deprecated field holds the value of the
// reservation-mode setting that was deprecated since Kea 1.9.x.
type ReservationModes struct {
	OutOfPool  *bool   `mapstructure:"reservations-out-of-pool,omitempty"`
	InSubnet   *bool   `mapstructure:"reservations-in-subnet,omitempty"`
	Global     *bool   `mapstructure:"reservations-global,omitempty"`
	Deprecated *string `mapstructure:"reservation-mode,omitempty"`
}

// Structure representing multi-threading parameters.
type MultiThreading struct {
	EnableMultiThreading *bool `mapstructure:"enable-multi-threading"`
	ThreadPoolSize       *int  `mapstructure:"thread-pool-size"`
	PacketQueueSize      *int  `mapstructure:"packet-queue-size"`
}

// Creates new instance from the pointer to the map of interfaces.
func New(rawCfg *map[string]interface{}) *Map {
	newCfg := Map(*rawCfg)
	return &newCfg
}

// Create new instance from the configuration provided as JSON text.
func NewFromJSON(rawCfg string) (*Map, error) {
	var cfg Map
	err := jsonc.Unmarshal([]byte(rawCfg), &cfg)
	if err != nil {
		err := errors.Wrapf(err, "problem parsing JSON text: %s", rawCfg)
		return nil, err
	}
	return &cfg, nil
}

// Returns name of the root configuration node, e.g. Dhcp4.
// The second returned value designates whether the root node
// name was successfully found or not.
func (c *Map) GetRootName() (string, bool) {
	// This map will typically hold just a single element, but
	// in the past Kea supported Logging parameter aside of the
	// DHCP server configuration so we need to eliminate this one.
	for key := range *c {
		if key != "Logging" {
			return key, true
		}
	}
	return "", false
}

// Returns root node of the Kea configuration.
func (c *Map) getRootNode() (rootNode map[string]interface{}, ok bool) {
	rootName, rootNameOk := c.GetRootName()
	if !rootNameOk {
		return rootNode, rootNameOk
	}
	if cfg, rootNodeOk := (*c)[rootName]; rootNodeOk {
		rootNode, ok = cfg.(map[string]interface{})
	}
	return rootNode, ok
}

// Returns an entry found at the top level of the configuration under a
// given name. If the given parameter does not exist, the ok value
// returned is set to false.
func (c *Map) getTopLevelEntry(entryName string) (interface{}, bool) {
	root, ok := c.getRootNode()
	if !ok {
		return nil, false
	}

	raw, ok := root[entryName]
	return raw, ok
}

// Returns a list found at the top level of the configuration under
// a given name. If the given parameter does not exist or it is
// not a list, the ok value returned is set to false.
func (c *Map) GetTopLevelList(name string) (list []interface{}, ok bool) {
	node, ok := c.getTopLevelEntry(name)
	if ok {
		list, ok = node.([]interface{})
	}
	return
}

// Returns a map found at the top level of the configuration under a
// given name. If the given parameter does not exist or it is not
// a map, the ok value returned is set to false.
func (c *Map) GetTopLevelMap(name string) (m map[string]interface{}, ok bool) {
	node, ok := c.getTopLevelEntry(name)
	if ok {
		m, ok = node.(map[string]interface{})
	}
	return
}

// Returns a string found at the top level of the configuration under a
// given name. If the given parameter does not exist, the string is empty, and
// the ok value returned is set to false.
func (c *Map) getTopLevelEntryString(entryName string) (out string, ok bool) {
	raw, ok := c.getTopLevelEntry(entryName)
	if ok {
		out, ok = raw.(string)
	}
	return
}

// Returns a list of all hooks libraries found in the configuration.
func (c *Map) GetHooksLibraries() (parsedLibraries []HooksLibrary) {
	if hooksLibrariesList, ok := c.GetTopLevelList("hooks-libraries"); ok {
		_ = mapstructure.Decode(hooksLibrariesList, &parsedLibraries)
	}
	return parsedLibraries
}

// Returns the information about a hooks library having a specified name
// if it exists in the configuration. The name parameter designates the
// name of the library, e.g. libdhcp_ha. The returned values include the
// path to the library, library configuration and the flag indicating
// whether the library exists or not.
func (c *Map) GetHooksLibrary(name string) (path string, params map[string]interface{}, ok bool) {
	libraries := c.GetHooksLibraries()
	for _, lib := range libraries {
		if strings.Contains(lib.Library, name) {
			path = lib.Library
			params = lib.Parameters
			ok = true
		}
	}
	return path, params, ok
}

// Returns the multi-threading parameters or nil if they are not provided.
func (c *Map) GetMultiThreading() (output *MultiThreading) {
	if data, ok := c.getTopLevelEntry("multi-threading"); ok {
		_ = mapstructure.Decode(data, &output)
	}
	return
}

// Returns configuration of the HA hooks library in a parsed form.
func (c *Map) GetHAHooksLibrary() (path string, params HA, ok bool) {
	path, paramsMap, ok := c.GetHooksLibrary("libdhcp_ha")
	if !ok {
		return path, params, ok
	}

	// HA hooks library should contain high-availability parameter being a
	// single element list. If it doesn't exist, it is an error.
	if haParamsList, ok := paramsMap["high-availability"].([]interface{}); !ok {
		path = ""
	} else {
		// Parse the list of HA configurations into a list of structures.
		var paramsList []HA
		err := mapstructure.Decode(haParamsList, &paramsList)
		if err != nil || len(paramsList) == 0 {
			path = ""
		} else {
			// HA configuration found, return it.
			params = paramsList[0]
		}
	}

	return path, params, ok
}

// Checks if the mandatory peer parameters are set. It doesn't check if the
// values are correct.
func (p Peer) IsSet() bool {
	return p.Name != nil && p.URL != nil && p.Role != nil
}

// Checks if the mandatory Kea HA configuration parameters are set. It doesn't
// check parameters consistency, though.
func (c HA) IsSet() bool {
	// Check if peers are valid.
	for _, p := range c.Peers {
		if !p.IsSet() {
			return false
		}
	}
	// Check other required parameters.
	return c.ThisServerName != nil && c.Mode != nil
}

// Parses a list of loggers specified for the server.
func (c *Map) GetLoggers() (parsedLoggers []Logger) {
	if loggersList, ok := c.GetTopLevelList("loggers"); ok {
		_ = mapstructure.Decode(loggersList, &parsedLoggers)
	}
	return parsedLoggers
}

// Parses a map of control sockets in Kea Control Agent.
func (c *Map) GetControlSockets() (parsedSockets ControlSockets) {
	if socketsMap, ok := c.GetTopLevelMap("control-sockets"); ok {
		_ = mapstructure.Decode(socketsMap, &parsedSockets)
	}
	return parsedSockets
}

// Returns a list of daemons for which sockets have been configured.
func (sockets ControlSockets) ConfiguredDaemonNames() (names []string) {
	s := reflect.ValueOf(&sockets).Elem()
	t := s.Type()
	for i := 0; i < s.NumField(); i++ {
		if !s.Field(i).IsNil() {
			names = append(names, strings.ToLower(t.Field(i).Name))
		}
	}
	return names
}

// Convenience function extracting database connection information at the
// certain scope level. The first argument is the map structure containing
// the map under specified name. This map should contain the database
// connection information to be returned. If that map doesn't exist, a nil
// value is returned. This function can be used to extract the values of the
// lease-database and legal logging configurations.
func getDatabase(scope map[string]interface{}, name string) *Database {
	if databaseNode, ok := scope[name]; ok {
		database := Database{}
		_ = mapstructure.Decode(databaseNode, &database)
		// Set default host value.
		if len(database.Host) == 0 {
			database.Host = localhost
		}
		return &database
	}
	return nil
}

// Convenience function extracting an array of the database connection
// information at the certain scope level. The first argument is the map
// structure containing the list under specified name. This list should
// contain zero, one or more maps with database connection information
// to be returned. If that map doesn't exist an empty slice is returned.
// This function can be used to extract values of hosts-databases and
// config-databases lists.
func getDatabases(scope map[string]interface{}, name string) (databases []Database) {
	if databaseNode, ok := scope[name]; ok {
		_ = mapstructure.Decode(databaseNode, &databases)
		// Set default host value.
		for i := range databases {
			if len(databases[i].Host) == 0 {
				databases[i].Host = localhost
			}
		}
	}
	return databases
}

// It returns all database backend configurations found in the Kea configuration.
// It includes lease-database, host-database or hosts-databases, config-databases
// and the database used by the Legal Log hooks library.
func (c *Map) GetAllDatabases() (databases Databases) {
	rootNode, ok := c.getRootNode()
	if !ok {
		return
	}
	// lease-database
	databases.Lease = getDatabase(rootNode, "lease-database")
	// hosts-database
	hostsDatabase := getDatabase(rootNode, "hosts-database")
	if hostsDatabase == nil {
		// hosts-database is empty, but hosts-databases can contain
		// multiple entries.
		databases.Hosts = getDatabases(rootNode, "hosts-databases")
	} else {
		// hosts-database was not empty, so append this single
		// element.
		databases.Hosts = append(databases.Hosts, *hostsDatabase)
	}
	// config-databases
	if configControl, ok := rootNode["config-control"].(map[string]interface{}); ok {
		databases.Config = getDatabases(configControl, "config-databases")
	}
	// Forensic Logging hooks library configuration.
	if _, legalParams, ok := c.GetHooksLibrary("libdhcp_legal_log"); ok {
		database := Database{}
		_ = mapstructure.Decode(legalParams, &database)
		// Set default host value.
		if len(database.Path) == 0 && len(database.Host) == 0 {
			database.Host = localhost
		}
		databases.Forensic = &database
	}
	return databases
}

// Checks if the global reservation mode has been enabled.
// Returns (first parameter):
// - reservations-global value if set OR
// - true when reservation-mode is "global".
// The second parameter indicates whether the returned value was set
// explicitly (when true) or is a default value (when false).
func (modes *ReservationModes) IsGlobal() (bool, bool) {
	if modes.Global != nil {
		return *modes.Global, true
	}
	if modes.Deprecated != nil {
		return *modes.Deprecated == "global", true
	}
	return false, false
}

// Checks if the in-subnet reservation mode has been enabled.
// Returns (first parameter):
// - reservations-in-subnet value if set OR
// - true when reservation-mode is set and is "all" or "out-of-pool" OR
// - false when reservation-mode is set and configured to other values OR
// - true when no mode is explicitly configured.
// The second parameter indicates whether the returned value was set
// explicitly (when true) or is a default value (when false).
func (modes *ReservationModes) IsInSubnet() (bool, bool) {
	if modes.InSubnet != nil {
		return *modes.InSubnet, true
	}
	if modes.Deprecated != nil {
		return *modes.Deprecated == "all" || *modes.Deprecated == "out-of-pool", true
	}
	return true, false
}

// Checks if the out-of-pool reservation mode has been enabled.
// Returns (first parameter):
// - reservations-out-of-pool value if set OR,
// - true when reservation-mode is "out-of-pool",
// - false otherwise.
// The second parameter indicates whether the returned value was set
// explicitly (when true) or is a default value (when false).
func (modes *ReservationModes) IsOutOfPool() (bool, bool) {
	if modes.OutOfPool != nil {
		return *modes.OutOfPool, true
	}
	if modes.Deprecated != nil {
		return *modes.Deprecated == "out-of-pool", true
	}
	return false, false
}

// Parses and returns top-level reservation modes.
func (c *Map) GetGlobalReservationModes() *ReservationModes {
	rootNode, ok := c.getRootNode()
	if !ok {
		return nil
	}
	modes := &ReservationModes{}
	_ = decode(rootNode, modes)

	return modes
}

// Hide any sensitive data in the config.
func (c *Map) HideSensitiveData() {
	hideSensitiveData((*map[string]interface{})(c))
}

// Hide any sensitive data in the object. Data is sensitive if its key is equal to "password", "token" or "secret".
func hideSensitiveData(obj *map[string]interface{}) {
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
				subobject, ok := arrayItemValue.(map[string]interface{})
				if ok {
					hideSensitiveData(&subobject)
				}
			}
			continue
		}
		// Check if it is a subobject (but not array).
		subobject, ok := entryValue.(map[string]interface{})
		if ok {
			hideSensitiveData(&subobject)
		}
	}
}

// Convenience function used to check if a given host reservation
// mode has been enabled at one of the levels at which the
// reservation mode can be configured. The reservation modes specified
// using the variadic parameters should be ordered from the lowest to
// highest configuration level, e.g., subnet-level, shared network-level,
// and finally global-level host reservation configuration. The first
// argument is a function implementing a condition to be checked for
// each ReservationModes. The example condition is:
//
//	func (modes ReservationModes) (bool, bool) {
//		return modes.IsOutOfPool()
//	}
//
// The function returns true when the condition function returns
// (true, true) for one of the N-1 reservation modes. If it doesn't,
// it returns true when the last reservation mode returns (true, true)
// or (true, false).
//
// Note that this function handles Kea configuration inheritance scheme.
// It checks for explicitly set values at subnet and shared network levels
// which override the global-level setting. The global-level setting
// applies regardless whether or not it is specified. If it is not
// specified a default value is used.
func IsInAnyReservationModes(condition func(modes ReservationModes) (bool, bool), modes ...ReservationModes) bool {
	for i, mode := range modes {
		cond, explicit := condition(mode)
		if cond && (explicit || i >= len(modes)-1) {
			return true
		}
	}
	return false
}

// Returns a list of configured client classes.
func (c *Map) GetClientClasses() (clientClasses []ClientClass) {
	if classList, ok := c.GetTopLevelList("client-classes"); ok {
		_ = mapstructure.Decode(classList, &clientClasses)
	}
	return
}

// Deletes client classes from the configuration.
func (c *Map) DeleteClientClasses() {
	if node, ok := c.getRootNode(); ok {
		delete(node, "client-classes")
	}
}

// Matches the prefix of a subnet with the given IP network. If the match is
// found the local subnet id of that subnet is returned. Otherwise, the value
// of 0 is returned.
func getMatchingSubnetLocalID(subnet interface{}, ipNet *net.IPNet) int64 {
	sn := subnet.(map[string]interface{})

	// Parse the prefix into a common form that can be used for comparison.
	_, localNetwork, err := net.ParseCIDR(sn["subnet"].(string))
	if err != nil {
		return 0
	}
	// Compare the prefix of the subnet we have found and the specified prefix.
	if (localNetwork != nil) && net.IP.Equal(ipNet.IP, localNetwork.IP) &&
		bytes.Equal(ipNet.Mask, localNetwork.Mask) {
		snID, ok := sn["id"]
		if ok {
			return int64(snID.(float64))
		}
		return int64(0)
	}
	// No match.
	return 0
}

// Scans subnets within the Kea configuration and returns the ID of the subnet having
// the specified prefix.
func (c *Map) GetLocalSubnetID(prefix string) int64 {
	_, globalNetwork, err := net.ParseCIDR(prefix)
	if err != nil || globalNetwork == nil {
		return 0
	}

	// Depending on the DHCP server type, we need to use different name of the list
	// holding the subnets.
	rootName, ok := c.GetRootName()
	if !ok {
		return 0
	}
	var subnetParamName string
	switch rootName {
	case RootNameDHCPv4:
		subnetParamName = "subnet4"
	case RootNameDHCPv6:
		subnetParamName = "subnet6"
	default:
		// If this is neither the DHCPv4 nor DHCPv6 server, there is nothing to do.
		return 0
	}

	// First, let's iterate over the subnets which are not associated with any
	// shared network.
	if subnetList, ok := c.GetTopLevelList(subnetParamName); ok {
		for _, s := range subnetList {
			id := getMatchingSubnetLocalID(s, globalNetwork)
			if id > 0 {
				return id
			}
		}
	}

	// No match. Let's get the subnets belonging to the shared networks.
	if networkList, ok := c.GetTopLevelList("shared-networks"); ok {
		for _, n := range networkList {
			if network, ok := n.(map[string]interface{}); ok {
				if subnetList, ok := network[subnetParamName].([]interface{}); ok {
					for _, s := range subnetList {
						id := getMatchingSubnetLocalID(s, globalNetwork)
						if id > 0 {
							return id
						}
					}
				}
			}
		}
	}

	return 0
}

// Parses shared-networks list into the specified structure. The argument
// must be a pointer to a slice of structures reflecting the shared network
// data.
func (c *Map) DecodeSharedNetworks(decodedSharedNetworks interface{}) error {
	if sharedNetworksList, ok := c.GetTopLevelList("shared-networks"); ok {
		if err := decode(sharedNetworksList, decodedSharedNetworks); err != nil {
			return errors.WithMessage(err, "problem parsing shared-networks")
		}
	}
	return nil
}

// Parses subnet4 or subnet6 list into the specified structure. The argument
// must be a pointer to a slice of structures reflecting the subnet
// data.
func (c *Map) DecodeTopLevelSubnets(decodedSubnets interface{}) error {
	rootName, ok := c.GetRootName()
	if !ok {
		return errors.New("missing root node")
	}
	var subnetsList []interface{}
	switch rootName {
	case "Dhcp4":
		subnetsList, ok = c.GetTopLevelList("subnet4")
	case "Dhcp6":
		subnetsList, ok = c.GetTopLevelList("subnet6")
	default:
		return errors.Errorf("invalid configuration root node %s", rootName)
	}
	if ok {
		if err := decode(subnetsList, decodedSubnets); err != nil {
			return errors.WithMessage(err, "problem parsing subnets")
		}
	}
	return nil
}
