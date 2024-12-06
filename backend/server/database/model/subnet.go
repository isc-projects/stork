package dbmodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Interface checks.
var _ keaconfig.SubnetAccessor = (*Subnet)(nil)

// Identifier of the well-known subnet statistics.
type SubnetStatsName = string

const (
	// Total number of network addresses.
	SubnetStatsNameTotalNAs SubnetStatsName = "total-nas"
	// Number of assigned network addresses.
	SubnetStatsNameAssignedNAs SubnetStatsName = "assigned-nas"
	// Number of declined network addresses.
	SubnetStatsNameDeclinedNAs SubnetStatsName = "declined-nas"
	// Total number of delegated prefixes.
	SubnetStatsNameTotalPDs SubnetStatsName = "total-pds"
	// Number of assigned delegated prefixes.
	SubnetStatsNameAssignedPDs SubnetStatsName = "assigned-pds"
	// Total number of addresses.
	SubnetStatsNameTotalAddresses SubnetStatsName = "total-addresses"
	// Number of assigned addresses.
	SubnetStatsNameAssignedAddresses SubnetStatsName = "assigned-addresses"
	// Number of declined addresses.
	SubnetStatsNameDeclinedAddresses SubnetStatsName = "declined-addresses"
	// Cumulative number of assigned network addresses.
	SubnetStatsNameCumulativeAssignedAddresses SubnetStatsName = "cumulative-assigned-addresses"
)

// Custom statistic type to redefine JSON marshalling.
type SubnetStats map[SubnetStatsName]any

// Returns the value of the statistic with the specified name as a big counter.
func (s SubnetStats) GetBigCounter(name SubnetStatsName) *storkutil.BigCounter {
	value, ok := s[name]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case *big.Int:
		return storkutil.NewBigCounterFromBigInt(v)
	case int64:
		return storkutil.NewBigCounterFromInt64(v)
	case uint64:
		return storkutil.NewBigCounter(v)
	default:
		return nil
	}
}

// Sets the value of the statistic with the specified name as a big counter.
func (s SubnetStats) SetBigCounter(name SubnetStatsName, counter *storkutil.BigCounter) {
	s[name] = counter.ConvertToNativeType()
}

// Subnet statistics may contain the integer number within arbitrary range
// (int64, uint64, bigint) (max value is 2^63-1, 2^64-1, or any). The value
// returned by the Kea and stored in the Postgres database is exact. But when
// the frontend fetches this data, it deserializes it using the standard
// JSON.parse function. This function treats all number literals as floating
// double-precision numbers. This type can exact handle integers up to
// (2^53 - 1); greater numbers are inaccurate.
// All the numeric statistics are serialized to string and next deserialized using
// a custom function to avoid losing the precision.
//
// It doesn't use the pointer to receiver type for compatibility with go-pg serialization
// during inserting to the database.
func (s SubnetStats) MarshalJSON() ([]byte, error) {
	if s == nil {
		return json.Marshal(nil)
	}

	toMarshal := make(map[string]any, len(s))

	for k, v := range s {
		switch value := v.(type) {
		case *big.Int, int64, uint64:
			toMarshal[k] = fmt.Sprint(value)
		default:
			toMarshal[k] = value
		}
	}

	return json.Marshal(toMarshal)
}

// Deserialize statistics and convert back the strings to int64 or uint64.
// I assume that the statistics will always contain numeric data, no string
// that look like integers.
// During the serialization we lost the original data type of the number.
// We assume that strings with positive numbers are uint64 and negative numbers are int64.
func (s *SubnetStats) UnmarshalJSON(data []byte) error {
	toUnmarshal := make(map[string]interface{})
	err := json.Unmarshal(data, &toUnmarshal)
	if err != nil {
		return err
	}

	if toUnmarshal == nil {
		*s = nil
		return nil
	}

	if *s == nil {
		*s = SubnetStats{}
	}

	for k, v := range toUnmarshal {
		vStr, ok := v.(string)
		if !ok {
			(*s)[k] = v
			continue
		}

		vUint64, err := strconv.ParseUint(vStr, 10, 64)
		if err == nil {
			(*s)[k] = vUint64
			continue
		}

		vInt64, err := strconv.ParseInt(vStr, 10, 64)
		if err == nil {
			(*s)[k] = vInt64
			continue
		}

		vBigInt, ok := new(big.Int).SetString(vStr, 10)
		if ok {
			(*s)[k] = vBigInt
			continue
		}

		(*s)[k] = v
	}

	return nil
}

