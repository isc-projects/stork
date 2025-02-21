package apps

import (
	"sync"

	"github.com/pkg/errors"
)

// A type representing a configuration lock key.
type LockKey int64

// Low-level mechanism for locking the daemons' configurations.
// For high-level mechanism that utilizes the context look at the ManagerLocker
// interface.
// Locker must be thread-safe.
type DaemonLocker interface {
	// Locks the daemons' configurations for update.
	Lock(userID int64, daemonIDs ...int64) (LockKey, error)
	// Unlocks the daemons' configurations.
	Unlock(key LockKey, daemonIDs ...int64) error
	// Checks if the configuration of the specified daemon is already locked
	// The first returned parameter is an ID of the user who owns the lock.
	// It is equal to 0 when the lock is not present.
	IsLocked(daemonID int64) (int64, bool)
}

// Represents a configuration lock for a user.
type configLock struct {
	key    LockKey
	userID int64
}

// Implementation of the DaemonLocker interface.
// It is thread-safe.
type daemonLocker struct {
	// A map holding acquired locks for daemons. The map key is an
	// ID of the daemon for which the lock has been acquired.
	locks map[int64]configLock
	// Last generated lock key.
	key LockKey
	// Access mutex used in the exported methods.
	mutex sync.RWMutex
}

// Constructs a new instance of the daemonLocker.
func NewDaemonLocker() DaemonLocker {
	return &daemonLocker{
		locks: make(map[int64]configLock),
	}
}

// Generates a key for a newly acquired lock. It is called internally
// by the Lock() function.
func (locker *daemonLocker) generateKey() LockKey {
	locker.key++
	return locker.key
}

// Checks if the configuration of the specified daemon is already locked
// for updates. The first returned parameter is an ID of the user who
// owns the lock. It is equal to 0 when the lock is not present.
func (locker *daemonLocker) IsLocked(daemonID int64) (int64, bool) {
	locker.mutex.RLock()
	defer locker.mutex.RUnlock()

	if lock, ok := locker.locks[daemonID]; ok {
		return lock.userID, true
	}
	return 0, false
}

// Attempts to lock configurations of the specified daemons.
// If an attempt to lock any of the configurations fails, it will remove
// already acquired locks and return an error.
func (locker *daemonLocker) Lock(userID int64, daemonIDs ...int64) (LockKey, error) {
	locker.mutex.Lock()
	defer locker.mutex.Unlock()

	// Lock the daemons' configurations.
	key := locker.generateKey()

	for _, daemonID := range daemonIDs {
		// Try to acquire a lock for each daemon.
		if err := locker.lock(key, userID, daemonID); err != nil {
			// Locking failed. Remove the applied locks.
			for _, innerDaemonID := range daemonIDs {
				if innerDaemonID == daemonID {
					break
				}
				delete(locker.locks, innerDaemonID)
			}
			return 0, err
		}
	}
	return key, nil
}

// Attempts to acquire a lock on the specified daemon's configuration.
// It returns an error if the lock exists already. This function is
// called internally from the Lock() function.
func (locker *daemonLocker) lock(key LockKey, userID int64, daemonID int64) error {
	// Check if the daemon configuration has been locked already.
	if userID, locked := locker.locks[daemonID]; locked {
		return errors.Errorf("configuration for daemon %d is locked for updates by user %d", daemonID, userID)
	}

	// Acquire the lock.
	locker.locks[daemonID] = configLock{key: key, userID: userID}
	return nil
}

// Unlocks the configurations of the specified daemons if the key stored in the
// context matches.
func (locker *daemonLocker) Unlock(key LockKey, daemonIDs ...int64) error {
	locker.mutex.Lock()
	defer locker.mutex.Unlock()

	for _, daemonID := range daemonIDs {
		if lock, ok := locker.locks[daemonID]; ok && lock.key == key {
			delete(locker.locks, daemonID)
		}
	}
	return nil
}
