package keactrl

import (
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	storkutil "isc.org/stork/util"
)

const (
	StatisticGet    CommandName = "statistic-get"
	StatisticGetAll CommandName = "statistic-get-all"
)

var (
	// Matches a subnet ID in the Kea statistic name.
	subnetStatNameRegex = regexp.MustCompile(`subnet\[(\d+)\]\.(.+)`)
	// Matches a pool ID in the Kea statistic name.
	poolStatNameRegex = regexp.MustCompile(`pool\[(\d+)\]\.(.+)`)
	// Matches a prefix pool ID in the Kea statistic name.
	pdPoolStatNameRegex = regexp.MustCompile(`pd-pool\[(\d+)\]\.(.+)`)
)

// JSON statistic-get-all response.
type StatisticGetAllResponse struct {
	ResponseHeader
	Arguments StatisticGetAllResponseArguments
}

// The Golang representation of the statistic-get-all arguments.
type StatisticGetAllResponseArguments []*StatisticGetAllResponseSample

// Single statistic from the statistic-get-all response.
type StatisticGetAllResponseSample struct {
	// Statistic name.
	Name string
	// Subnet ID is used for the subnet statistics.
	// It is zero for the global statistics.
	SubnetID int64
	// Address pool ID is used for the pool statistics.
	// It is nil for non-pool statistics.
	// It is mutually exclusive with the prefix pool ID.
	AddressPoolID *int64
	// Prefix pool ID is used for the prefix pool statistics.
	// It is nil for non-prefix pool statistics.
	// It is mutually exclusive with the address pool ID.
	PrefixPoolID *int64
	// Statistic value.
	// The value is a big integer because it can be larger than uint64.
	// Warning: We expect the value to be a positive integer because in older
	// Kea versions the statistics were stored as uint64 but returned as int64,
	// so any negative value was expected to be a number above maxInt64.
	// This problem was fixed in Kea 2.5.3. See kea#3068.
	Value *big.Int
}

// Indicates if the sample contains an address or prefix pool statistic.
func (s *StatisticGetAllResponseSample) IsPoolSample() bool {
	return s.IsAddressPoolSample() || s.IsPrefixPoolSample()
}

// Indicates if the sample contains an address pool statistic.
func (s *StatisticGetAllResponseSample) IsAddressPoolSample() bool {
	return s.AddressPoolID != nil
}

// Indicates if the sample contains a prefix pool statistic.
func (s *StatisticGetAllResponseSample) IsPrefixPoolSample() bool {
	return s.PrefixPoolID != nil
}

// Indicates if the sample contains a subnet statistic.
func (s *StatisticGetAllResponseSample) IsSubnetSample() bool {
	return s.SubnetID != 0 && !s.IsPoolSample()
}