// An interface for a wrapper of subnet statistics that encapsulates the
// utilization calculations. It corresponds to the
// `statisticscounter.subnetStats` interface and prevents the dependency cycle.
type utilizationStats interface {
	GetAddressUtilization() float64
	GetDelegatedPrefixUtilization() float64
	GetStatistics() SubnetStats
}

// This structure holds subnet information retrieved from an app. Multiple
// DHCP server apps may be configured to serve leases in the same subnet.
// For the same subnet configured on different DHCP server there will be
// a separate instance of the LocalSubnet structure. Apart from possibly
// different local subnet id between different apps there will also be
// other information stored here, e.g. statistics for the particular
// subnet retrieved from the given app. Multiple local subnets can be
// associated with a single global subnet depending on how many daemons
// serve the same subnet.
type LocalSubnet struct {
	DHCPOptionSet
	ID            int64
	SubnetID      int64
	DaemonID      int64
	Daemon        *Daemon `pg:"rel:has-one"`
	Subnet        *Subnet `pg:"rel:has-one"`
	LocalSubnetID int64

	Stats            SubnetStats
	StatsCollectedAt time.Time

	AddressPools []AddressPool `pg:"rel:has-many"`
	PrefixPools  []PrefixPool  `pg:"rel:has-many"`

	KeaParameters *keaconfig.SubnetParameters
	UserContext   map[string]any
}

// Reflects IPv4 or IPv6 subnet from the database.
type Subnet struct {
	ID          int64
	CreatedAt   time.Time
	Prefix      string
	ClientClass string

	SharedNetworkID int64
	SharedNetwork   *SharedNetwork `pg:"rel:has-one"`

	LocalSubnets []*LocalSubnet `pg:"rel:has-many"`

	Hosts []Host `pg:"rel:has-many"`

	AddrUtilization  int16
	PdUtilization    int16
	Stats            SubnetStats
	StatsCollectedAt time.Time
}

// Returns local subnet id for the specified daemon.
func (s *Subnet) GetID(daemonID int64) int64 {
	for _, ls := range s.LocalSubnets {
		if ls.DaemonID == daemonID {
			return ls.LocalSubnetID
		}
	}
	return 0
}

// Returns the Kea DHCP parameters for the subnet configured in the specified daemon.
func (s *Subnet) GetKeaParameters(daemonID int64) *keaconfig.SubnetParameters {
	for _, ls := range s.LocalSubnets {
		if ls.DaemonID == daemonID {
			return ls.KeaParameters
		}
	}
	return nil
}

// Returns subnet prefix.
func (s *Subnet) GetPrefix() string {
	return s.Prefix
}

// Returns a slice of interfaces to the subnet address pools.
func (s *Subnet) GetAddressPools(daemonID int64) (accessors []dhcpmodel.AddressPoolAccessor) {
	for _, ls := range s.LocalSubnets {
		if ls.DaemonID != daemonID {
			continue
		}
		for i := range ls.AddressPools {
			accessors = append(accessors, &ls.AddressPools[i])
		}
	}
	return
}

// Returns a slice of interfaces to the subnet delegated prefix pools.
func (s *Subnet) GetPrefixPools(daemonID int64) (accessors []dhcpmodel.PrefixPoolAccessor) {
	for _, ls := range s.LocalSubnets {
		if ls.DaemonID != daemonID {
			continue
		}
		for i := range ls.PrefixPools {
			accessors = append(accessors, &ls.PrefixPools[i])
		}
	}
	return
}

// Returns DHCP options for the subnet configured in the specified daemon.
func (s *Subnet) GetDHCPOptions(daemonID int64) (accessors []dhcpmodel.DHCPOptionAccessor) {
	for _, ls := range s.LocalSubnets {
		if ls.DaemonID == daemonID {
			for i := range ls.DHCPOptionSet.Options {
				accessors = append(accessors, ls.DHCPOptionSet.Options[i])
			}
		}
	}
	return
}

