package bind9config

import "slices"

// Parsed BIND 9 configuration can be serialized into a string representation.
// Often it is desired to output only a subset of the configuration elements
// because DNS configurations can grow large. The filter can be used to select
// specific configuration elements. The corresponding element names are specified
// using the filter tag in the configuration structures. For example, the
// Statement struct includes a filter tag for each field. This tag associates the
// fields with the filter types defined below. If the specified filter types include
// the filter name specified in the tag, the field is serialized and included in
// the output. Otherwise, it is omitted. If the filter is nil, all fields are
// returned by default. If the filter tag is not specified for the field, the
// field is always serialized and returned.
type FilterType string

const (
	// Returns server configuration statements (e.g., global options, logging options).
	FilterTypeConfig FilterType = "config"
	// Returns view statements. The view excludes zone statements unless the zone
	// filter is also enabled.
	FilterTypeView FilterType = "view"
	// Returns zone statements.
	FilterTypeZone FilterType = "zone"
	// Returns no-parse directives and their contents.
	FilterTypeNoParse FilterType = "no-parse"
)

// The filter structure allows for selecting specific configuration elements
// to be serialized and returned.
type Filter struct {
	filterTypes []FilterType
}

// Creates a new filter with the specified filter types.
func NewFilter(filterTypes ...FilterType) *Filter {
	return &Filter{
		filterTypes: filterTypes,
	}
}

// Checks if the specified filter type is enabled.
func (f *Filter) IsEnabled(filterType FilterType) bool {
	return f == nil || slices.Contains(f.filterTypes, filterType)
}
