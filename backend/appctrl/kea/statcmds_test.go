package keactrl

import (
	_ "embed"
	"encoding/json"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test if the Kea DHCPv4 JSON statistic-get-all response is unmarshal correctly.
func TestUnmarshalKeaDHCPv4StatisticGetAllResponse(t *testing.T) {
	// Arrange
	rawResponse := `
	[
		{
			"arguments": {
				"cumulative-assigned-addresses": [ [0, "2021-10-14 10:44:18.687247"] ],
				"declined-addresses": [ [50, "2021-10-14 10:44:18.687235"] ],
				"assigned-addresses": [ [150, "2021-10-14 10:44:18.687235"] ],
				"pkt4-ack-received": [ [0, "2021-10-14 10:44:18.672377"] ],
				"pkt4-ack-sent": [ [0, "2021-10-14 10:44:18.672378"] ],
				"pkt4-decline-received": [ [0, "2021-10-14 10:44:18.672379"] ],
				"pkt4-discover-received": [ [0, "2021-10-14 10:44:18.672380"] ],
				"pkt4-inform-received": [ [0, "2021-10-14 10:44:18.672380"] ],
				"pkt4-nak-received": [ [0, "2021-10-14 10:44:18.672381"] ],
				"pkt4-nak-sent": [ [0, "2021-10-14 10:44:18.672382"] ],
				"pkt4-offer-received": [ [0, "2021-10-14 10:44:18.672382"] ],
				"pkt4-offer-sent": [ [0, "2021-10-14 10:44:18.672383"] ],
				"pkt4-parse-failed": [ [0, "2021-10-14 10:44:18.672384"] ],
				"pkt4-receive-drop": [ [0, "2021-10-14 10:44:18.672389"] ],
				"pkt4-received": [ [0, "2021-10-14 10:44:18.672390"] ],
				"pkt4-release-received": [ [0, "2021-10-14 10:44:18.672390"] ],
				"pkt4-request-received": [ [0, "2021-10-14 10:44:18.672391"] ],
				"pkt4-sent": [ [0, "2021-10-14 10:44:18.672392"] ],
				"pkt4-unknown-received": [ [0, "2021-10-14 10:44:18.672392"] ],
				"reclaimed-declined-addresses": [ [0, "2021-10-14 10:44:18.687239"] ],
				"reclaimed-leases": [ [0, "2021-10-14 10:44:18.687243"] ],
				"subnet[1].assigned-addresses": [ [10, "2021-10-14 10:44:18.687253"] ],
				"subnet[1].cumulative-assigned-addresses": [ [0, "2021-10-14 10:44:18.687229"] ],
				"subnet[1].declined-addresses": [ [3, "2021-10-14 10:44:18.687266"] ],
				"subnet[1].reclaimed-declined-addresses": [ [0, "2021-10-14 10:44:18.687274"] ],
				"subnet[1].reclaimed-leases": [ [0, "2021-10-14 10:44:18.687282"] ],
				"subnet[1].total-addresses": [ [200, "2021-10-14 10:44:18.687221"] ],
				"subnet[1].pool[0].assigned-addresses": [ [7, "2025-04-22 17:59:15.339186"] ],
				"subnet[1].pool[0].cumulative-assigned-addresses": [ [0, "2025-04-22 17:59:15.328531" ] ],
				"subnet[1].pool[0].declined-addresses": [ [2, "2025-04-22 17:59:15.339184" ] ],
				"subnet[1].pool[0].reclaimed-declined-addresses": [ [0, "2025-04-22 17:59:15.338433" ] ],
				"subnet[1].pool[0].reclaimed-leases": [ [0, "2025-04-22 17:59:15.338438"] ],
				"subnet[1].pool[0].total-addresses": [ [42, "2025-04-22 17:59:15.328653" ] ]
			},
			"result": 0
		},
		{
			"result": 1,
			"text": "Unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline"
		}
	]`

	// Act
	var response StatisticGetAllResponse
	err := json.Unmarshal([]byte(rawResponse), &response)

	// Assert
	require.NoError(t, err)
	require.Len(t, response, 2)
	require.Len(t, response[0].Arguments, 33)
	require.Nil(t, response[1].Arguments)
	require.NoError(t, response[0].GetError())
	require.Error(t, response[1].GetError())

	index := slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "total-addresses" && item.IsSubnetSample() && item.SubnetID == 1
	})
	require.NotEqual(t, -1, index)
	item := response[0].Arguments[index]
	require.EqualValues(t, 200, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "reclaimed-leases"
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.Zero(t, item.Value.Int64())

	// Check the assigned lease statistics. They should count the assigned and
	// declined leases together.
	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-addresses" && item.IsAddressPoolSample() && item.SubnetID == 1 && *item.AddressPoolID == 0
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 7, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-addresses" && item.IsSubnetSample() && item.SubnetID == 1
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 10, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-addresses" && !item.IsSubnetSample() && !item.IsPoolSample()
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 150, item.Value.Int64())

	// Check if the statistics can be adjusted to exclude the declined leases
	// from the assigned ones.
	AdjustAssignedStatistics(response[0].Arguments)

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-addresses" && item.IsAddressPoolSample() && item.SubnetID == 1 && *item.AddressPoolID == 0
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 5, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-addresses" && item.IsSubnetSample() && item.SubnetID == 1
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 7, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-addresses" && !item.IsSubnetSample() && !item.IsPoolSample()
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 100, item.Value.Int64())
}