// Return user context for the subnet configured in the specified daemon.
func (s *Subnet) GetUserContext(daemonID int64) map[string]any {
	for _, ls := range s.LocalSubnets {
		if ls.DaemonID == daemonID {
			return ls.UserContext
		}
	}
	return nil
}

// Return family of the subnet.
func (s *Subnet) GetFamily() int {
	family := 4
	if strings.Contains(s.Prefix, ":") {
		family = 6
	}
	return family
}

// Sets LocalSharedNetwork instance for the SharedNetwork. If the corresponding
// LocalSharedNetwork (having the same daemon ID) already exists, it is replaced
// with the specified instance. Otherwise, the instance is appended to the slice
// of LocalSharedNetwork.
func (s *Subnet) SetLocalSubnet(localSubnet *LocalSubnet) {
	for i, lsn := range s.LocalSubnets {
		if lsn.DaemonID == localSubnet.DaemonID {
			s.LocalSubnets[i] = localSubnet
			return
		}
	}
	s.LocalSubnets = append(s.LocalSubnets, localSubnet)
}

// Combines two hosts into a single host by copying LocalHost data from
// the other host.
func (s *Subnet) Join(other *Subnet) {
	for i := range other.LocalSubnets {
		s.SetLocalSubnet(other.LocalSubnets[i])
	}
}

// Hook executed after inserting a subnet to the database. It updates subnet
// id on the hosts belonging to this subnet.
func (s *Subnet) AfterInsert(ctx context.Context) error {
	if s != nil && s.ID != 0 {
		for i := range s.Hosts {
			s.Hosts[i].SubnetID = s.ID
		}
	}
	return nil
}

// Fetches daemon information for each daemon ID within the local subnets.
// The subnet information can be partial when it is created from the request
// received over the REST API. In particular, the LocalSubnets can merely
// contain DaemonID values and the Daemon pointers can be nil. In order
// to initialize Daemon pointers, this function fetches the daemons from
// the database and assigns them to the respective LocalSubnet instances.
// If any of the daemons does not exist or an error occurs, the subnet
// is not updated.
func (s Subnet) PopulateDaemons(dbi dbops.DBI) error {
	var daemons []*Daemon
	for _, ls := range s.LocalSubnets {
		// DaemonID is required for this function to run.
		if ls.DaemonID == 0 {
			return pkgerrors.Errorf("problem with populating daemons: subnet %d lacks daemon ID", s.ID)
		}
		daemon, err := GetDaemonByID(dbi, ls.DaemonID)
		if err != nil {
			return pkgerrors.WithMessage(err, "problem with populating daemons")
		}
		// Daemon does not exist.
		if daemon == nil {
			return pkgerrors.Errorf("problem with populating daemons for subnet %d: daemon %d does not exist", s.ID, ls.DaemonID)
		}
		daemons = append(daemons, daemon)
	}
	// Everything fine. Assign fetched daemons to the subnet.
	for i := range s.LocalSubnets {
		s.LocalSubnets[i].Daemon = daemons[i]
	}
	return nil
}

