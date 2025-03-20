package config

import (
	"sync"

	"github.com/pkg/errors"
)

// A type representing a configuration lock key.
type LockKey int64

// Low-level mechanism for locking the daemons' configurations.
// For high-level mechanism that utilizes the context lock at the ManagerLocker
// interface.
// Locker must be thread-safe.
type DaemonLocker interface {
	// Locks the daemons' configurations for update.
	Lock(daemonIDs ...int64) (LockKey, error)
	// Unlocks the daemons' configurations.
	Unlock(key LockKey, daemonIDs ...int64) error
	// Checks if the configuration of the specified daemon is already locked.
	IsLocked(daemonID int64) bool
}

// Implementation of the DaemonLocker interface.
// It is thread-safe.
type daemonLocker struct {
	// A map holding acquired locks for daemons. The map key is an
	// ID of the daemon for which the lock has been acquired.
	locks map[int64]LockKey
	// Last generated lock key.
	key LockKey
	// Access mutex used in the exported methods.
	mutex sync.RWMutex
}

// Constructs a new instance of the daemonLocker.
func NewDaemonLocker() DaemonLocker {
	return &daemonLocker{
		locks: make(map[int64]LockKey),
	}
}

// Generates a key for a newly acquired lock. It is called internally
// by the Lock() function.
func (locker *daemonLocker) generateKey() LockKey {
	locker.key++
	return locker.key
}

// Checks if the configuration of the specified daemon is already locked
// for updates.
func (locker *daemonLocker) IsLocked(daemonID int64) bool {
	locker.mutex.RLock()
	defer locker.mutex.RUnlock()

	_, ok := locker.locks[daemonID]
	return ok
}

// Attempts to lock configurations of the specified daemons.
// If an attempt to lock any of the configurations fails, it will remove
// already acquired locks and return an error.
func (locker *daemonLocker) Lock(daemonIDs ...int64) (LockKey, error) {
	locker.mutex.Lock()
	defer locker.mutex.Unlock()

	// Lock the daemons' configurations.
	key := locker.generateKey()

	for _, daemonID := range daemonIDs {
		// Try to acquire a lock for each daemon.
		if err := locker.lock(key, daemonID); err != nil {
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
func (locker *daemonLocker) lock(key LockKey, daemonID int64) error {
	// Check if the daemon configuration has been locked already.
	if _, locked := locker.locks[daemonID]; locked {
		return errors.Errorf("configuration for daemon %d is already locked", daemonID)
	}

	// Acquire the lock.
	locker.locks[daemonID] = key
	return nil
}

// Unlocks the configurations of the specified daemons if the key stored in the
// context matches.
func (locker *daemonLocker) Unlock(key LockKey, daemonIDs ...int64) error {
	locker.mutex.Lock()
	defer locker.mutex.Unlock()

	for _, daemonID := range daemonIDs {
		if lockedKey, ok := locker.locks[daemonID]; ok && lockedKey == key {
			delete(locker.locks, daemonID)
		}
	}
	return nil
}
