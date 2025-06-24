package pdnsdata

import "strconv"

// A structure containing the general server information from the PowerDNS
// server.
type ServerInfo struct {
	Type             string `json:"type"`
	ID               string `json:"id"`
	DaemonType       string `json:"daemon_type"`
	Version          string `json:"version"`
	URL              string `json:"url"`
	ConfigURL        string `json:"config_url"`
	ZonesURL         string `json:"zones_url"`
	AutoprimariesURL string `json:"autoprimaries_url"`
	Uptime           int64
}

// A structure containing a single statistic item from the PowerDNS server.
type StatisticItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Returns the integer value of the statistic item. If the value is not a valid
// integer, 0 is returned.
func (item *StatisticItem) GetInt64() int64 {
	if value, err := strconv.ParseInt(item.Value, 10, 64); err == nil {
		return value
	}
	return 0
}