// Add address and prefix pools from the local subnet instance and remove the ones
// that no longer belong to the local subnet in a transaction.
// The subnet is expected to exist in the database.
func addAndClearSubnetPools(dbi dbops.DBI, localSubnet *LocalSubnet) (err error) {
	// Remove out-of-date pools.
	existingAddressPoolIDs := []int64{}
	for _, p := range localSubnet.AddressPools {
		if p.ID != 0 {
			existingAddressPoolIDs = append(existingAddressPoolIDs, p.ID)
		}
	}
	q := dbi.Model((*AddressPool)(nil)).
		Where("local_subnet_id = ?", localSubnet.ID)
	if len(existingAddressPoolIDs) != 0 {
		q = q.WhereIn("id NOT IN (?)", existingAddressPoolIDs)
	}
	if _, err = q.Delete(); err != nil {
		return pkgerrors.Wrap(err, "problem removing out-of-date address pools")
	}

	existingPrefixPoolIDs := []int64{}
	for _, p := range localSubnet.PrefixPools {
		if p.ID != 0 {
			existingPrefixPoolIDs = append(existingPrefixPoolIDs, p.ID)
		}
	}
	q = dbi.Model((*PrefixPool)(nil)).
		Where("local_subnet_id = ?", localSubnet.ID)
	if len(existingPrefixPoolIDs) != 0 {
		q = q.WhereIn("id NOT IN (?)", existingPrefixPoolIDs)
	}
	if _, err = q.Delete(); err != nil {
		return pkgerrors.Wrap(err, "problem removing out-of-date prefix pools")
	}

	// Check if there are entries to add or update.
	if len(localSubnet.AddressPools) == 0 && len(localSubnet.PrefixPools) == 0 {
		return nil
	}

	// Add address pools first.
	for i, p := range localSubnet.AddressPools {
		pool := p
		pool.LocalSubnetID = localSubnet.ID
		if pool.ID == 0 {
			_, err = dbi.Model(&pool).Insert()
			if err != nil {
				return pkgerrors.Wrapf(err, "problem adding address pool %s-%s for subnet with ID %d",
					pool.LowerBound, pool.UpperBound, localSubnet.ID)
			}
		}
		localSubnet.AddressPools[i] = pool
	}

	// Add prefix pools. This should be empty for IPv4 case.
	for i, p := range localSubnet.PrefixPools {
		pool := p
		pool.LocalSubnetID = localSubnet.ID
		if p.ID == 0 {
			_, err = dbi.Model(&pool).Insert()
			if err != nil {
				err = pkgerrors.Wrapf(err, "problem adding prefix pool %s for subnet with ID %d",
					pool.Prefix, localSubnet.ID)
				return err
			}
		}
		localSubnet.PrefixPools[i] = pool
	}

	return nil
}

// Adds a new subnet into the database within a transaction.
func addSubnet(tx *pg.Tx, subnet *Subnet) (err error) {
	// Add the subnet first.
	_, err = tx.Model(subnet).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding subnet with prefix %s", subnet.Prefix)
		return err
	}
	return nil
}

// Updates a subnet in the database within a transaction.
func updateSubnet(dbi dbops.DBI, subnet *Subnet) (err error) {
	// Update the subnet first.
	_, err = dbi.Model(subnet).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating subnet with prefix %s", subnet.Prefix)
		return err
	}
	return nil
}

// Adds a subnet with its pools into the database. If the subnet has any
// associations with a shared network, those associations are also created
// in the database. It begins a new transaction when dbi has a *pg.DB type
// or uses an existing transaction when dbi has a *pg.Tx type.
func AddSubnet(dbi dbops.DBI, subnet *Subnet) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addSubnet(tx, subnet)
		})
	}
	return addSubnet(dbi.(*pg.Tx), subnet)
}

