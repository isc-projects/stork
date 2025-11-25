package bind9stats

import (
	"encoding/json"
	"iter"
	"slices"
	"strings"
)

var _ ZoneIteratorAccessor = (*View)(nil)

// Represents a BIND9 view retrieved from the stats channel.
type View struct {
	// View name.
	Name string `json:"-"`
	// List of zones.
	Zones zones `json:"zones"`
}

// An interface returning view name and iterator to zones belonging to
// this view. It should be implemented by the structures representing
// views in memory or on disk.
type ZoneIteratorAccessor interface {
	GetViewName() string
	GetZoneCount() (int64, error)
	GetZoneIterator(filter *ZoneFilter) iter.Seq2[*Zone, error]
}

// Instantiates the view from the list of zones.
func NewView(viewName string, zones []*Zone) *View {
	view := &View{
		Name:  viewName,
		Zones: *newZones(zones),
	}
	return view
}

// Returns view name.
func (view *View) GetViewName() string {
	return view.Name
}

// Returns a list of zone names.
func (view *View) GetZoneNames() []string {
	return view.Zones.GetZoneNames()
}

// Returns the number of zones in the view.
func (view *View) GetZoneCount() (int64, error) {
	return view.Zones.GetZoneCount(), nil
}

// Returns a zone by name.
func (view *View) GetZone(name string) *Zone {
	return view.Zones.GetZone(name)
}

// Returns zones list.
func (view *View) GetZones() []*Zone {
	return view.Zones.zoneList
}

// Returns iterator to zones with optional filtering.
func (view *View) GetZoneIterator(filter *ZoneFilter) iter.Seq2[*Zone, error] {
	zones := ApplyZoneLowerBoundFilter(view.Zones.zoneList, filter)
	return func(yield func(*Zone, error) bool) {
		var count int
		for _, zone := range zones {
			if filter != nil {
				if filter.LoadedAfter != nil && !zone.Loaded.After(*filter.LoadedAfter) {
					continue
				}
				if filter.Limit != nil {
					count++
					if count > *filter.Limit {
						return
					}
				}
			}
			if !yield(zone, nil) {
				return
			}
		}
	}
}

// Assigns collection of zones to a view.
func (view *View) SetZones(zones []*Zone) {
	view.Zones.zoneList = zones
	view.Zones.sort()
}

// Represents a collection of views.
type Views struct {
	// Ordered collection of views.
	Views []*View
}

// Instantiates views from a slice with sorting.
func NewViews(viewsList []*View) *Views {
	views := &Views{
		Views: viewsList,
	}
	views.sort()
	return views
}

// Sorts views by name.
func (views *Views) sort() {
	slices.SortFunc(views.Views, func(view1, view2 *View) int {
		return strings.Compare(view1.Name, view2.Name)
	})
}

// Custom implementation of the JSON unmarshaller for a view. It sorts the
// views by name.
func (views *Views) UnmarshalJSON(data []byte) error {
	viewsMap := make(map[string]*View)
	err := json.Unmarshal(data, &viewsMap)
	if err != nil {
		return err
	}
	for viewName, view := range viewsMap {
		view.Name = viewName
		views.Views = append(views.Views, view)
	}
	// Ensure the views are sorted by name.
	views.sort()
	return nil
}

// Custom marshaller for a collection of views. It outputs the views
// as a map keyed with view names.
func (views Views) MarshalJSON() ([]byte, error) {
	viewsMap := make(map[string]any)
	for _, view := range views.Views {
		viewsMap[view.Name] = view
	}
	return json.Marshal(viewsMap)
}

// Returns a list of view names.
func (views *Views) GetViewNames() (viewNames []string) {
	for _, view := range views.Views {
		viewNames = append(viewNames, view.Name)
	}
	return
}

// Returns zone count in all views.
func (views *Views) GetZoneCount() int64 {
	var zoneCount int64
	for _, view := range views.Views {
		currentCount, _ := view.GetZoneCount()
		zoneCount += currentCount
	}
	return zoneCount
}

// Returns a view by name.
func (views *Views) GetView(name string) *View {
	if index := slices.IndexFunc(views.Views, func(view *View) bool {
		return view.Name == name
	}); index >= 0 {
		return views.Views[index]
	}
	return nil
}
