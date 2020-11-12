package keaconfig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// This test verifies that new instance of the IndexedSubnets structure
// can be created and that the indexes are initially empty.
func TestNewIndexedSubnets(t *testing.T) {
	rawConfig, err := NewFromJSON(`{
        "Dhcp4": {
        }
    }`)
	require.NoError(t, err)
	is := NewIndexedSubnets(rawConfig)
	require.NotNil(t, is)
	require.Empty(t, is.RandomAccess)
	require.Nil(t, is.ByPrefix)
}

// This test verifies that NewIndexedSubnets function panics when provided
// config pointer is nil.
func TestNewIndexedSubnetsNilConfig(t *testing.T) {
	require.Panics(t, func() { _ = NewIndexedSubnets(nil) })
}

// This test verifies that subnets can be inserted into the IndexedSubnets
// structure and that duplicated entries are rejected.
func TestIndexedSubnetsPopulate(t *testing.T) {
	config, err := NewFromJSON(`{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                },
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.4.0/24"
                        }
                    ]
                }
            ],
            "subnet4": [
                {
                    "subnet": "192.0.2.0/24"
                },
                {
                    "subnet": "10.0.0.0/8"
                }
            ]
        }
	}`)
	require.NoError(t, err)

	is := NewIndexedSubnets(config)
	require.NotNil(t, is)

	require.NoError(t, is.Populate())

	// All subnets should be now stored in random access index.
	require.Len(t, is.RandomAccess, 4)
	require.Contains(t, is.RandomAccess[0], "subnet")
	require.EqualValues(t, "192.0.3.0/24", is.RandomAccess[0].(map[string]interface{})["subnet"])
	require.Contains(t, is.RandomAccess[1], "subnet")
	require.EqualValues(t, "192.0.4.0/24", is.RandomAccess[1].(map[string]interface{})["subnet"])
	require.Contains(t, is.RandomAccess[2], "subnet")
	require.EqualValues(t, "192.0.2.0/24", is.RandomAccess[2].(map[string]interface{})["subnet"])
	require.Contains(t, is.RandomAccess[3], "subnet")
	require.EqualValues(t, "10.0.0.0/8", is.RandomAccess[3].(map[string]interface{})["subnet"])

	// All subnets should ne stored in the by-prefix index.
	require.Len(t, is.ByPrefix, 4)
	require.Contains(t, is.ByPrefix, "192.0.2.0/24")
	require.Contains(t, is.ByPrefix, "192.0.4.0/24")
	require.Contains(t, is.ByPrefix, "192.0.3.0/24")
	require.Contains(t, is.ByPrefix, "10.0.0.0/8")
}

// Test that invalid configurations are rejected when indexing subnets.
func TestIndexedSubnetsPopulateWrongConfigs(t *testing.T) {
	testCases := []struct {
		name   string
		config string
	}{
		{
			name:   "empty config",
			config: "{}",
		},
		{
			name: "non DHCP config",
			config: `{
                "DhcpDddns": { }
            }`,
		},
		{
			name: "invalid shared network structure",
			config: `{
                "Dhcp4": {
                    "shared-networks": [1]
                }
            }`,
		},
		{
			name: "invalid subnet structure",
			config: `{
                "Dhcp4": {
                    "subnet4": [1]
                }
            }`,
		},
		{
			name: "no subnet prefix",
			config: `{
                "Dhcp4": {
                    "subnet4": [
                        {
                            "id": 1
                        }
                    ]
                }
            }`,
		},
		{
			name: "duplicate prefix",
			config: `{
                "Dhcp4": {
                    "subnet4": [
                        {
                            "prefix": "10.0.0.0/8"
                        },
                        {
                            "prefix": "10.0.0.0/8"
                        }
                    ]
                }
            }`,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			config, err := NewFromJSON(tc.config)
			require.NoError(t, err)
			is := NewIndexedSubnets(config)
			require.NotNil(t, is)
			require.Error(t, is.Populate())
			require.Empty(t, is.RandomAccess)
			require.Empty(t, is.ByPrefix)
		})
	}
}

// Benchmark measuring performance of indexing many subnets by prefix.
func BenchmarkIndexedSubnetsPopulate(b *testing.B) {
	// Create many subnets.
	subnets := []interface{}{}
	for i := 0; i < 10000; i++ {
		subnet := map[string]interface{}{
			"subnet": fmt.Sprintf("%d.%d.%d.%d/24", byte(i>>24), byte(i>>16), byte(i>>8), byte(i)),
		}
		subnets = append(subnets, subnet)
	}

	config := map[string]interface{}{
		"Dhcp4": map[string]interface{}{
			"subnet4": subnets,
		},
	}

	// Index the subnets.
	indexedSubnets := NewIndexedSubnets(New(&config))

	// Actual benchmark starts here.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexedSubnets.Populate()
	}
}
