package dbmodel

// Represents address pool structure within Kea configuration.
type KeaConfigPool struct {
	Pool string
}

// Represents prefix delegation pool structure within Kea configuration.
type KeaConfigPdPool struct {
	Prefix       string
	DelegatedLen int `mapstructure:"delegated-len"`
}

// Represents a subnet with pools within Kea configuration.
type KeaConfigSubnet struct {
	ID      int64
	Subnet  string
	ClientClass    string `mapstructure:"client-class"`
	Pools   []KeaConfigPool
	PdPools []KeaConfigPdPool `mapstructure:"pd-pools"`
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
