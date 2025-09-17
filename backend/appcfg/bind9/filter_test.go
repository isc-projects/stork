package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
