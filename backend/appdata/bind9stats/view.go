package bind9stats

import (
	"encoding/json"
	"slices"
	"strings"
)

// Represents a BIND9 view retrieved from the stats channel.
type View struct {
	// View name.
	Name string `json:"-"`
	// List of zones.
	Zones zones `json:"zones"`
}

// Returns a list of zone names.
func (view *View) GetZoneNames() []string {
	return view.Zones.GetZoneNames()
}

// Returns a zone by name.
func (view *View) GetZone(name string) *Zone {
	return view.Zones.GetZone(name)
}

// Represents a collection of views.
type Views struct {
	// Ordered collection of views.
	Views []*View
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
	slices.SortFunc(views.Views, func(view1, view2 *View) int {
		return strings.Compare(view1.Name, view2.Name)
	})
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

// Returns a view by name.
func (views *Views) GetView(name string) *View {
	if index := slices.IndexFunc(views.Views, func(view *View) bool {
		return view.Name == name
	}); index >= 0 {
		return views.Views[index]
	}
	return nil
}
