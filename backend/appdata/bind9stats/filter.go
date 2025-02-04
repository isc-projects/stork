package bind9stats

import (
	"strings"
	"time"
)

// An interface returning a (zone) name.
//
// The use case of this particular interface is to return zone names by
// the objects representing zones in memory and on disk. A file on disk is
// represented by os.DirEntry which exposes the Name() function. The zone
// object implemented in this package also exposes this function so it can
// be hidden behind the NameAccessor interface and accessed by the
// ApplyPagingZoneFilter function.
type NameAccessor interface {
	Name() string
}

// Filters the zones according to the lower bound zone filtering rules. This
// function assumes that the zones are sorted in a DNS order. It returns a
// subslice with the first zone matching the specified name or the next zone
// ordered by the specified name.
func ApplyZoneLowerBoundFilter[TZone NameAccessor](zones []TZone, filter *ZoneFilter) []TZone {
	// There is nothing to do, if filtering is disabled.
	if filter == nil || filter.LowerBound == nil || *filter.LowerBound == "" {
		return zones
	}
	// Mark the beginning of the subslice.
	start, equal := searchZoneLowerBound(zones, *filter.LowerBound)
	if equal {
		start += 1
	}
	if start < 0 || start >= len(zones) {
		return []TZone{}
	}
	return zones[start:]
}

// A structure specifying zones filtering.
type ZoneFilter struct {
	View        *string
	LowerBound  *string
	Limit       *int
	LoadedAfter *time.Time
	Offset      *int
}

// Instantiates zone filter with all filters disabled.
func NewZoneFilter() *ZoneFilter {
	return &ZoneFilter{}
}

// Enables filtering by view name.
func (filter *ZoneFilter) SetView(view string) {
	view = strings.TrimSpace(view)
	filter.View = &view
}

// Enables filtering the zones loaded after specified time.
func (filter *ZoneFilter) SetLoadedAfter(loadedAfter time.Time) {
	filter.LoadedAfter = &loadedAfter
}

// Enables getting the zones page by page.
func (filter *ZoneFilter) SetLowerBound(lowerBound string, limit int) {
	filter.LowerBound = &lowerBound
	filter.Limit = &limit
}

// Sets the offset and limit on the number of zones.
func (filter *ZoneFilter) SetOffsetLimit(offset, limit int) {
	filter.Offset = &offset
	filter.Limit = &limit
}