// Test that the assigned NAs can be assigned correctly for the Kea DHCPv6.
func TestAdjustAssignedStatisticsForKeaDHCPv6(t *testing.T) {
	// Arrange
	rawResponse := `
	[
		{
			"arguments": {
				"declined-addresses": [ [150, "2021-10-14 10:44:18.687235"] ],
				"assigned-nas": [ [450, "2021-10-14 10:44:18.687235"] ],
				"subnet[2].assigned-nas": [ [230, "2021-10-14 10:44:18.687253"] ],
				"subnet[2].declined-addresses": [ [5, "2021-10-14 10:44:18.687266"] ],
				"subnet[2].pool[1].assigned-nas": [ [36, "2025-04-22 17:59:15.339186"] ],
				"subnet[2].pool[1].declined-addresses": [ [6, "2025-04-22 17:59:15.339184" ] ]
			},
			"result": 0
		},
		{
			"result": 1,
			"text": "Unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline"
		}
	]`

	// Act
	var response StatisticGetAllResponse
	err := json.Unmarshal([]byte(rawResponse), &response)
	AdjustAssignedStatistics(response[0].Arguments)

	// Assert
	require.NoError(t, err)

	index := slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-nas" && item.IsAddressPoolSample() && item.SubnetID == 2 && *item.AddressPoolID == 1
	})
	require.NotEqual(t, -1, index)
	item := response[0].Arguments[index]
	require.EqualValues(t, 30, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-nas" && item.IsSubnetSample() && item.SubnetID == 2
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 225, item.Value.Int64())

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "assigned-nas" && !item.IsSubnetSample() && !item.IsPoolSample()
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, 300, item.Value.Int64())
}

// Test that unmarshalling of the Kea statistic-get-all response does not lose
// precision when the values exceed the maximum value of int64.
func TestUnmarshalStatisticGetAllResponseBigNumbers(t *testing.T) {
	// Arrange
	jsonString := `
		[{
			"result": 0,
			"arguments": {
				"subnet[1].total-nas": [
					[844424930131968, "2021-10-14 10:44:18.687221"]
				],
				"subnet[2].total-nas": [
					[281474976710656, "2021-10-14 10:44:18.687221"]
				],
				"subnet[4].total-nas": [
					[36893488147419103232, "2021-10-14 10:44:18.687221"]
				],
				"subnet[5].total-nas": [
					[-1, "2021-10-14 10:44:18.687221"]
				]
			}
		}]
	`

	var response StatisticGetAllResponse
	expected0, _ := big.NewInt(0).SetString("844424930131968", 10)
	expected1, _ := big.NewInt(0).SetString("281474976710656", 10)
	expected2, _ := big.NewInt(0).SetString("36893488147419103232", 10)
	expected3 := big.NewInt(0).SetUint64(math.MaxUint64)

	// Act
	err := json.Unmarshal([]byte(jsonString), &response)

	// Assert
	require.NoError(t, err)
	require.Len(t, response, 1)

	index := slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "total-nas" && item.SubnetID == 1
	})
	require.NotEqual(t, -1, index)
	item := response[0].Arguments[index]
	require.EqualValues(t, expected0, item.Value)

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "total-nas" && item.SubnetID == 2
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, expected1, item.Value)

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "total-nas" && item.SubnetID == 4
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, expected2, item.Value)

	index = slices.IndexFunc(response[0].Arguments, func(item *StatisticGetAllResponseSample) bool {
		return item.Name == "total-nas" && item.SubnetID == 5
	})
	require.NotEqual(t, -1, index)
	item = response[0].Arguments[index]
	require.EqualValues(t, expected3, item.Value)
}

