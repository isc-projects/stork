package bind9stats

import (
	"encoding/json"
	"slices"
	"time"

	storkutil "isc.org/stork/util"
)

var _ NameAccessor = (*Zone)(nil)

// A maximum size of the zone collection for which a linear zone search
// is performed. A binary search is used for larger collections.
const binaryZoneSearchThreshold = 3000

// Represents a DNS zone information retrieved from the stats channel.
type Zone struct {
	ZoneName string    `json:"name"`
	Class    string    `json:"class"`
	Serial   int64     `json:"serial"`
	Type     string    `json:"type"`
	Loaded   time.Time `json:"loaded"`
}

// Implements NameAccessor interface and returns zone name.
func (zone *Zone) Name() string {
	return zone.ZoneName
}

// Base zone information combined with additional metadata describing the zone.
// The metadata are optional and depend on the use case.
type ExtendedZone struct {
	Zone
	ViewName       string
	TotalZoneCount int64
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

// Instantiates the zones from a list.
func newZones(zonesList []*Zone) *zones {
	zones := &zones{
		zoneList: zonesList,
	}
	zones.sort()
	return zones
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
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
}

// Returns a list of zone names.
func (zones *zones) GetZoneNames() (zoneNames []string) {
	for _, zone := range zones.zoneList {
		zoneNames = append(zoneNames, zone.Name())
	}
	return
}

// Returns the number of zones.
func (zones *zones) GetZoneCount() int64 {
	return int64(len(zones.zoneList))
}

// Returns a zone by name using linear or binary search.
func (zones *zones) GetZone(name string) *Zone {
	if index := searchZone(zones.zoneList, name); index >= 0 {
		return zones.zoneList[index]
	}
	return nil
}

// Performs a linear search of the zone by name.
func linearSearchZone[ZoneI NameAccessor](zones []ZoneI, name string) int {
	for index, zone := range zones {
		if zone.Name() == name {
			return index
		}
	}
	return -1
}

// Returns a zone index by name. Depending on the size of the collection it performs
// a linear or binary zone search. The linear search is more efficient up to
// thousands of zones. The binary search is more efficient for larger collections.
// The binaryZoneSearchThreshold specifies the maximum size of the collection for
// which a linear search is performed. Since this function can perform a binary
// search it assumes that the zones are sorted using the storkutil.CompareNames
// function.
func searchZone[ZoneI NameAccessor](zones []ZoneI, name string) int {
	if len(zones) < binaryZoneSearchThreshold {
		return linearSearchZone(zones, name)
	}
	return binarySearchZone(zones, name)
}

// Finds a zone lower bound in the zone collection by name. Depending on the size
// of the collection it performs a linear or binary zone search. The linear search
// is more efficient up to thousands of zones. The binary search is more efficient
// for larger collections. The binaryZoneSearchThreshold specifies the maximum size
// of the collection for which a linear search is performed. Since this function
// can perform a binary search it assumes that the zones are sorted using the
// storkutil.CompareNames function.
func searchZoneLowerBound[ZoneI NameAccessor](zones []ZoneI, name string) (int, bool) {
	if len(zones) < binaryZoneSearchThreshold {
		return linearSearchZoneLowerBound(zones, name)
	}
	return binarySearchZoneLowerBound(zones, name)
}

// Finds lower bound zone in the zones collection by name using linear search.
// It returns an index of -1 when the lower bound does not exist. Otherwise it
// returns a lower bound index. The index may point to a zone having the specified
// name or the first zone ordered after this name if the zone with the specified
// name does not exist. In the former case the second returned parameter is true.
// In the latter case, it is set to false.
func linearSearchZoneLowerBound[ZoneI NameAccessor](zones []ZoneI, name string) (int, bool) {
	for index, zone := range zones {
		switch storkutil.CompareNames(zone.Name(), name) {
		case 0:
			// Found the zone with the specified name.
			return index, true
		case 1:
			// Did not found the zone with the specified name but the next
			// one ordered after this name.
			return index, false
		default:
			// Didn't found the lower bound yet.
			continue
		}
	}
	return -1, false
}

// Performs a binary search of the zone by name. It requires that the
// collection is sorted in DNS order using storkutil.CompareNames function.
func binarySearchZone[ZoneI NameAccessor](zones []ZoneI, name string) int {
	if index, ok := slices.BinarySearchFunc(zones, name, func(zone1 ZoneI, name string) int {
		return storkutil.CompareNames(zone1.Name(), name)
	}); ok {
		return index
	}
	return -1
}

// Finds lower bound zone in the zones collection by name using binary search.
// It returns an index of -1 when the lower bound does not exist. Otherwise it
// returns a lower bound index. The index may point to a zone having the specified
// name or the first zone ordered after this name if the zone with the specified
// name does not exist. In the former case the second returned parameter is true.
// In the latter case, it is set to false.
func binarySearchZoneLowerBound[ZoneI NameAccessor](zones []ZoneI, name string) (int, bool) {
	if index, ok := slices.BinarySearchFunc(zones, name, func(zone1 ZoneI, name string) int {
		// Found the lower bound when the current zone name is equal or later than
		// the specified zone name.
		if comparison := storkutil.CompareNames(zone1.Name(), name); comparison >= 0 {
			return 0
		}
		return -1
	}); ok {
		return index, zones[index].Name() == name
	}
	return -1, false
}