// Returns a pool ID related to the sample. If sample is not a pool sample,
// returns nil.
func (s *StatisticGetAllResponseSample) GetPoolID() *int64 {
	if s.IsAddressPoolSample() {
		return s.AddressPoolID
	}
	if s.IsPrefixPoolSample() {
		return s.PrefixPoolID
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler. It unpacks the Kea response
// to simpler Go-friendly form.
func (r *StatisticGetAllResponseArguments) UnmarshalJSON(b []byte) error {
	// Arguments property of the Kea response looks like below. Its inner list
	// contains two different types of values: number and string. The Go JSON
	// library does not support mixed-type arrays. The workaround is to
	// unmarshal the values manually by using the json.RawMessage type.
	//
	// Example of the arguments property for Kea DHCPv6 2.7.6:
	//
	// "arguments": {
	//     "cumulative-assigned-nas": [
	//         [
	//             0,
	//             "2025-04-22 17:59:15.338212"
	//         ]
	//     ],
	//     "subnet[10].assigned-nas": [
	//         [
	//             0,
	//             "2024-10-04 14:24:04.401919"
	//         ]
	//      ],
	//      "subnet[10].pool[0].total-nas": [
	//          [
	//              4,
	//              "2024-10-04 14:24:04.401216"
	//          ]
	//      ],
	//      "subnet[1].pd-pool[42].total-pds": [
	//          [
	//              512,
	//              "2024-10-04 14:24:04.401035"
	//          ],
	//          [
	//              256,
	//              "2024-10-04 14:24:04.401028"
	//          ]
	//      ],
	// }

	var obj map[string][][]json.RawMessage

	err := json.Unmarshal(b, &obj)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse response arguments from Kea")
		return err
	}

	var samples []*StatisticGetAllResponseSample

	// Retrieve values of mixed-type arrays.
	// Unpack the complex structure to simpler form.
	for statName, statValueOuterList := range obj {
		var subnetID int64
		var addressPoolID *int64
		var prefixPoolID *int64
		// Extract the subnet ID and pool ID if present.
		if strings.HasPrefix(statName, "subnet[") {
			matches := subnetStatNameRegex.FindStringSubmatch(statName)
			subnetIDRaw := matches[1]
			statName = matches[2]

			subnetID, err = strconv.ParseInt(subnetIDRaw, 10, 64)
			if err != nil {
				log.WithField("statistic", statName).
					Errorf("Problem converting subnet ID: %s", subnetIDRaw)
				continue
			}

			// Extract the pool (or prefix pool) ID if present.
			if strings.HasPrefix(statName, "pool[") {
				matches = poolStatNameRegex.FindStringSubmatch(statName)
				poolIDRaw := matches[1]
				statName = matches[2]

				parsedAddressPoolID, err := strconv.ParseInt(poolIDRaw, 10, 64)
				if err != nil {
					log.WithField("statistic", statName).
						Errorf("Problem converting pool ID: %s", poolIDRaw)
					continue
				}

				addressPoolID = &parsedAddressPoolID
			} else if strings.HasPrefix(statName, "pd-pool[") {
				matches = pdPoolStatNameRegex.FindStringSubmatch(statName)
				prefixPoolIDRaw := matches[1]
				statName = matches[2]

				parsedPrefixPoolID, err := strconv.ParseInt(prefixPoolIDRaw, 10, 64)
				if err != nil {
					log.WithField("statistic", statName).
						Errorf("Problem converting prefix pool ID: %s", prefixPoolIDRaw)
					continue
				}

				prefixPoolID = &parsedPrefixPoolID
			}
		}

		// Fix typo for legacy Kea versions.
		statName = strings.Replace(statName, "addreses", "addresses", 1) // cspell:disable-line

		if len(statValueOuterList) == 0 {
			log.Errorf("Empty list of stat values")
			continue
		}
		statValueInnerList := statValueOuterList[0]

		if len(statValueInnerList) == 0 {
			log.Errorf("Empty list of stat values")
			continue
		}

		var statValue storkutil.BigIntJSON
		err = json.Unmarshal(statValueInnerList[0], &statValue)
		if err != nil {
			log.WithError(err).Errorf(
				"Problem unmarshalling statistic value: '%s'",
				statValueInnerList[0],
			)
			continue
		}

		statValueBigInt := statValue.BigInt()

		if statValueBigInt.Sign() == -1 {
			// Handle negative statistics from older Kea versions.
			// Older Kea versions stored the statistics as uint64
			// but they were returned as int64.
			//
			// For the negative int64 values:
			// uint64 = maxUint64 + (int64 + 1)
			statValueBigInt = big.NewInt(0).Add(
				big.NewInt(0).SetUint64(math.MaxUint64),
				big.NewInt(0).Add(
					big.NewInt(1),
					statValueBigInt,
				),
			)
		}

		sample := &StatisticGetAllResponseSample{
			Name:          statName,
			SubnetID:      subnetID,
			AddressPoolID: addressPoolID,
			PrefixPoolID:  prefixPoolID,
			Value:         statValueBigInt,
		}

		samples = append(samples, sample)
	}

	*r = samples
	return nil
}
