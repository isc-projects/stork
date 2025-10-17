package bind9config

import (
	"slices"

	agentapi "isc.org/stork/api"
)

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

// Creates a new filter from the list of filters in protobuf format.
func NewFilterFromProto(filters []*agentapi.GetBind9ConfigFilter) *Filter {
	if len(filters) == 0 {
		// No filters specified. It means that filtering is disabled.
		return nil
	}
	filterTypes := make([]FilterType, 0, len(filters))
	for _, filter := range filters {
		switch filter.FilterType {
		case agentapi.GetBind9ConfigFilter_CONFIG:
			filterTypes = append(filterTypes, FilterTypeConfig)
		case agentapi.GetBind9ConfigFilter_VIEW:
			filterTypes = append(filterTypes, FilterTypeView)
		case agentapi.GetBind9ConfigFilter_ZONE:
			filterTypes = append(filterTypes, FilterTypeZone)
		case agentapi.GetBind9ConfigFilter_NO_PARSE:
			filterTypes = append(filterTypes, FilterTypeNoParse)
		}
	}
	// Filtering enabled.
	return NewFilter(filterTypes...)
}

// Enables the specified filter type.
func (f *Filter) Enable(filterType FilterType) {
	f.filterTypes = append(f.filterTypes, filterType)
}

// Checks if the specified filter type is enabled.
func (f *Filter) IsEnabled(filterType FilterType) bool {
	return f == nil || slices.Contains(f.filterTypes, filterType)
}

// Returns the list of enabled filter types.
func (f *Filter) GetFilterTypes() []FilterType {
	return f.filterTypes
}

// Convenience function returning a list of filters in protobuf format.
func (f *Filter) GetFilterAsProto() []*agentapi.GetBind9ConfigFilter {
	filters := []*agentapi.GetBind9ConfigFilter{}
	if f != nil {
		for _, filterType := range f.filterTypes {
			switch filterType {
			case FilterTypeConfig:
				filters = append(filters, &agentapi.GetBind9ConfigFilter{FilterType: agentapi.GetBind9ConfigFilter_CONFIG})
			case FilterTypeView:
				filters = append(filters, &agentapi.GetBind9ConfigFilter{FilterType: agentapi.GetBind9ConfigFilter_VIEW})
			case FilterTypeZone:
				filters = append(filters, &agentapi.GetBind9ConfigFilter{FilterType: agentapi.GetBind9ConfigFilter_ZONE})
			case FilterTypeNoParse:
				filters = append(filters, &agentapi.GetBind9ConfigFilter{FilterType: agentapi.GetBind9ConfigFilter_NO_PARSE})
			}
		}
	}
	return filters
}
