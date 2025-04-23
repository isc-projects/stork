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

// JSON get-all-statistic response.
// There is a response entry for each service. The order of entries is the
// same as the order of services in the request.
type GetAllStatisticsResponse = []GetAllStatisticResponseItem

// JSON get-all-statistic single item response returned.
type GetAllStatisticResponseItem struct {
	ResponseHeader
	Arguments GetAllStatisticArguments
}

type GetAllStatisticArguments []GetAllStatisticResponseSample

type GetAllStatisticResponseSample struct {
	// Statistic name.
	Name string
	// Subnet ID is used for the subnet statistics.
	// It is zero for the global statistics.
	SubnetID int64
	// Pool ID is used for the pool statistics.
	// It is zero for non-pool statistics.
	PoolID int64
	// Prefix pool ID is used for the prefix pool statistics.
	// It is zero for non-prefix pool statistics.
	PrefixPoolID int64
	Value        *big.Int
}

// UnmarshalJSON implements json.Unmarshaler. It unpacks the Kea response
// to simpler Go-friendly form.
func (r *GetAllStatisticArguments) UnmarshalJSON(b []byte) error {
	// Raw structures - corresponding to real received JSON.
	// Arguments node looks like:
	// "arguments": {
	//     "cumulative-assigned-nas": [
	//         [
	//             0,
	//             "2025-04-22 17:59:15.338212"
	//         ]
	//     ],
	// }

	var obj map[string][][]json.RawMessage

	err := json.Unmarshal(b, &obj)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse response arguments from Kea")
		return err
	}

	var samples []GetAllStatisticResponseSample

	// Retrieve values of mixed-type arrays.
	// Unpack the complex structure to simpler form.
	for statName, statValueOuterList := range obj {
		var subnetID int64
		var poolID int64
		var prefixPoolID int64
		// Extract the subnet ID and pool ID if present.
		if strings.HasPrefix(statName, "subnet[") {
			re := regexp.MustCompile(`subnet\[(\d+)\]\.(.+)`)
			matches := re.FindStringSubmatch(statName)
			subnetIDRaw := matches[1]
			statName = matches[2]

			subnetID, err = strconv.ParseInt(subnetIDRaw, 10, 64)
			if err != nil {
				log.Errorf("Problem converting subnetID: %s", subnetIDRaw)
				continue
			}

			// Extract the pool (or prefix pool) ID if present.
			if strings.HasPrefix(statName, "pool[") {
				re = regexp.MustCompile(`pool\[(\d+)\]\.(.+)`)
				matches = re.FindStringSubmatch(statName)
				poolIDRaw := matches[1]
				statName = matches[2]

				poolID, err = strconv.ParseInt(poolIDRaw, 10, 64)
				if err != nil {
					log.Errorf("Problem converting poolID: %s", poolIDRaw)
					continue
				}
			} else if strings.HasPrefix(statName, "prefix-pool[") {
				re = regexp.MustCompile(`prefix-pool\[(\d+)\]\.(.+)`)
				matches = re.FindStringSubmatch(statName)
				prefixPoolIDRaw := matches[1]
				statName = matches[2]

				prefixPoolID, err = strconv.ParseInt(prefixPoolIDRaw, 10, 64)
				if err != nil {
					log.Errorf("Problem converting prefixPoolID: %s", prefixPoolIDRaw)
					continue
				}
			}
		}

		// Fix typo for legacy Kea versions.
		statName = strings.Replace(statName, "addreses", "addresses", 1)

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

		sample := GetAllStatisticResponseSample{
			Name:         statName,
			SubnetID:     subnetID,
			PoolID:       poolID,
			PrefixPoolID: prefixPoolID,
			Value:        statValueBigInt,
		}

		samples = append(samples, sample)
	}

	*r = samples
	return nil
}
