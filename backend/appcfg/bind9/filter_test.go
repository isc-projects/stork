package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	agentapi "isc.org/stork/api"
)

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoConfig(t *testing.T) {
	protoFilter := &agentapi.ReceiveBind9ConfigFilter{
		FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{
			agentapi.ReceiveBind9ConfigFilter_CONFIG,
		},
	}
	filter := NewFilterFromProto(protoFilter)
	require.NotNil(t, filter)
	require.True(t, filter.IsEnabled(FilterTypeConfig))
	require.False(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoView(t *testing.T) {
	protoFilter := &agentapi.ReceiveBind9ConfigFilter{
		FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{
			agentapi.ReceiveBind9ConfigFilter_VIEW,
		},
	}
	filter := NewFilterFromProto(protoFilter)
	require.NotNil(t, filter)
	require.False(t, filter.IsEnabled(FilterTypeConfig))
	require.True(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoZone(t *testing.T) {
	protoFilter := &agentapi.ReceiveBind9ConfigFilter{
		FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{
			agentapi.ReceiveBind9ConfigFilter_ZONE,
		},
	}
	filter := NewFilterFromProto(protoFilter)
	require.NotNil(t, filter)
	require.False(t, filter.IsEnabled(FilterTypeConfig))
	require.False(t, filter.IsEnabled(FilterTypeView))
	require.True(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoNoParse(t *testing.T) {
	protoFilter := &agentapi.ReceiveBind9ConfigFilter{
		FilterTypes: []agentapi.ReceiveBind9ConfigFilter_FilterType{
			agentapi.ReceiveBind9ConfigFilter_NO_PARSE,
		},
	}
	filter := NewFilterFromProto(protoFilter)
	require.NotNil(t, filter)
	require.False(t, filter.IsEnabled(FilterTypeConfig))
	require.False(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.True(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test that the filter is enabled when no filter types are specified.
func TestFilterIsEnabledNone(t *testing.T) {
	filter := NewFilter()
	filterTypes := []FilterType{FilterTypeConfig, FilterTypeView, FilterTypeZone, FilterTypeNoParse}
	for _, filterType := range filterTypes {
		require.True(t, filter.IsEnabled(filterType))
	}
}

// Test that the filter is enabled when the filter is nil.
func TestFilterNil(t *testing.T) {
	var filter *Filter
	filterTypes := []FilterType{FilterTypeConfig, FilterTypeView, FilterTypeZone, FilterTypeNoParse}
	for _, filterType := range filterTypes {
		require.True(t, filter.IsEnabled(filterType))
	}
}

// Test that the filter can be selectively enabled for specific filter types.
func TestFilterIsEnabled(t *testing.T) {
	filter := NewFilter(FilterTypeConfig, FilterTypeView)
	require.True(t, filter.IsEnabled(FilterTypeConfig))
	require.True(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test that the filter can be selectively enabled for specific filter types.
func TestFilterEnable(t *testing.T) {
	filter := NewFilter()
	filter.Enable(FilterTypeConfig)
	filter.Enable(FilterTypeView)
	require.True(t, filter.IsEnabled(FilterTypeConfig))
	require.True(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test getting the list of enabled filter types.
func TestFilterGetFilterTypes(t *testing.T) {
	filter := NewFilter(FilterTypeConfig, FilterTypeView)
	filterTypes := filter.GetFilterTypes()
	require.Equal(t, 2, len(filterTypes))
	require.Contains(t, filterTypes, FilterTypeConfig)
	require.Contains(t, filterTypes, FilterTypeView)
}

// Test that the filter can be converted to a list of filters in protobuf format.
// In this case the filter contains both config and view filters.
func TestFilterGetFilterAsProtoConfigAndView(t *testing.T) {
	filter := NewFilter(FilterTypeConfig, FilterTypeView)
	protoFilter := filter.GetFilterAsProto()
	require.NotNil(t, protoFilter)
	require.Equal(t, 2, len(protoFilter.FilterTypes))
	require.Contains(t, protoFilter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_CONFIG)
	require.Contains(t, protoFilter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_VIEW)
}

// Test that the filter can be converted to a list of filters in protobuf format.
// In this case the filter contains both view and zone filters.
func TestFilterGetFilterAsProtoViewAndZone(t *testing.T) {
	filter := NewFilter(FilterTypeView, FilterTypeZone)
	protoFilter := filter.GetFilterAsProto()
	require.NotNil(t, protoFilter)
	require.Equal(t, 2, len(protoFilter.FilterTypes))
	require.Contains(t, protoFilter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_VIEW)
	require.Contains(t, protoFilter.FilterTypes, agentapi.ReceiveBind9ConfigFilter_ZONE)
}

// Test creating a new file selector from the file types specified in the .proto file.
func TestNewFileTypeSelectorWithConfig(t *testing.T) {
	selector := NewFileTypeSelector(FileTypeConfig)
	require.NotNil(t, selector)
	require.True(t, selector.IsEnabled(FileTypeConfig))
	require.False(t, selector.IsEnabled(FileTypeRndcKey))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFileTypeSelectorWithRndcKey(t *testing.T) {
	selector := NewFileTypeSelector(FileTypeRndcKey)
	require.NotNil(t, selector)
	require.True(t, selector.IsEnabled(FileTypeRndcKey))
	require.False(t, selector.IsEnabled(FileTypeConfig))
}

// Test that the file selector is enabled when no file types are specified.
func TestFileTypeSelectorIsEnabledNone(t *testing.T) {
	selector := NewFileTypeSelector()
	require.True(t, selector.IsEnabled(FileTypeConfig))
	require.True(t, selector.IsEnabled(FileTypeRndcKey))
}

// Test that the filter is enabled when the file type selector is nil.
func TestFileTypeSelectorNil(t *testing.T) {
	var selector *FileTypeSelector
	require.True(t, selector.IsEnabled(FileTypeConfig))
	require.True(t, selector.IsEnabled(FileTypeRndcKey))
}

// Test that the file selector can be selectively enabled for specific file types.
func TestFileTypeSelectorIsEnabled(t *testing.T) {
	selector := NewFileTypeSelector(FileTypeConfig, FileTypeRndcKey)
	require.True(t, selector.IsEnabled(FileTypeConfig))
	require.True(t, selector.IsEnabled(FileTypeRndcKey))
}

// Test that the file selector can be selectively enabled for specific file types.
func TestFileTypeSelectorEnable(t *testing.T) {
	selector := NewFileTypeSelector()
	selector.Enable(FileTypeConfig)
	selector.Enable(FileTypeRndcKey)
	require.True(t, selector.IsEnabled(FileTypeConfig))
	require.True(t, selector.IsEnabled(FileTypeRndcKey))
}

// Test getting the list of enabled filter types.
func TestFileTypeSelectorGetFileTypes(t *testing.T) {
	selector := NewFileTypeSelector(FileTypeConfig, FileTypeRndcKey)
	fileTypes := selector.GetFileTypes()
	require.Equal(t, 2, len(fileTypes))
	require.Contains(t, fileTypes, FileTypeConfig)
	require.Contains(t, fileTypes, FileTypeRndcKey)
}

// Test that the file selector can be converted to a list of file types in protobuf format.
// In this case the file selector contains both config and rndc key file types.
func TestFileTypeSelectorGetFileTypesAsProtoConfigAndRndcKey(t *testing.T) {
	selector := NewFileTypeSelector(FileTypeConfig, FileTypeRndcKey)
	protoSelector := selector.GetFileTypesAsProto()
	require.NotNil(t, protoSelector)
	require.Equal(t, 2, len(protoSelector.FileTypes))
	require.Contains(t, protoSelector.FileTypes, agentapi.Bind9ConfigFileType_CONFIG)
	require.Contains(t, protoSelector.FileTypes, agentapi.Bind9ConfigFileType_RNDC_KEY)
}

// Test that the file selector can be converted to a list of file types in protobuf format.
// In this case the filter contains both view and zone filters.
func TestFileTypeSelectorGetFileTypesAsProtoRndcKey(t *testing.T) {
	selector := NewFileTypeSelector(FileTypeRndcKey)
	protoSelector := selector.GetFileTypesAsProto()
	require.NotNil(t, protoSelector)
	require.Equal(t, 1, len(protoSelector.FileTypes))
	require.Equal(t, agentapi.Bind9ConfigFileType_RNDC_KEY, protoSelector.FileTypes[0])
}