// Iterates over the LocalSubnet instances of a Subnet and inserts them or
// updates in the database.
func AddLocalSubnets(dbi dbops.DBI, subnet *Subnet) error {
	for i := range subnet.LocalSubnets {
		subnet.LocalSubnets[i].SubnetID = subnet.ID
		q := dbi.Model(subnet.LocalSubnets[i]).
			OnConflict("(daemon_id, subnet_id) DO UPDATE").
			Set("local_subnet_id = EXCLUDED.local_subnet_id").
			Set("kea_parameters = EXCLUDED.kea_parameters").
			Set("dhcp_option_set = EXCLUDED.dhcp_option_set").
			Set("dhcp_option_set_hash = EXCLUDED.dhcp_option_set_hash").
			Set("user_context = EXCLUDED.user_context")
		_, err := q.Insert()
		if err != nil {
			return pkgerrors.Wrapf(err, "problem associating the daemon %d with the subnet %s",
				subnet.LocalSubnets[i].DaemonID, subnet.Prefix)
		}
		err = addAndClearSubnetPools(dbi, subnet.LocalSubnets[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// Fetches the subnet and its pools by id from the database.
func GetSubnet(dbi dbops.DBI, subnetID int64) (*Subnet, error) {
	subnet := &Subnet{}
	err := dbi.Model(subnet).
		Relation("LocalSubnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Relation("LocalSubnets.Daemon.App.Machine").
		Relation("LocalSubnets.Daemon.KeaDaemon").
		Relation("SharedNetwork.LocalSharedNetworks").
		Where("subnet.id = ?", subnetID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting the subnet with ID %d", subnetID)
		return nil, err
	}
	return subnet, err
}

// Fetches all subnets associated with the given daemon by ID.
func GetSubnetsByDaemonID(dbi dbops.DBI, daemonID int64) ([]Subnet, error) {
	subnets := []Subnet{}

	q := dbi.Model(&subnets).
		Join("INNER JOIN local_subnet AS ls ON ls.subnet_id = subnet.id").
		Relation("LocalSubnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Relation("SharedNetwork").
		Where("ls.daemon_id = ?", daemonID)

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting subnets by daemon ID %d", daemonID)
		return nil, err
	}
	return subnets, err
}

// Fetches the subnet by prefix from the database.
func GetSubnetsByPrefix(dbi dbops.DBI, prefix string) ([]Subnet, error) {
	subnets := []Subnet{}
	err := dbi.Model(&subnets).
		Relation("LocalSubnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Relation("SharedNetwork").
		Where("subnet.prefix = ?", prefix).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting subnets with prefix %s", prefix)
		return nil, err
	}
	return subnets, err
}

// Fetches all subnets belonging to a given family. If the family is set to 0
// it fetches both IPv4 and IPv6 subnet.
func GetAllSubnets(dbi dbops.DBI, family int) ([]Subnet, error) {
	subnets := []Subnet{}
	q := dbi.Model(&subnets).
		Relation("LocalSubnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Relation("LocalSubnets.Daemon.App.Machine").
		Relation("SharedNetwork").
		OrderExpr("id ASC")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all subnets for family %d", family)
		return nil, err
	}
	return subnets, err
}

// Fetches all global subnets, i.e., subnets that do not belong to shared
// networks. If the family is set to 0 it fetches both IPv4 and IPv6 subnet.
func GetGlobalSubnets(dbi dbops.DBI, family int) ([]Subnet, error) {
	subnets := []Subnet{}
	q := dbi.Model(&subnets).
		Relation("LocalSubnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		OrderExpr("id ASC").
		Where("subnet.shared_network_id IS NULL")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting global (top-level) subnets for family %d", family)
		return nil, err
	}
	return subnets, nil
}

// Container for values filtering subnets fetched by page.
type SubnetsByPageFilters struct {
	AppID         *int64
	LocalSubnetID *int64
	Family        *int64
	Text          *string
}

// Shorthand to set the IPv4 family.
func (f *SubnetsByPageFilters) SetIPv4Family() {
	family := int64(4)
	f.Family = &family
}

// Shorthand to set the IPv6 family.
func (f *SubnetsByPageFilters) SetIPv6Family() {
	family := int64(6)
	f.Family = &family
}

// Fetches a collection of subnets from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. The filters object is used to filter subnets. The nil value disables
// such filtering. The sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting. If SortDirAny is used then ASC
// order is used. This function returns a collection of subnets, the
// total number of subnets and error.
func GetSubnetsByPage(dbi dbops.DBI, offset, limit int64, filters *SubnetsByPageFilters, sortField string, sortDir SortDirEnum) ([]Subnet, int64, error) {
	if filters == nil {
		filters = &SubnetsByPageFilters{}
	}

	subnets := []Subnet{}
	q := dbi.Model(&subnets).Distinct()

	if filters.AppID != nil || filters.LocalSubnetID != nil || filters.Text != nil {
		q = q.Join("INNER JOIN local_subnet AS ls ON subnet.id = ls.subnet_id")
	}
	// When filtering by appID we also need the local_subnet table as it holds the
	// application identifier.
	if filters.AppID != nil {
		q = q.Join("INNER JOIN daemon AS d ON ls.daemon_id = d.id")
	}
	// Pools are also required when trying to filter by text.
	if filters.Text != nil {
		q = q.Join("LEFT JOIN address_pool AS ap ON ls.id = ap.local_subnet_id")
	}
	// Include pools, shared network the subnets belong to, local subnet info
	// and the associated apps in the results.
	q = q.Relation("SharedNetwork").
		Relation("LocalSubnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.Machine")

	// Applicable family values are 4 and 6.
	if filters.Family != nil {
		q = q.Where("family(subnet.prefix) = ?", *filters.Family)
	}

	// Filter by appID.
	if filters.AppID != nil {
		q = q.Where("d.app_id = ?", *filters.AppID)
	}

	// Filter by local subnet ID.
	if filters.LocalSubnetID != nil {
		q = q.Where("ls.local_subnet_id = ?", *filters.LocalSubnetID)
	}

	// Quick filtering by subnet prefix, pool ranges or shared network name.
	if filters.Text != nil {
		// The combination of the concat and host functions reconstruct the textual
		// version of the pool range as specified in Kea, e.g. 192.0.2.10-192.0.2.20.
		// This allows for quick filtering by strings like: 2.10-192.0.
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(subnet.prefix) LIKE ?", "%"+*filters.Text+"%").
				WhereOr("concat(host(ap.lower_bound), '-', host(ap.upper_bound)) LIKE ?", "%"+*filters.Text+"%").
				WhereOr("shared_network.name LIKE ?", "%"+*filters.Text+"%").
				WhereOr("ls.user_context->>'subnet-name' LIKE ?", "%"+*filters.Text+"%")
			return q, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("subnet", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	// This returns the limited results plus the total number of records.
	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting subnets by page")
	}
	return subnets, int64(total), err
}

// Get list of Subnets with LocalSubnets ordered by SharedNetworkID.
func GetSubnetsWithLocalSubnets(dbi dbops.DBI) ([]*Subnet, error) {
	subnets := []*Subnet{}
	q := dbi.Model(&subnets)
	// only selected columns are returned for performance reasons
	q = q.Column("id", "shared_network_id", "prefix")
	q = q.Relation("LocalSubnets")
	q = q.Order("shared_network_id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrap(err, "problem getting all subnets")
		return nil, err
	}
	return subnets, nil
}

// Get a list of existing subnet prefixes.
func GetSubnetPrefixes(dbi dbops.DBI) ([]string, error) {
	subnets := []string{}
	err := dbi.Model((*Subnet)(nil)).Column("prefix").Select(&subnets)
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrap(err, "problem getting subnet prefixes")
	}
	return subnets, err
}

// Associates a daemon with the subnet having a specified ID and prefix
// in a transaction. Internally, the association is made via the local_subnet
// table which holds the information about the subnet from the given daemon
// perspective, local subnet id, statistics etc.
func addDaemonToSubnet(dbi dbops.DBI, subnet *Subnet, daemon *Daemon) error {
	localSubnetID := int64(0)
	// If the prefix is available we should try to match the subnet prefix
	// with the app's configuration and retrieve the local subnet id from
	// there.
	if len(subnet.Prefix) > 0 {
		localSubnetID = daemon.GetLocalSubnetID(subnet.Prefix)
	}
	localSubnet := LocalSubnet{
		SubnetID:      subnet.ID,
		DaemonID:      daemon.ID,
		LocalSubnetID: localSubnetID,
	}
	// Try to insert. If such association already exists we could maybe do
	// nothing, but we do update instead to force setting the new value
	// of the local_subnet_id if it has changed.
	_, err := dbi.Model(&localSubnet).
		Column("subnet_id").
		Column("daemon_id").
		Column("local_subnet_id").
		OnConflict("(daemon_id, subnet_id) DO UPDATE").
		Set("daemon_id = EXCLUDED.daemon_id").
		Set("local_subnet_id = EXCLUDED.local_subnet_id").
		Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem associating the daemon %d with the subnet %s",
			daemon.ID, subnet.Prefix)
	}
	return err
}

// Associates a daemon with the subnet having a specified ID and prefix.
// It begins a new transaction when dbi has a *pg.DB type or uses an existing
// transaction when dbi has a *pg.Tx type.
func AddDaemonToSubnet(dbi dbops.DBI, subnet *Subnet, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addDaemonToSubnet(tx, subnet, daemon)
		})
	}
	return addDaemonToSubnet(dbi.(*pg.Tx), subnet, daemon)
}

