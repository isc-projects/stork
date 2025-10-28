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
	*selectorImpl[FilterType]
}

// Creates a new filter with the specified filter types.
func NewFilter(filterTypes ...FilterType) *Filter {
	return &Filter{
		newSelectorImpl[FilterType](filterTypes...),
	}
}

// Creates a new filter from the list of filters in protobuf format.
func NewFilterFromProto(filter *agentapi.ReceiveBind9ConfigFilter) *Filter {
	if filter == nil || len(filter.FilterTypes) == 0 {
		// No filters specified. It means that filtering is disabled.
		return nil
	}
	filterTypes := make([]FilterType, 0, len(filter.FilterTypes))
	for _, filter := range filter.FilterTypes {
		switch filter {
		case agentapi.ReceiveBind9ConfigFilter_CONFIG:
			filterTypes = append(filterTypes, FilterTypeConfig)
		case agentapi.ReceiveBind9ConfigFilter_VIEW:
			filterTypes = append(filterTypes, FilterTypeView)
		case agentapi.ReceiveBind9ConfigFilter_ZONE:
			filterTypes = append(filterTypes, FilterTypeZone)
		case agentapi.ReceiveBind9ConfigFilter_NO_PARSE:
			filterTypes = append(filterTypes, FilterTypeNoParse)
		}
	}
	// Filtering enabled.
	return NewFilter(filterTypes...)
}

// Enables the specified filter type.
func (f *Filter) Enable(filterType FilterType) {
	f.enable(filterType)
}

// Checks if the specified filter type is enabled.
func (f *Filter) IsEnabled(filterType FilterType) bool {
	return f == nil || f.isEnabled(filterType)
}

// Returns the list of enabled filter types.
func (f *Filter) GetFilterTypes() []FilterType {
	return f.getItems()
}

// Convenience function returning a list of filters in protobuf format.
func (f *Filter) GetFilterAsProto() *agentapi.ReceiveBind9ConfigFilter {
	var filter *agentapi.ReceiveBind9ConfigFilter
	if f != nil {
		filter = &agentapi.ReceiveBind9ConfigFilter{
			FilterTypes: make([]agentapi.ReceiveBind9ConfigFilter_FilterType, 0, len(f.getItems())),
		}
		for _, filterType := range f.getItems() {
			switch filterType {
			case FilterTypeConfig:
				filter.FilterTypes = append(filter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_CONFIG)
			case FilterTypeView:
				filter.FilterTypes = append(filter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_VIEW)
			case FilterTypeZone:
				filter.FilterTypes = append(filter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_ZONE)
			case FilterTypeNoParse:
				filter.FilterTypes = append(filter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_NO_PARSE)
			}
		}
	}
	return filter
}

// File types supported by the BIND 9 configuration.
type FileType string

const (
	FileTypeConfig  FileType = "config"
	FileTypeRndcKey FileType = "rndc-key"
)

// Selector for the BIND 9 configuration file types.
type FileTypeSelector struct {
	*selectorImpl[FileType]
}

// Instantiates a new file type selector with the specified file types.
func NewFileTypeSelector(fileTypes ...FileType) *FileTypeSelector {
	return &FileTypeSelector{
		newSelectorImpl(fileTypes...),
	}
}

// Enables the specified file type.
func (s *FileTypeSelector) Enable(fileType FileType) {
	s.enable(fileType)
}

// Checks if the specified file type is enabled.
func (s *FileTypeSelector) IsEnabled(fileType FileType) bool {
	return s == nil || s.isEnabled(fileType)
}

// Returns the list of enabled file types.
func (s *FileTypeSelector) GetFileTypes() []FileType {
	return s.getItems()
}

// Convenience function returning a list of file types in protobuf format.
func (s *FileTypeSelector) GetFileTypesAsProto() *agentapi.ReceiveBind9ConfigFileSelector {
	var fileSelector *agentapi.ReceiveBind9ConfigFileSelector
	if s != nil {
		fileSelector = &agentapi.ReceiveBind9ConfigFileSelector{
			FileTypes: make([]agentapi.Bind9ConfigFileType, 0, len(s.getItems())),
		}
		for _, fileType := range s.getItems() {
			switch fileType {
			case FileTypeConfig:
				fileSelector.FileTypes = append(fileSelector.FileTypes, agentapi.Bind9ConfigFileType_CONFIG)
			case FileTypeRndcKey:
				fileSelector.FileTypes = append(fileSelector.FileTypes, agentapi.Bind9ConfigFileType_RNDC_KEY)
			}
		}
	}
	return fileSelector
}

// Internal implementation of the selector used by the concrete
// configuration filters and file type selectors.
type selectorImpl[T comparable] struct {
	items []T
}

// Instantiates a new selector with the specified items.
func newSelectorImpl[T comparable](items ...T) *selectorImpl[T] {
	return &selectorImpl[T]{
		items: items,
	}
}

// Enables the specified item.
func (s *selectorImpl[T]) enable(item T) {
	if !slices.Contains(s.items, item) {
		s.items = append(s.items, item)
	}
}

// Checks if the specified item is enabled.
func (s *selectorImpl[T]) isEnabled(item T) bool {
	return len(s.items) == 0 || slices.Contains(s.items, item)
}

// Returns the list of enabled items.
func (s *selectorImpl[T]) getItems() []T {
	return s.items
}
