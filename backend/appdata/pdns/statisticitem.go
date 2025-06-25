package pdnsdata

import (
	"encoding/json"
	"strconv"
)

// A type of the statistic item: StatisticItem, MapStatisticItem, RingStatisticItem.
// See: https://doc.powerdns.com/authoritative/http-api/statistics.html?highlight=statistics#objects
type StatisticItemType string

const (
	StatisticItem     StatisticItemType = "StatisticItem"
	MapStatisticItem  StatisticItemType = "MapStatisticItem"
	RingStatisticItem StatisticItemType = "RingStatisticItem"
)

// A structure containing a single statistic item from the PowerDNS server.
type AnyStatisticItem struct {
	Name  string            `json:"name"`
	Size  string            `json:"size,omitempty"`
	Type  StatisticItemType `json:"type,omitempty"`
	Value json.RawMessage   `json:"value"`
}

// Returns the integer value of the statistic item. If the value is not a valid
// integer, 0 is returned.
func (item *AnyStatisticItem) GetInt64() int64 {
	var strValue string
	if err := json.Unmarshal(item.Value, &strValue); err == nil {
		if value, err := strconv.ParseInt(strValue, 10, 64); err == nil {
			return value
		}
	}
	return 0
}