// Dissociates a daemon from the subnet having a specified id.
// The first returned value indicates if any row was removed from the
// local_subnet table.
func DeleteDaemonFromSubnet(dbi dbops.DBI, subnetID int64, daemonID int64) (bool, error) {
	localSubnet := &LocalSubnet{
		DaemonID: daemonID,
		SubnetID: subnetID,
	}
	rows, err := dbi.Model(localSubnet).
		Where("daemon_id = ?", daemonID).
		Where("subnet_id = ?", subnetID).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the daemon with ID %d from the subnet with %d",
			daemonID, subnetID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Dissociates a daemon from the subnets. The first returned value
// indicates if any row was removed from the local_subnet table.
func DeleteDaemonFromSubnets(dbi dbops.DBI, daemonID int64) (int64, error) {
	result, err := dbi.Model((*LocalSubnet)(nil)).
		Where("daemon_id = ?", daemonID).
		Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting daemon %d from subnets", daemonID)
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Finds and returns an app associated with a subnet having the specified id.
func (s *Subnet) GetApp(appID int64) *App {
	for _, s := range s.LocalSubnets {
		daemon := s.Daemon
		if daemon.App != nil && daemon.App.ID == appID {
			return daemon.App
		}
	}
	return nil
}

// Iterates over the provided slice of subnets and stores them in the database
// if they are not there yet. In addition, it associates the subnets with the
// specified Kea application. Returns a list of added subnets.
func commitSubnetsIntoDB(tx *pg.Tx, networkID int64, subnets []Subnet) (addedSubnets []*Subnet, err error) {
	for i := range subnets {
		subnet := &subnets[i]
		if networkID != 0 {
			subnet.SharedNetworkID = networkID
		}

		if subnet.ID == 0 {
			err = AddSubnet(tx, subnet)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected subnet %s to the database",
					subnet.Prefix)
				return nil, err
			}
			addedSubnets = append(addedSubnets, subnet)
		} else {
			err = updateSubnet(tx, subnet)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to update detected subnet %s in the database",
					subnet.Prefix)
				return nil, err
			}
		}
		err = AddLocalSubnets(tx, subnet)
		if err != nil {
			return nil, err
		}

		err = CommitSubnetHostsIntoDB(tx, subnet)
		if err != nil {
			return nil, err
		}
	}
	return addedSubnets, nil
}

