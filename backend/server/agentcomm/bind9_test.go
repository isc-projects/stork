package agentcomm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that named statistics-channel response can be parsed.
func TestUnmarshalNamedStats(t *testing.T) {
	response := `{
            "json-stats-version": "1.2",
            "views": {
                "_default": {
                    "resolver": {
                        "cachestats": {
                            "CacheHits": 50,
                            "CacheMisses": 10,
                            "QueryHits": 0,
                            "QueryMisses": 0
                        }
                    }
                },
                "_bind": {
                    "resolver": {
                        "cachestats": {
                            "CacheHits": 0,
                            "CacheMisses": 0
                        }
                    }
                }
            }
        }`

	testOutput := NamedStatsGetResponse{}
	err := UnmarshalNamedStatsResponse(response, &testOutput)
	require.NoError(t, err)
	require.NotNil(t, testOutput)

	// Two views.
	require.NotNil(t, testOutput.Views)
	require.Contains(t, *(testOutput).Views, "_default")
	require.Contains(t, *(testOutput).Views, "_bind")
}

// Test that named statistics-channel response with no views.
func TestUnmarshalNamedStatsNoViews(t *testing.T) {
	response := `{
            "json-stats-version": "1.2"
        }`

	testOutput := NamedStatsGetResponse{}
	err := UnmarshalNamedStatsResponse(response, &testOutput)
	require.NoError(t, err)
	require.NotNil(t, testOutput)
	require.Nil(t, testOutput.Views)
}

// Test that named statistics-channel malformed output.
func TestUnmarshalNamedStatsMalformed(t *testing.T) {
	response := `{
            "views": 1
        }`

	testOutput := NamedStatsGetResponse{}
	err := UnmarshalNamedStatsResponse(response, &testOutput)
	require.Error(t, err)
}
