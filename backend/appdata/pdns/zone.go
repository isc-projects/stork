package pdnsdata

import (
	"encoding/json"
	"iter"
	"slices"

	storkutil "isc.org/stork/util"
)

// Represents a DNS zone information retrieved from the PowerDNS.
type Zone struct {
	ZoneName string `json:"name"`
	Serial   int64  `json:"serial"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
}

// Implements NameAccessor interface and returns zone name.
func (zone *Zone) Name() string {
	return zone.ZoneName
}

// Represents a collection of Zones. The Zones must be instantiated using
// the JSON unmarshaller because it sorts them in the DNS order. If the Zones
// list is created by appending the Zones to the Zones slice this is not
// guaranteed and therefore the GetZone function may return wrong results.
type Zones struct {
	// A slice of zones indexed by zone name.
	zoneList []*Zone
}

// Instantiates the zones from a list.
func NewZones(zonesList []*Zone) *Zones {
	zones := &Zones{
		zoneList: zonesList,
	}
	zones.sort()
	return zones
}

// Returns iterator to zones.
func (zones *Zones) GetIterator() iter.Seq[*Zone] {
	return func(yield func(*Zone) bool) {
		for _, zone := range zones.zoneList {
			if !yield(zone) {
				return
			}
		}
	}
}

// Custom implementation of the JSON unmarshaller for zones. It sorts
// the zones by name.
func (zones *Zones) UnmarshalJSON(data []byte) error {
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
func (zones Zones) MarshalJSON() ([]byte, error) {
	return json.Marshal(zones.zoneList)
}

// Sorts zones using DNS order.
func (zones *Zones) sort() {
	slices.SortFunc(zones.zoneList, func(zone1, zone2 *Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
}