// Iterates over the shared networks, subnets and hosts and commits them to the database.
// In addition it associates them with the specified app. Returns a list of added subnets.
// This function runs all database operations in a transaction.
func commitNetworksIntoDB(tx *pg.Tx, networks []SharedNetwork, subnets []Subnet) ([]*Subnet, error) {
	var (
		addedSubnets      []*Subnet
		addedSubnetsToNet []*Subnet
		err               error
	)

	// Go over the networks that the Kea daemon belongs to.
	for i := range networks {
		network := &networks[i]
		if network.ID == 0 {
			// This is new shared network. Add it to the database.
			err = AddSharedNetwork(tx, network)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected shared network %s to the database",
					network.Name)
				return nil, err
			}
		} else {
			err = UpdateSharedNetwork(tx, network)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to update shared network %s in the database",
					network.Name)
				return nil, err
			}
		}
		if err = AddLocalSharedNetworks(tx, network); err != nil {
			return nil, err
		}
		// Associate subnets with the daemon.
		addedSubnetsToNet, err = commitSubnetsIntoDB(tx, network.ID, network.Subnets)
		if err != nil {
			return nil, err
		}
		addedSubnets = append(addedSubnets, addedSubnetsToNet...)
	}

	// Finally, add top level subnets to the database and associate them with
	// the Kea daemon.
	addedSubnetsToNet, err = commitSubnetsIntoDB(tx, 0, subnets)
	if err != nil {
		return nil, err
	}
	addedSubnets = append(addedSubnets, addedSubnetsToNet...)

	return addedSubnets, nil
}

