package dbmodel

// A structure reflecting the access_point SQL table.
type AccessPoint struct {
	AppID             int64  `pg:",pk"`
	Type              string `pg:",pk"`
	MachineID         int64
	Address           string
	Port              int64
	Key               string
	UseSecureProtocol bool `pg:",use_zero"`
}

const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

// AppendAccessPoint is an utility function that appends an access point to a
// list.
func AppendAccessPoint(list []*AccessPoint, tp, address, key string, port int64, useSecureProtocol bool) []*AccessPoint {
	list = append(list, &AccessPoint{
		Type:              tp,
		Address:           address,
		Port:              port,
		Key:               key,
		UseSecureProtocol: useSecureProtocol,
	})
	return list
}
