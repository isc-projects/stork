package dbmodel

// Represents host reservation within Kea configuration.
type KeaConfigReservation struct {
	HWAddress   string   `mapstructure:"hw-address" json:"hw-address,omitempty"`
	DUID        string   `mapstructure:"duid" json:"duid,omitempty"`
	CircuitID   string   `mapstructure:"circuit-id" json:"circuit-id,omitempty"`
	ClientID    string   `mapstructure:"client-id" json:"client-id,omitempty"`
	FlexID      string   `mapstructure:"flex-id" json:"flex-id,omitempty"`
	IPAddress   string   `mapstructure:"ip-address" json:"ip-address,omitempty"`
	IPAddresses []string `mapstructure:"ip-addresses" json:"ip-addresses,omitempty"`
	Prefixes    []string `mapstructure:"prefixes" json:"prefixes,omitempty"`
	Hostname    string   `mapstructure:"hostname" json:"hostname,omitempty"`
}

// Represents address pool structure within Kea configuration.
type KeaConfigPool struct {
	Pool string
}

// Represents prefix delegation pool structure within Kea configuration.
type KeaConfigPdPool struct {
	Prefix       string
	PrefixLen    int `mapstructure:"prefix-len"`
	DelegatedLen int `mapstructure:"delegated-len"`
}

// Represents a subnet with pools within Kea configuration.
type KeaConfigSubnet struct {
	ID           int64
	Subnet       string
	ClientClass  string `mapstructure:"client-class"`
	Pools        []KeaConfigPool
	PdPools      []KeaConfigPdPool `mapstructure:"pd-pools"`
	Reservations []KeaConfigReservation
}

// Represents a shared network with subnets within Kea configuration.
type KeaConfigSharedNetwork struct {
	Name    string
	Subnet4 []KeaConfigSubnet
	Subnet6 []KeaConfigSubnet
}

// Represents a subnet retrieved from database from app table,
// form config json field.
type KeaSubnet struct {
	ID             int
	AppID          int
	Subnet         string
	Pools          []map[string]interface{}
	SharedNetwork  string
	MachineAddress string
	AgentPort      int64
}

// Represents a shared network retrieved from database from app table,
// from config json field.
type KeaSharedNetwork struct {
	Name           string
	AppID          int
	Subnets        []map[string]interface{}
	MachineAddress string
	AgentPort      int64
}