// Iterates over the shared networks, subnets and hosts and commits them to the database.
// In addition it associates them with the specified daemon. Returns a list of added subnets.
func CommitNetworksIntoDB(dbi dbops.DBI, networks []SharedNetwork, subnets []Subnet) (addedSubnets []*Subnet, err error) {
	if db, ok := dbi.(*pg.DB); ok {
		err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			addedSubnets, err = commitNetworksIntoDB(tx, networks, subnets)
			return err
		})
		return
	}
	addedSubnets, err = commitNetworksIntoDB(dbi.(*pg.Tx), networks, subnets)
	return
}

// Fetch all local subnets for indicated app.
func GetAppLocalSubnets(dbi dbops.DBI, appID int64) ([]*LocalSubnet, error) {
	subnets := []*LocalSubnet{}
	q := dbi.Model(&subnets)
	q = q.Join("INNER JOIN daemon AS d ON local_subnet.daemon_id = d.id")
	// only selected columns are returned while stats columns are skipped for performance reasons (they are pretty big json fields)
	q = q.Column("local_subnet.id", "local_subnet.daemon_id", "local_subnet.subnet_id", "local_subnet.local_subnet_id")
	q = q.Relation("Subnet")
	q = q.Relation("Daemon.App")
	q = q.Where("d.app_id = ?", appID)

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all local subnets for app %d", appID)
		return nil, err
	}
	return subnets, nil
}

// Update stats pulled for given local subnet.
func (lsn *LocalSubnet) UpdateStats(dbi dbops.DBI, stats SubnetStats) error {
	lsn.Stats = stats
	lsn.StatsCollectedAt = storkutil.UTCNow()
	q := dbi.Model(lsn)
	q = q.Column("stats", "stats_collected_at")
	q = q.WherePK()
	result, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating stats in local subnet: [daemon:%d, subnet:%d, local subnet:%d]",
			lsn.DaemonID, lsn.SubnetID, lsn.LocalSubnetID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "local subnet: [daemon:%d, subnet:%d, local subnet:%d] does not exist",
			lsn.DaemonID, lsn.SubnetID, lsn.LocalSubnetID)
	}
	return err
}

// Update statistics in Subnet.
func (s *Subnet) UpdateStatistics(dbi dbops.DBI, statistics utilizationStats) error {
	addrUtilization := statistics.GetAddressUtilization()
	pdUtilization := statistics.GetDelegatedPrefixUtilization()
	s.AddrUtilization = int16(addrUtilization * 1000)
	s.PdUtilization = int16(pdUtilization * 1000)
	s.Stats = statistics.GetStatistics()
	s.StatsCollectedAt = time.Now().UTC()
	q := dbi.Model(s)
	q = q.Column("addr_utilization", "pd_utilization", "stats", "stats_collected_at")
	q = q.WherePK()
	result, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating statistics in the subnet: %d",
			s.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "subnet with ID %d does not exist", s.ID)
	}
	return err
}

// Deletes subnets which are not associated with any apps. Returns deleted subnet
// count and an error.
func DeleteOrphanedSubnets(dbi dbops.DBI) (int64, error) {
	subquery := dbi.Model(&[]LocalSubnet{}).
		Column("id").
		Limit(1).
		Where("subnet.id = local_subnet.subnet_id")
	result, err := dbi.Model(&[]Subnet{}).
		Where("(?) IS NULL", subquery).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting orphaned subnets")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Delete subnet by ID.
func DeleteSubnet(dbi dbops.DBI, subnetID int64) error {
	subnet := &Subnet{
		ID: subnetID,
	}
	result, err := dbi.Model(subnet).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the subnet with ID %d", subnetID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "subnet with ID %d does not exist", subnetID)
	}
	return err
}

// Finds a maximum local subnet ID in the database. It returns 0, if no subnet IDs are found.
func GetMaxLocalSubnetID(dbi dbops.DBI) (int64, error) {
	count := int64(0)
	err := dbi.Model((*LocalSubnet)(nil)).ColumnExpr("MAX(local_subnet_id)").Select(&count)
	if err != nil {
		return 0, pkgerrors.Wrapf(err, "problem getting max local subnet ID from the database")
	}
	return count, err
}
