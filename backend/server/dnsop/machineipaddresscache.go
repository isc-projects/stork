package dnsop

import (
	"net"
	"strings"
	"sync"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Caches IP address to machines mappings to use in translating the
// client and server IP addresses appearing the XFR states to
// respective machines. This solution is aimed at avoiding excessive
// database queries during the zone transfer tracking. The cache should
// be periodically refreshed. However, if the queried IP address is not
// found, the cache can pull this address from the database and update
// itself without the need to re-populate the whole cache.
type machineIPAddressCache struct {
	db          *pg.DB
	ipAddresses map[string][]dbmodel.Machine
	mutex       sync.RWMutex
}

// Instantiates a new machine IP address cache.
func newMachineIPAddressCache(db *pg.DB) *machineIPAddressCache {
	return &machineIPAddressCache{
		db: db,
	}
}

// Populates the cache. It queries the database for all IP addresses and
// stores them in the cache. The old cache is discarded. The local loopback
// addresses are not included in the cache.
func (cache *machineIPAddressCache) populate() error {
	ipAddresses, err := dbmodel.GetMachineNetworkInterfaceIPAddresses(cache.db, dbmodel.MachineNetworkInterfaceIPAddressRelationMachine)
	if err != nil {
		return errors.WithMessage(err, "failed to populate the cache mapping IP addresses to machines")
	}
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	// Re-create the cache.
	cache.ipAddresses = make(map[string][]dbmodel.Machine)
	// Save all IP addresses to the cache.
	for _, ipAddress := range ipAddresses {
		if ipAddress.Interface == nil || ipAddress.Interface.Machine == nil || net.Flags(ipAddress.Interface.Flags)&net.FlagLoopback != 0 {
			continue
		}
		// Ignore the prefix length.
		key, _, _ := strings.Cut(ipAddress.IPAddress, "/")
		cache.ipAddresses[key] = append(cache.ipAddresses[key], *ipAddress.Interface.Machine)
	}
	return nil
}

// Returns the machines having an interface with the given IP address.
// The specified ipAddress must exclude the prefix length. The local
// loopback addresses are not included in the cache. If the IP address is
// recognized as a loopback address, the function returns false in the
// second return value.
func (cache *machineIPAddressCache) getMachines(ipAddress string) ([]dbmodel.Machine, bool) {
	parsedIP := storkutil.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, false
	}
	if parsedIP.IP.IsLoopback() {
		return nil, true
	}
	cache.mutex.RLock()
	// Most of the time this should be successful if the cache is regularly refreshed.
	machines, ok := cache.ipAddresses[ipAddress]
	cache.mutex.RUnlock()
	if ok {
		return append([]dbmodel.Machine(nil), machines...), false
	}

	// The cache doesn't have this IP address. Try to get it from the database.
	dbMachines, err := dbmodel.GetMachinesByNetworkInterfaceIPAddress(cache.db, ipAddress, dbmodel.MachineRelationNetworkInterfacesIPAddresses)
	if err != nil {
		return nil, false
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// When we unlocked the read lock, the cache might have been updated by another
	// goroutine. Let's check if the cache now has this IP address.
	if machines, ok = cache.ipAddresses[ipAddress]; ok {
		return append([]dbmodel.Machine(nil), machines...), false
	}
	// Let's make sure that the cache is initialized.
	if cache.ipAddresses == nil {
		cache.ipAddresses = make(map[string][]dbmodel.Machine)
	}
	// Save the machines gathered from the database to the cache.
	cache.ipAddresses[ipAddress] = dbMachines
	// Return the machines.
	return append([]dbmodel.Machine(nil), dbMachines...), false
}
