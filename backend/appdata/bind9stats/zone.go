package bind9stats

import (
	"encoding/json"
	"slices"
	"time"

	storkutil "isc.org/stork/util"
)

// A maximum size of the zone collection for which a linear zone search
// is performed. A binary search is be used for larger collections.
const binaryZoneSearchThreshold = 3000

// Represents a BIND9 zone retrieved from the stats channel.
type Zone struct {
	Name   string    `json:"name"`
	Class  string    `json:"class"`
	Serial int64     `json:"serial"`
	Type   string    `json:"type"`
	Loaded time.Time `json:"loaded"`
}

// Represents a collection of zones. It is used internally by the View.
// Therefore it must not be exported. The zones must be instantiated using
// the JSON unmarshaller because it sorts them in the DNS order. If the zones
// list is created by appending the zones to the Zones slice this is not
// guaranteed and therefore the GetZone function may return wrong results.
type zones struct {
	// A slice of zones indexed by zone name.
	zoneList []*Zone
}

// Custom implementation of the JSON unmarshaller for a zone.
func (zones *zones) UnmarshalJSON(data []byte) error {
	zones.zoneList = []*Zone{}
	err := json.Unmarshal(data, &zones.zoneList)
	if err != nil {
		return err
	}
	zones.sort()
	return nil
}

// Custom marshaller for a collection of zones. It outputs the zones
// as a list.
func (zones zones) MarshalJSON() ([]byte, error) {
	return json.Marshal(zones.zoneList)
}

// Sorts zones using DNS order.
func (zones *zones) sort() {
	slices.SortFunc(zones.zoneList, func(zone1, zone2 *Zone) int {
		return storkutil.CompareNames(zone1.Name, zone2.Name)
	})
}

// Returns a list of zone names.
func (zones *zones) GetZoneNames() (zoneNames []string) {
	for _, zone := range zones.zoneList {
		zoneNames = append(zoneNames, zone.Name)
	}
	return
}

// Performs a linear search of the zone by name.
func (zones *zones) linearSearch(name string) *Zone {
	for _, zone := range zones.zoneList {
		if zone.Name == name {
			return zone
		}
	}
	return nil
}

// Performs a binary search of the zone by name. It requires that the
// collection is sorted in DNS order using storkutil.CompareNames function.
func (zones *zones) binarySearch(name string) *Zone {
	if index, ok := slices.BinarySearchFunc(zones.zoneList, name, func(zone1 *Zone, name string) int {
		return storkutil.CompareNames(zone1.Name, name)
	}); ok {
		return zones.zoneList[index]
	}

	return nil
}

// Returns a zone by name. Depending on the size of the collection it performs
// a linear or binary zone search. The linear search is more efficient up to
// thousands of zones. The binary search is more efficient for larger collections.
// The binaryZoneSearchThreshold specifies the maximum size of the collection for
// which a linear search is performed. Since this function can perform a binary
// search it assumes that the zones are sorted using the storkutil.CompareNames
// function.
func (zones *zones) GetZone(name string) *Zone {
	if len(zones.zoneList) < binaryZoneSearchThreshold {
		return zones.linearSearch(name)
	}
	return zones.binarySearch(name)
}
