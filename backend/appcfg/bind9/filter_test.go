package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	agentapi "isc.org/stork/api"
)

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoConfig(t *testing.T) {
	filters := []*agentapi.GetBind9ConfigFilter{
		{FilterType: agentapi.GetBind9ConfigFilter_CONFIG},
	}
	filter := NewFilterFromProto(filters)
	require.NotNil(t, filter)
	require.True(t, filter.IsEnabled(FilterTypeConfig))
	require.False(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoView(t *testing.T) {
	filters := []*agentapi.GetBind9ConfigFilter{
		{FilterType: agentapi.GetBind9ConfigFilter_VIEW},
	}
	filter := NewFilterFromProto(filters)
	require.NotNil(t, filter)
	require.False(t, filter.IsEnabled(FilterTypeConfig))
	require.True(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoZone(t *testing.T) {
	filters := []*agentapi.GetBind9ConfigFilter{
		{FilterType: agentapi.GetBind9ConfigFilter_ZONE},
	}
	filter := NewFilterFromProto(filters)
	require.NotNil(t, filter)
	require.False(t, filter.IsEnabled(FilterTypeConfig))
	require.False(t, filter.IsEnabled(FilterTypeView))
	require.True(t, filter.IsEnabled(FilterTypeZone))
	require.False(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test creating a new filter from the filter types specified in the .proto file.
func TestNewFilterFromProtoNoParse(t *testing.T) {
	filters := []*agentapi.GetBind9ConfigFilter{
		{FilterType: agentapi.GetBind9ConfigFilter_NO_PARSE},
	}
	filter := NewFilterFromProto(filters)
	require.NotNil(t, filter)
	require.False(t, filter.IsEnabled(FilterTypeConfig))
	require.False(t, filter.IsEnabled(FilterTypeView))
	require.False(t, filter.IsEnabled(FilterTypeZone))
	require.True(t, filter.IsEnabled(FilterTypeNoParse))
}

// Test that the filter is disabled when no filter types are specified.
func TestFilterIsEnabledNone(t *testing.T) {
	filter := NewFilter()
	filterTypes := []FilterType{FilterTypeConfig, FilterTypeView, FilterTypeZone, FilterTypeNoParse}
	for _, filterType := range filterTypes {
		require.False(t, filter.IsEnabled(filterType))
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
	require.Equal(t, FilterTypeConfig, filterTypes[0])
	require.Equal(t, FilterTypeView, filterTypes[1])
}

// Test that the filter can be converted to a list of filters in protobuf format.
// In this case the filter contains both config and view filters.
func TestFilterGetFilterAsProtoConfigAndView(t *testing.T) {
	filter := NewFilter(FilterTypeConfig, FilterTypeView)
	filters := filter.GetFilterAsProto()
	require.Equal(t, 2, len(filters))
	require.Equal(t, agentapi.GetBind9ConfigFilter_CONFIG, filters[0].FilterType)
	require.Equal(t, agentapi.GetBind9ConfigFilter_VIEW, filters[1].FilterType)
}

// Test that the filter can be converted to a list of filters in protobuf format.
// In this case the filter contains both view and zone filters.
func TestFilterGetFilterAsProtoViewAndZone(t *testing.T) {
	filter := NewFilter(FilterTypeView, FilterTypeZone)
	filters := filter.GetFilterAsProto()
	require.Equal(t, 2, len(filters))
	require.Equal(t, agentapi.GetBind9ConfigFilter_VIEW, filters[0].FilterType)
	require.Equal(t, agentapi.GetBind9ConfigFilter_ZONE, filters[1].FilterType)
}