// Test that the pool sample is recognized correctly.
func TestIsPoolSample(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		addressPoolID *int64
		prefixPoolID  *int64
		expected      bool
	}{
		{
			name:          "address pool ID is set",
			addressPoolID: storkutil.Ptr(int64(1)),
			prefixPoolID:  nil,
			expected:      true,
		},
		{
			name:          "prefix pool ID is set",
			addressPoolID: nil,
			prefixPoolID:  storkutil.Ptr(int64(2)),
			expected:      true,
		},
		{
			name:          "both pool IDs are set",
			addressPoolID: storkutil.Ptr(int64(1)),
			prefixPoolID:  storkutil.Ptr(int64(2)),
			expected:      true,
		},
		{
			name:          "no pool IDs are set",
			addressPoolID: nil,
			prefixPoolID:  nil,
			expected:      false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sample := StatisticGetAllResponseSample{
				AddressPoolID: tc.addressPoolID,
				PrefixPoolID:  tc.prefixPoolID,
			}
			result := sample.IsPoolSample()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIsAddressPoolSample(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		addressPoolID *int64
		expected      bool
	}{
		{
			name:          "address pool ID is set",
			addressPoolID: storkutil.Ptr(int64(1)),
			expected:      true,
		},
		{
			name:          "address pool ID is not set",
			addressPoolID: nil,
			expected:      false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sample := StatisticGetAllResponseSample{
				AddressPoolID: tc.addressPoolID,
			}
			result := sample.IsAddressPoolSample()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPrefixPoolSample(t *testing.T) {
	// Test cases
	testCases := []struct {
		name         string
		prefixPoolID *int64
		expected     bool
	}{
		{
			name:         "prefix pool ID is set",
			prefixPoolID: storkutil.Ptr(int64(1)),
			expected:     true,
		},
		{
			name:         "prefix pool ID is not set",
			prefixPoolID: nil,
			expected:     false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sample := StatisticGetAllResponseSample{
				PrefixPoolID: tc.prefixPoolID,
			}
			result := sample.IsPrefixPoolSample()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIsSubnetSample(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		subnetID      int64
		addressPoolID *int64
		prefixPoolID  *int64
		expected      bool
	}{
		{
			name:          "subnet ID is set but not a pool",
			subnetID:      1,
			addressPoolID: nil,
			prefixPoolID:  nil,
			expected:      true,
		},
		{
			name:          "subnet ID is not set",
			subnetID:      0,
			addressPoolID: nil,
			prefixPoolID:  nil,
			expected:      false,
		},
		{
			name:          "subnet ID is set but address pool ID is also set",
			subnetID:      1,
			addressPoolID: storkutil.Ptr(int64(1)),
			prefixPoolID:  nil,
			expected:      false,
		},
		{
			name:          "subnet ID is set but prefix pool ID is also set",
			subnetID:      1,
			addressPoolID: nil,
			prefixPoolID:  storkutil.Ptr(int64(1)),
			expected:      false,
		},
		{
			name:          "subnet ID is set but both pool IDs are also set",
			subnetID:      1,
			addressPoolID: storkutil.Ptr(int64(1)),
			prefixPoolID:  storkutil.Ptr(int64(2)),
			expected:      false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sample := StatisticGetAllResponseSample{
				SubnetID:      tc.subnetID,
				AddressPoolID: tc.addressPoolID,
				PrefixPoolID:  tc.prefixPoolID,
			}
			result := sample.IsSubnetSample()
			require.Equal(t, tc.expected, result)
		})
	}
}

// Test that the pool ID is retrieved from a statistic sample properly.
func TestStatisticGetAllResponseSampleGetPoolID(t *testing.T) {
	t.Run("empty sample", func(t *testing.T) {
		// Arrange
		sample := &StatisticGetAllResponseSample{}

		// Act & Assert
		require.Nil(t, sample.GetPoolID())
	})

	t.Run("subnet sample", func(t *testing.T) {
		// Arrange
		sample := &StatisticGetAllResponseSample{SubnetID: 42}

		// Act & Assert
		require.Nil(t, sample.GetPoolID())
	})

	t.Run("address pool sample", func(t *testing.T) {
		// Arrange
		sample := &StatisticGetAllResponseSample{
			AddressPoolID: storkutil.Ptr(int64(42)),
		}

		// Act & Assert
		require.NotNil(t, sample.GetPoolID())
		require.EqualValues(t, 42, *sample.GetPoolID())
	})

	t.Run("delegated prefix pool sample", func(t *testing.T) {
		// Arrange
		sample := &StatisticGetAllResponseSample{
			PrefixPoolID: storkutil.Ptr(int64(42)),
		}

		// Act & Assert
		require.NotNil(t, sample.GetPoolID())
		require.EqualValues(t, 42, *sample.GetPoolID())
	})

	// This case should never happen. The statistic sample must contain
	// only single pool ID. This test checks if Stork doesn't crash in such
	// case.
	t.Run("both pool sample", func(t *testing.T) {
		// Arrange
		sample := &StatisticGetAllResponseSample{
			AddressPoolID: storkutil.Ptr(int64(42)),
			PrefixPoolID:  storkutil.Ptr(int64(24)),
		}

		// Act & Assert
		require.NotNil(t, sample.GetPoolID())
		require.EqualValues(t, 42, *sample.GetPoolID())
	})
}
