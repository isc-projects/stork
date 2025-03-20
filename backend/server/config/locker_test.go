package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the daemon locker can be instantiated.
func TestNewDaemonLocker(t *testing.T) {
	// Arrange & Act
	locker := NewDaemonLocker()

	// Assert
	require.NotNil(t, locker)
	require.NotNil(t, locker.(*daemonLocker).locks)
	require.Zero(t, locker.(*daemonLocker).key)
}

// Test that locking the daemon configuration works.
func TestDaemonLockerLock(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()

	// Act
	key, err := locker.Lock(42)

	// Assert
	require.NoError(t, err)
	require.NotZero(t, key)
	require.True(t, locker.IsLocked(42))
}

// Test that multiple daemon configurations can be locked.
func TestDaemonLockerLockMultiple(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	daemonIDs := []int64{1, 2, 3}

	// Act
	key, err := locker.Lock(daemonIDs...)

	// Assert
	require.NoError(t, err)
	require.NotZero(t, key)

	for _, daemonID := range daemonIDs {
		require.True(t, locker.IsLocked(daemonID))
	}
}

// Test that locking already locked daemon configuration fails.
func TestDaemonLockerLockAlreadyLocked(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	locker.Lock(42)

	// Act
	key, err := locker.Lock(42)

	// Assert
	require.ErrorContains(t, err, "is already locked")
	require.Zero(t, key)
}

// Test that unlocking the daemon configuration works.
func TestDaemonLockerUnlock(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	key, _ := locker.Lock(42)

	// Act
	err := locker.Unlock(key, 42)

	// Assert
	require.NoError(t, err)
	require.False(t, locker.IsLocked(42))
}

// Test that unlocking multiple daemon configurations works.
func TestDaemonLockerUnlockMultiple(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	daemonIDs := []int64{1, 2, 3}
	key, _ := locker.Lock(daemonIDs...)

	// Act
	err := locker.Unlock(key, daemonIDs...)

	// Assert
	require.NoError(t, err)

	for _, daemonID := range daemonIDs {
		require.False(t, locker.IsLocked(daemonID))
	}
}

// Test that unlocking not locked daemon configuration throws no error.
func TestDaemonLockerUnlockNotLocked(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	var key LockKey = 1

	// Act
	err := locker.Unlock(key, 42)

	// Assert
	require.NoError(t, err)
}

// Test that unlocking no daemon configurations throws no error.
func TestDaemonLockerUnlockNone(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	var key LockKey = 1

	// Act
	err := locker.Unlock(key)

	// Assert
	require.NoError(t, err)
}

// Test that unlocking with wrong key throws no error.
func TestDaemonLockerUnlockWrongKey(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	key, _ := locker.Lock(42)
	wrongKey := key + 1

	// Act
	err := locker.Unlock(wrongKey, 42)

	// Assert
	require.NoError(t, err)
	require.True(t, locker.IsLocked(42))
}

// Test that the already locked daemons are unlocked if locking fails.
func TestDaemonLockerLockFailed(t *testing.T) {
	// Arrange
	locker := NewDaemonLocker()
	_, _ = locker.Lock(42)

	// Act
	key, err := locker.Lock(24, 42, 4224)

	// Assert
	require.ErrorContains(t, err, "42 is already locked")
	require.Zero(t, key)
	require.False(t, locker.IsLocked(24))
	require.False(t, locker.IsLocked(4224))
}
