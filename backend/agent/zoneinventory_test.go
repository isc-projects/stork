package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"path"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"isc.org/stork/appdata/bind9stats"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// This function generates a collection of zones used in the benchmarks.
// The function argument specifies the number of zones to be generated.
func generateRandomZones(num int) []*bind9stats.Zone {
	generatedZones := testutil.GenerateRandomZones(num)
	var zones []*bind9stats.Zone
	for _, generatedZone := range generatedZones {
		zones = append(zones, &bind9stats.Zone{
			ZoneName: generatedZone.Name,
			Class:    generatedZone.Class,
			Serial:   generatedZone.Serial,
			Type:     generatedZone.Type,
			Loaded:   time.Date(2025, 1, 1, 15, 19, 20, 0, time.UTC),
		})
	}
	return zones
}

// Sort zones and return the one at specified index.
func getOrderedZoneByIndex(zones []*bind9stats.Zone, index int) *bind9stats.Zone {
	slices.SortFunc(zones, func(zone1, zone2 *bind9stats.Zone) int {
		return storkutil.CompareNames(zone1.Name(), zone2.Name())
	})
	return zones[index]
}

// Setup stub response from the BIND9 stats.
func setGetViewsResponseOK(t *testing.T, response map[string]any) (*bind9StatsClient, func()) {
	zones, err := json.Marshal(response)
	require.NoError(t, err)

	gock.New("http://localhost:5380/").
		Get("json/v1").
		Persist().
		Reply(http.StatusOK).
		AddHeader("Content-Type", "application/json").
		BodyString(string(zones))
	bind9StatsClient := NewBind9StatsClient()
	gock.InterceptClient(bind9StatsClient.innerClient.GetClient())

	return bind9StatsClient, func() {
		gock.Off()
	}
}

// Setup stub response from the BIND9 stats (benchmark edition).
func setBenchmarkGetViewsResponseOK(b *testing.B, response map[string]any) (*bind9StatsClient, func()) {
	zones, err := json.Marshal(response)
	if err != nil {
		b.Fatal(err)
	}

	gock.New("http://localhost:5380/").
		Get("json/v1").
		Persist().
		Reply(http.StatusOK).
		AddHeader("Content-Type", "application/json").
		BodyString(string(zones))
	bind9StatsClient := NewBind9StatsClient()
	gock.InterceptClient(bind9StatsClient.innerClient.GetClient())

	return bind9StatsClient, func() {
		gock.Off()
	}
}

// Test zoneInventoryBusyError.
func TestZoneInventoryBusyError(t *testing.T) {
	require.ErrorContains(t, newZoneInventoryBusyError(newZoneInventoryStateInitial(), newZoneInventoryStateReceivingZones()),
		"cannot transition to the RECEIVING_ZONES state while the zone inventory is in INITIAL state")
}

// Test zoneInventoryNotInitedError.
func TestZoneInventoryNotInitedError(t *testing.T) {
	require.ErrorContains(t, newZoneInventoryNotInitedError(),
		"zone inventory has not been initialized yet")
}

// Test zoneInventoryNoDiskStorageError.
func TestZoneInventoryNoDiskStorageError(t *testing.T) {
	require.ErrorContains(t, newZoneInventoryNoDiskStorageError(),
		"zone inventory has no persistent storage")
}

// Test checking that the state is initial.
func TestZoneInventoryStateIsInitial(t *testing.T) {
	require.True(t, newZoneInventoryStateInitial().isInitial())
	require.False(t, newZoneInventoryStateLoaded(time.Time{}).isInitial())
	require.False(t, newZoneInventoryStateLoading().isInitial())
	require.False(t, newZoneInventoryStateLoadingErred(nil).isInitial())
	require.False(t, newZoneInventoryStatePopulated().isInitial())
	require.False(t, newZoneInventoryStatePopulating().isInitial())
	require.False(t, newZoneInventoryStatePopulatingErred(nil).isInitial())
	require.False(t, newZoneInventoryStateReceivingZones().isInitial())
	require.False(t, newZoneInventoryStateReceivedZones().isInitial())
}

// Test checking that the state is one of the long lasting operations.
func TestZoneInventoryStateIsLongLasting(t *testing.T) {
	require.False(t, newZoneInventoryStateInitial().isLongLasting())
	require.False(t, newZoneInventoryStateLoaded(time.Time{}).isLongLasting())
	require.True(t, newZoneInventoryStateLoading().isLongLasting())
	require.False(t, newZoneInventoryStateLoadingErred(nil).isLongLasting())
	require.False(t, newZoneInventoryStatePopulated().isLongLasting())
	require.True(t, newZoneInventoryStatePopulating().isLongLasting())
	require.False(t, newZoneInventoryStatePopulatingErred(nil).isLongLasting())
	require.True(t, newZoneInventoryStateReceivingZones().isLongLasting())
	require.False(t, newZoneInventoryStateReceivedZones().isLongLasting())
}

// Test checking that the state is populated or loaded.
func TestZoneInventoryStateIsReady(t *testing.T) {
	require.False(t, newZoneInventoryStateInitial().isReady())
	require.True(t, newZoneInventoryStateLoaded(time.Time{}).isReady())
	require.False(t, newZoneInventoryStateLoading().isReady())
	require.False(t, newZoneInventoryStateLoadingErred(nil).isReady())
	require.True(t, newZoneInventoryStatePopulated().isReady())
	require.False(t, newZoneInventoryStatePopulating().isReady())
	require.False(t, newZoneInventoryStatePopulatingErred(nil).isReady())
	require.False(t, newZoneInventoryStateReceivingZones().isReady())
	require.True(t, newZoneInventoryStateReceivedZones().isReady())
}

// Test checking that the state is erred.
func TestZoneInventoryStateIsErred(t *testing.T) {
	require.False(t, newZoneInventoryStateInitial().isErred())
	require.False(t, newZoneInventoryStateLoaded(time.Time{}).isErred())
	require.False(t, newZoneInventoryStateLoading().isErred())
	require.True(t, newZoneInventoryStateLoadingErred(nil).isErred())
	require.False(t, newZoneInventoryStatePopulated().isErred())
	require.False(t, newZoneInventoryStatePopulating().isErred())
	require.True(t, newZoneInventoryStatePopulatingErred(nil).isErred())
	require.False(t, newZoneInventoryStateReceivingZones().isErred())
	require.False(t, newZoneInventoryStateReceivedZones().isErred())
}

// Test storage configuration when memory and persistent storage is in use.
func TestZoneInventoryMemoryDiskStorage(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, storage)
	require.Equal(t, sandbox.BasePath, storage.disk.location)
}

// Test that an error is returned while creating a memory/disk storage under
// invalid path - a path to a file not a directory.
func TestZoneInventoryMemoryDiskStorageError(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	_, err := sandbox.Write("file.txt", "")
	require.NoError(t, err)
	storage, err := newZoneInventoryStorageMemoryDisk(path.Join(sandbox.BasePath, "file.txt"))
	require.Error(t, err)
	require.Nil(t, storage)
}

// Test zone inventory storage that saves data to disk and memory.
func TestZoneInventoryMemoryDiskStorageSaveLoadGetViews(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Save example zones on disk.
	views := bind9stats.NewViews([]*bind9stats.View{
		bind9stats.NewView("_default", []*bind9stats.Zone{
			{
				ZoneName: "zone1.example.org",
			},
			{
				ZoneName: "zone2.example.org",
			},
		}),
		bind9stats.NewView("_bind", []*bind9stats.Zone{
			{
				ZoneName: "zone2.example.org",
			},
		}),
	})
	err = storage.saveViews(views)
	require.NoError(t, err)

	// Replace the population time with some arbitrary time in the past.
	meta := ZoneInventoryMeta{
		PopulatedAt: time.Date(2002, 1, 1, 10, 15, 23, 0, time.UTC),
	}
	err = storage.disk.saveMeta(&meta)
	require.NoError(t, err)

	// Load the data into another zone inventory.
	loadedStorage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, loadedStorage)

	populatedAt, err := loadedStorage.loadViews()
	require.NoError(t, err)
	require.Equal(t, time.Date(2002, 1, 1, 10, 15, 23, 0, time.UTC), populatedAt)

	// Get the views and zones and ensure we get the same set of data.
	iterator := loadedStorage.getViewsIterator(nil)
	var capturedViews []bind9stats.ZoneIteratorAccessor
	for view, err := range iterator {
		require.NoError(t, err)
		capturedViews = append(capturedViews, view)
	}
	require.Equal(t, "_bind", capturedViews[0].GetViewName())
	zoneCount, err := capturedViews[0].GetZoneCount()
	require.NoError(t, err)
	require.EqualValues(t, 1, zoneCount)
	var zones []*bind9stats.Zone
	for zone, err := range capturedViews[0].GetZoneIterator(nil) {
		require.NoError(t, err)
		zones = append(zones, zone)
	}
	require.Len(t, zones, 1)
	require.Equal(t, "zone2.example.org", zones[0].Name())

	require.Equal(t, "_default", capturedViews[1].GetViewName())
	zoneCount, err = capturedViews[1].GetZoneCount()
	require.NoError(t, err)
	require.EqualValues(t, 2, zoneCount)
	zones = []*bind9stats.Zone{}
	for zone, err := range capturedViews[1].GetZoneIterator(nil) {
		require.NoError(t, err)
		zones = append(zones, zone)
	}
	require.Len(t, zones, 2)
	require.Equal(t, "zone1.example.org", zones[0].Name())
	require.Equal(t, "zone2.example.org", zones[1].Name())

	// Try selectively getting a zone.
	zone, err := loadedStorage.getZoneInView("_bind", "zone2.example.org")
	require.NoError(t, err)
	require.NotNil(t, zone)
	require.Equal(t, "zone2.example.org", zone.Name())

	// Try getting a non-existing zone.
	zone, err = loadedStorage.getZoneInView("_bind", "non-existing.example.org")
	require.NoError(t, err)
	require.Nil(t, zone)
}

// Test that the iterator returns an empty set of views when there are
// no views initialized.
func TestZoneInventoryMemoryDiskStorageGetViewsIteratorNoViews(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)

	iterator := storage.getViewsIterator(nil)
	var count int
	for range iterator {
		count++
	}
	require.Zero(t, count)
}

// Test storage creation when memory storage is in use.
func TestZoneInventoryMemoryStorage(t *testing.T) {
	storage := newZoneInventoryStorageMemory()
	require.NotNil(t, storage)
	require.Nil(t, storage.views)
}

// Test zone inventory storage that stores data in memory.
func TestZoneInventoryMemoryStorageSaveGetViews(t *testing.T) {
	storage := newZoneInventoryStorageMemory()
	require.NotNil(t, storage)

	// Save example zones in the storage.
	views := bind9stats.NewViews([]*bind9stats.View{
		bind9stats.NewView("_default", []*bind9stats.Zone{
			{
				ZoneName: "zone1.example.org",
			},
			{
				ZoneName: "zone2.example.org",
			},
		}),
		bind9stats.NewView("_bind", []*bind9stats.Zone{
			{
				ZoneName: "zone2.example.org",
			},
		}),
	})
	err := storage.saveViews(views)
	require.NoError(t, err)

	// Loading should fail because there is no persistent storage.
	_, err = storage.loadViews()
	require.Error(t, err)
	expectedError := &zoneInventoryNoDiskStorageError{}
	require.ErrorAs(t, err, &expectedError)

	// Get the views and zones.
	iterator := storage.getViewsIterator(nil)
	var capturedViews []bind9stats.ZoneIteratorAccessor
	for view, err := range iterator {
		require.NoError(t, err)
		capturedViews = append(capturedViews, view)
	}
	require.Equal(t, "_bind", capturedViews[0].GetViewName())
	zoneCount, err := capturedViews[0].GetZoneCount()
	require.NoError(t, err)
	require.EqualValues(t, 1, zoneCount)
	var zones []*bind9stats.Zone
	for zone, err := range capturedViews[0].GetZoneIterator(nil) {
		require.NoError(t, err)
		zones = append(zones, zone)
	}
	require.Len(t, zones, 1)
	require.Equal(t, "zone2.example.org", zones[0].Name())

	require.Equal(t, "_default", capturedViews[1].GetViewName())
	zoneCount, err = capturedViews[1].GetZoneCount()
	require.NoError(t, err)
	require.EqualValues(t, 2, zoneCount)
	zones = []*bind9stats.Zone{}
	for zone, err := range capturedViews[1].GetZoneIterator(nil) {
		require.NoError(t, err)
		zones = append(zones, zone)
	}
	require.Len(t, zones, 2)
	require.Equal(t, "zone1.example.org", zones[0].Name())
	require.Equal(t, "zone2.example.org", zones[1].Name())

	// Try selectively getting a zone.
	zone, err := storage.getZoneInView("_bind", "zone2.example.org")
	require.NoError(t, err)
	require.NotNil(t, zone)
	require.Equal(t, "zone2.example.org", zone.Name())

	// Try getting a non-existing zone.
	zone, err = storage.getZoneInView("_bind", "non-existing.example.org")
	require.NoError(t, err)
	require.Nil(t, zone)
}

// Test that the iterator returns an empty set of views when there are
// no views initialized.
func TestZoneInventoryMemoryStorageGetViewsIteratorNoViews(t *testing.T) {
	storage := newZoneInventoryStorageMemory()

	iterator := storage.getViewsIterator(nil)
	var count int
	for range iterator {
		count++
	}
	require.Zero(t, count)
}

// Test storage configuration when disk storage is in use.
func TestZoneInventoryDiskStorage(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, storage)
	require.Equal(t, sandbox.BasePath, storage.location)
}

// Test that an error is returned while creating a disk storage under
// invalid path - a path to a file not a directory.
func TestZoneInventoryDiskStorageError(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	_, err := sandbox.Write("file.txt", "")
	require.NoError(t, err)
	storage, err := newZoneInventoryStorageDisk(path.Join(sandbox.BasePath, "file.txt"))
	require.Error(t, err)
	require.Nil(t, storage)
}

// Test zone inventory storage that saves data to disk and not in memory.
func TestZoneInventoryDiskStorageSaveLoadGetViews(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Save example zones on disk.
	views := bind9stats.NewViews([]*bind9stats.View{
		bind9stats.NewView("_default", []*bind9stats.Zone{
			{
				ZoneName: "zone1.example.org",
			},
			{
				ZoneName: "zone2.example.org",
			},
		}),
		bind9stats.NewView("_bind", []*bind9stats.Zone{
			{
				ZoneName: "zone2.example.org",
			},
		}),
	})
	err = storage.saveViews(views)
	require.NoError(t, err)

	// Replace the population time with some arbitrary time in the past.
	meta := ZoneInventoryMeta{
		PopulatedAt: time.Date(2002, 1, 1, 10, 15, 23, 0, time.UTC),
	}
	err = storage.saveMeta(&meta)
	require.NoError(t, err)

	// Load the data into another zone inventory.
	loadedStorage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, loadedStorage)

	populatedAt, err := loadedStorage.loadViews()
	require.NoError(t, err)
	require.Equal(t, time.Date(2002, 1, 1, 10, 15, 23, 0, time.UTC), populatedAt)

	// Get the views and zones and ensure we get the same set of data.
	iterator := loadedStorage.getViewsIterator(nil)
	var capturedViews []bind9stats.ZoneIteratorAccessor
	for view, err := range iterator {
		require.NoError(t, err)
		capturedViews = append(capturedViews, view)
	}
	require.Equal(t, "_bind", capturedViews[0].GetViewName())
	zoneCount, err := capturedViews[0].GetZoneCount()
	require.NoError(t, err)
	require.EqualValues(t, 1, zoneCount)
	var zones []*bind9stats.Zone
	for zone, err := range capturedViews[0].GetZoneIterator(nil) {
		require.NoError(t, err)
		zones = append(zones, zone)
	}
	require.Len(t, zones, 1)
	require.Equal(t, "zone2.example.org", zones[0].Name())

	require.Equal(t, "_default", capturedViews[1].GetViewName())
	zoneCount, err = capturedViews[1].GetZoneCount()
	require.NoError(t, err)
	require.EqualValues(t, 2, zoneCount)
	zones = []*bind9stats.Zone{}
	for zone, err := range capturedViews[1].GetZoneIterator(nil) {
		require.NoError(t, err)
		zones = append(zones, zone)
	}
	require.Len(t, zones, 2)
	require.Equal(t, "zone1.example.org", zones[0].Name())
	require.Equal(t, "zone2.example.org", zones[1].Name())

	// Try selectively getting a zone.
	zone, err := loadedStorage.getZoneInView("_bind", "zone2.example.org")
	require.NoError(t, err)
	require.NotNil(t, zone)
	require.Equal(t, "zone2.example.org", zone.Name())

	// Try getting a non-existing zone.
	zone, err = loadedStorage.getZoneInView("_bind", "non-existing.example.org")
	require.NoError(t, err)
	require.Nil(t, zone)
}

// Test that an error is captured when there is an error reading from disk.
func TestZoneInventoryDiskStorageGetViewsIteratorError(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	// Close the sandbox so the files are no longer accessible.
	sandbox.Close()

	// Get the iterator. The first attempt to read the data should return
	// an error.
	iterator := storage.getViewsIterator(nil)
	var errs []error
	for zone, err := range iterator {
		require.Nil(t, zone)
		errs = append(errs, err)
		require.Error(t, err)
	}
	require.Len(t, errs, 1)
}

// Test instantiating zone inventory.
func TestNewZoneInventory(t *testing.T) {
	storage := newZoneInventoryStorageMemory()
	client := NewBind9StatsClient()
	inventory := newZoneInventory(storage, client, "myhost", 1234)
	defer inventory.awaitBackgroundTasks()
	require.NotNil(t, inventory)
	require.Equal(t, storage, inventory.storage)
	require.Equal(t, client, inventory.client)
	require.Equal(t, "myhost", inventory.host)
	require.EqualValues(t, 1234, inventory.port)
	require.NotNil(t, inventory.visitedStates)
	require.True(t, inventory.getCurrentState().isInitial())
}

// Test saving, reading and removing the inventory meta file.
func TestZoneInventoryStorageDiskMeta(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, storage)

	err = storage.removeMeta()
	require.NoError(t, err)

	meta := &ZoneInventoryMeta{
		PopulatedAt: time.Date(2002, 10, 2, 15, 11, 23, 0, time.UTC),
	}
	err = storage.saveMeta(meta)
	require.NoError(t, err)

	meta, err = storage.readMeta()
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, time.Date(2002, 10, 2, 15, 11, 23, 0, time.UTC), meta.PopulatedAt)

	err = storage.removeMeta()
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(sandbox.BasePath, zoneInventoryMetaFileName))
}

// Test transitioning the zone inventory between states.
func TestZoneInventoryTransition(t *testing.T) {
	storage := newZoneInventoryStorageMemory()
	client := NewBind9StatsClient()
	inventory := newZoneInventory(storage, client, "myhost", 1234)
	defer inventory.awaitBackgroundTasks()

	inventory.transition(newZoneInventoryStateLoaded(time.Time{}))
	inventory.transition(newZoneInventoryStateLoadingErred(newZoneInventoryNoDiskStorageError()))

	state := inventory.getVisitedState(zoneInventoryStateInitial)
	require.NotNil(t, state)
	require.Equal(t, zoneInventoryStateInitial, state.name)
	require.Nil(t, state.err)
	require.NotZero(t, state.createdAt)

	state = inventory.getVisitedState(zoneInventoryStateLoaded)
	require.NotNil(t, state)
	require.Equal(t, zoneInventoryStateLoaded, state.name)
	require.Nil(t, state.err)
	require.NotZero(t, state.createdAt)

	state = inventory.getVisitedState(zoneInventoryStateLoadingErred)
	require.NotNil(t, state)
	require.Equal(t, zoneInventoryStateLoadingErred, state.name)
	require.NotNil(t, state.err)
	require.ErrorIs(t, state.err, newZoneInventoryNoDiskStorageError())
	require.NotZero(t, state.createdAt)

	state = inventory.getVisitedState(zoneInventoryStateReceivedZones)
	require.Nil(t, state)
}

// Test populating the zones from the DNS server to memory and disk.
func TestZoneInventoryPopulateMemoryDisk(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	require.Equal(t, zoneInventoryStateInitial, inventory.getVisitedState(zoneInventoryStateInitial).name)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Make sure that view directories have been created.
	require.DirExists(t, path.Join(sandbox.BasePath, "_default"))
	require.DirExists(t, path.Join(sandbox.BasePath, "_bind"))

	// Make sure that the views contain zones.
	for _, zone := range defaultZones {
		require.FileExists(t, path.Join(sandbox.BasePath, "_default", zone.Name()))
	}
	for _, zone := range bindZones {
		require.FileExists(t, path.Join(sandbox.BasePath, "_bind", zone.Name()))
	}

	// Make sure that the inventory is in the correct state.
	require.Equal(t, zoneInventoryStatePopulated, inventory.getCurrentState().name)
}

// Test populating the zones from the DNS server to memory.
func TestZoneInventoryPopulateMemory(t *testing.T) {
	// Setup server response.
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": generateRandomZones(100),
			},
			"_bind": map[string]any{
				"zones": generateRandomZones(200),
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory with memory only.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory/memory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Make sure that the inventory is in the correct state.
	require.Equal(t, zoneInventoryStatePopulated, inventory.getCurrentState().name)
}

// Test populating the zones from the DNS server to disk.
func TestZoneInventoryPopulateDisk(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory with disk storage only.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory/disk.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Make sure that view directories have been created.
	require.DirExists(t, path.Join(sandbox.BasePath, "_default"))
	require.DirExists(t, path.Join(sandbox.BasePath, "_bind"))

	// Make sure that the views contain zones.
	for _, zone := range defaultZones {
		require.FileExists(t, path.Join(sandbox.BasePath, "_default", zone.Name()))
	}
	for _, zone := range bindZones {
		require.FileExists(t, path.Join(sandbox.BasePath, "_bind", zone.Name()))
	}
	// Make sure that the inventory is in the correct state.
	require.Equal(t, zoneInventoryStatePopulated, inventory.getCurrentState().name)
}

// Test that starting long lasting background operations fails while other
// long lasting operations are in progress.
func TestZoneInventoryPopulateLongLastingTaskConflict(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(5)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create the inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	// Create zone inventory.
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into inventory/disk.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	_, err = inventory.receiveZones(ctx, nil)
	require.NoError(t, err)
	defer cancel()

	require.Equal(t, zoneInventoryStateReceivingZones, inventory.getCurrentState().name)

	// Populating the zones should fail because the inventory is busy.
	_, err = inventory.populate(false)
	require.Error(t, err)
	var busyError *zoneInventoryBusyError
	require.ErrorAs(t, err, &busyError)
	require.Equal(t, zoneInventoryStateReceivingZones, inventory.getCurrentState().name)

	// Loading should fail too.
	_, err = inventory.load(false)
	require.Error(t, err)
	require.ErrorAs(t, err, &busyError)
	require.Equal(t, zoneInventoryStateReceivingZones, inventory.getCurrentState().name)

	// Receiving should fail.
	_, err = inventory.receiveZones(context.Background(), nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &busyError)
	require.Equal(t, zoneInventoryStateReceivingZones, inventory.getCurrentState().name)
}

// Test that awaitBackgroundTasks can be called following the zones population.
func TestZoneInventoryPopulateAwaitBackgroundTasks(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(5)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create the inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	// Create zone inventory.
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)

	// Populate the zones into inventory/disk.
	done, err := inventory.populate(true)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		// Await background tasks before finishing up populating the zones.
		// It should block until zones are populated.
		inventory.awaitBackgroundTasks()
	}()

	go func() {
		defer wg.Done()
		// Finish populating the zones.
		result := <-done
		require.NoError(t, result.err)
	}()
	// Wait for the background tasks to return.
	wg.Wait()

	// Make sure the zones have been populated.
	require.Equal(t, zoneInventoryStatePopulated, inventory.getCurrentState().name)
}

// Test loading the inventory from disk to memory.
func TestZoneInventoryLoadMemoryDisk(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones into disk.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Remove the zones from memory. That way we will be able to test if
	// the zones are reloaded.
	storage.memory.views.Views = nil

	// Load the zones from disk.
	done, err = inventory.load(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStateLoading {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Make sure that the views have been reloaded and contain zones.
	defaultView := storage.memory.views.GetView("_default")
	require.NotNil(t, defaultView)
	require.Len(t, defaultView.GetZoneNames(), 10)

	bindView := storage.memory.views.GetView("_bind")
	require.NotNil(t, bindView)
	require.Len(t, bindView.GetZoneNames(), 20)

	// Make sure that the inventory is in the correct state.
	require.Equal(t, zoneInventoryStateLoaded, inventory.getCurrentState().name)
}

// Test that loading the inventory fails when there is no disk storage.
func TestZoneInventoryLoadMemory(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory without persistent storage.
	storage := newZoneInventoryStorageMemory()
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the views/zones from DNS server to memory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Reset the views/zones in memory.
	storage.views.Views = nil

	// Attempting to load the views/zones should fail because there is
	// no disk storage.
	_, err = inventory.load(false)
	require.Error(t, err)
	require.ErrorIs(t, err, newZoneInventoryNoDiskStorageError())

	// Make sure that the inventory is in the correct state.
	require.Equal(t, zoneInventoryStatePopulated, inventory.getCurrentState().name)
}

// Test loading the inventory from disk.
func TestZoneInventoryLoadDisk(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory with persistent storage only.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the views/zones from DNS server to disk.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Replace the meta file to point way to the past.
	err = storage.saveMeta(&ZoneInventoryMeta{
		PopulatedAt: time.Date(2002, 3, 3, 15, 43, 1, 2, time.UTC),
	})
	require.NoError(t, err)

	// Load the zones. Technically, it should merely check whether the zones
	// have been populated and saved on disk, but not load them into memory.
	done, err = inventory.load(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStateLoading {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Make sure that the views have been reloaded and contain zones.
	zone, err := inventory.getZoneInView("_default", defaultZones[0].Name())
	require.NoError(t, err)
	require.NotNil(t, zone)

	zone, err = inventory.getZoneInView("_bind", bindZones[0].Name())
	require.NoError(t, err)
	require.NotNil(t, zone)

	// Make sure that the inventory is in the correct state and that
	// the inventory creation time was read from disk.
	require.Equal(t, zoneInventoryStateLoaded, inventory.getCurrentState().name)
	require.Equal(t, time.Date(2002, 3, 3, 15, 43, 1, 2, time.UTC), inventory.getCurrentState().createdAt)
}

// Test that awaitBackgroundTasks can be called during the zones loading.
func TestZoneInventoryLoadAwaitBackgroundTasks(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(5)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create the inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	// Create zone inventory.
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)

	// Populate the zones into inventory/disk.
	done, err := inventory.populate(false)
	require.NoError(t, err)

	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Load the inventory.
	done, err = inventory.load(true)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		// Await background tasks before finishing up loading the zones.
		// It should block until zones are loaded.
		inventory.awaitBackgroundTasks()
	}()
	go func() {
		defer wg.Done()
		// Finish loading the zones.
		result := <-done
		require.NoError(t, result.err)
	}()
	// Wait for the background tasks to return.
	wg.Wait()

	// Make sure that the long lasting operation was finished.
	require.Equal(t, zoneInventoryStateLoaded, inventory.getCurrentState().name)
}

// Test that it is allowed to try to get the zone from a view while
// a long lasting operation is running.
func TestGetZoneInViewDuringLongLastingTask(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the views/zones from DNS server to disk.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	t.Run("populating", func(t *testing.T) {
		done, err := inventory.populate(false)
		require.NoError(t, err)

		zone, err := inventory.getZoneInView("_bind", bindZones[0].Name())
		require.NoError(t, err)
		require.NotNil(t, zone)

		<-done
	})

	t.Run("loading", func(t *testing.T) {
		done, err := inventory.load(false)
		require.NoError(t, err)

		zone, err := inventory.getZoneInView("_bind", bindZones[0].Name())
		require.NoError(t, err)
		require.NotNil(t, zone)

		<-done
	})

	t.Run("receiving zones", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := inventory.receiveZones(ctx, nil)
		require.NoError(t, err)

		zone, err := inventory.getZoneInView("_bind", bindZones[0].Name())
		require.NoError(t, err)
		require.NotNil(t, zone)
	})
}

// Test that the zone inventory is sent zone by zone to the receiver
// over the channel from the in-memory storage.
func TestZoneInventoryReceiveZonesMemoryStorage(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create the inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones from the DNS server to the inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	t.Run("no filter", func(t *testing.T) {
		channel, err := inventory.receiveZones(context.Background(), nil)
		require.NoError(t, err)

		// Wait for the inventory to start sending zones.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that all zones have been received.
		require.Len(t, receivedZones, 30)
		for i, zone := range receivedZones {
			// Views should be sorted by name, so _bind goes first.
			if i < 20 {
				require.Equal(t, "_bind", zone.ViewName)
			} else {
				require.Equal(t, "_default", zone.ViewName)
			}
			require.EqualValues(t, 30, zone.TotalZoneCount)
		}
		// Make sure that the inventory is in the correct state.
		require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
	})

	t.Run("filter by view", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		channel, err := inventory.receiveZones(context.Background(), filter)
		require.NoError(t, err)

		// Wait for the inventory to start sending zones.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that all zones from the _bind view have been received.
		require.Len(t, receivedZones, 20)
		for _, zone := range receivedZones {
			require.Equal(t, "_bind", zone.ViewName)
			require.EqualValues(t, 20, zone.TotalZoneCount)
		}
		// Make sure that the inventory is in the correct state.
		require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
	})

	t.Run("filter by view and page", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		filter.SetLowerBound(getOrderedZoneByIndex(bindZones, 14).Name(), 15)
		channel, err := inventory.receiveZones(context.Background(), filter)
		require.NoError(t, err)

		// Wait for the inventory to start sending zones.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that the subset of zones has been returned.
		require.Len(t, receivedZones, 5)
		for index, zone := range receivedZones {
			require.Equal(t, "_bind", zone.ViewName)
			require.EqualValues(t, 20, zone.TotalZoneCount)
			require.Equal(t, getOrderedZoneByIndex(bindZones, index+15).Name(), zone.Name())
		}
		// Make sure that the inventory is in the correct state.
		require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
	})

	t.Run("page out of range", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		filter.SetLowerBound(getOrderedZoneByIndex(bindZones, 19).Name(), 20)
		channel, err := inventory.receiveZones(context.Background(), filter)
		require.NoError(t, err)

		// The inventory should almost immediately return.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivedZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that no zones have been returned.
		require.Empty(t, receivedZones)
	})
}

// Test that the zone inventory is sent zone by zone to the receiver
// over the channel from the disk storage.
func TestZoneInventoryReceiveZonesDiskStorage(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(10)
	bindZones := generateRandomZones(20)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create the inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones from the DNS server to the inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	t.Run("no filter", func(t *testing.T) {
		channel, err := inventory.receiveZones(context.Background(), nil)
		require.NoError(t, err)

		// Wait for the inventory to start sending zones.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that all zones have been received.
		require.Len(t, receivedZones, 30)
		for i, zone := range receivedZones {
			// Views should be sorted by name, so _bind goes first.
			if i < 20 {
				require.Equal(t, "_bind", zone.ViewName)
			} else {
				require.Equal(t, "_default", zone.ViewName)
			}
			require.EqualValues(t, 30, zone.TotalZoneCount)
		}
		// Make sure that the inventory is in the correct state.
		require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
	})

	t.Run("filter by view", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		channel, err := inventory.receiveZones(context.Background(), filter)
		require.NoError(t, err)

		// Wait for the inventory to start sending zones.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that all zones from the _bind view have been received.
		require.Len(t, receivedZones, 20)
		for _, zone := range receivedZones {
			// Views should be sorted by name, so _bind goes first.
			require.Equal(t, "_bind", zone.ViewName)
			require.EqualValues(t, 20, zone.TotalZoneCount)
		}
		// Make sure that the inventory is in the correct state.
		require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
	})

	t.Run("filter by view and page", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		filter.SetLowerBound(getOrderedZoneByIndex(bindZones, 14).Name(), 15)
		channel, err := inventory.receiveZones(context.Background(), filter)
		require.NoError(t, err)

		// Wait for the inventory to start sending zones.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Make sure that the second page with zones has been received.
		require.Len(t, receivedZones, 5)
		for index, zone := range receivedZones {
			require.Equal(t, "_bind", zone.ViewName)
			require.EqualValues(t, 20, zone.TotalZoneCount)
			require.Equal(t, getOrderedZoneByIndex(bindZones, index+15).Name(), zone.Name())
		}
		// Make sure that the inventory is in the correct state.
		require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
	})

	t.Run("page out of range", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		filter.SetLowerBound(getOrderedZoneByIndex(bindZones, 19).Name(), 20)
		channel, err := inventory.receiveZones(context.Background(), filter)
		require.NoError(t, err)

		// The inventory should almost immediately return.
		require.Eventually(t, func() bool {
			return inventory.getCurrentState().name == zoneInventoryStateReceivedZones
		}, time.Second, time.Millisecond)

		// Get the zones from the channel.
		var receivedZones []*bind9stats.ExtendedZone
		for result := range channel {
			require.NoError(t, result.err)
			receivedZones = append(receivedZones, result.zone)
		}
		// Should return no zones.
		require.Empty(t, receivedZones)
	})
}

// Test that receiving the zones over the channel can be cancelled.
func TestZoneInventoryReceiveZonesCancel(t *testing.T) {
	// Setup server response.
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": generateRandomZones(10),
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create the zone inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones from the DNS server to the inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Create the cancellable context.
	ctx, cancel := context.WithCancel(context.Background())

	// Start receiving the zones.
	channel, err := inventory.receiveZones(ctx, nil)
	require.NoError(t, err)

	// Wait for the inventory to start loading.
	require.Eventually(t, func() bool {
		return inventory.getCurrentState().name == zoneInventoryStateReceivingZones
	}, time.Second, time.Millisecond)

	// Get one zone from the channel.
	result := <-channel
	require.NoError(t, result.err)
	require.NotNil(t, result.zone)
	require.NotEmpty(t, result.zone.Name())

	// Cancel getting the zones.
	cancel()

	// It should be ok to read from the closed channel. It may be nil
	// or non-nil value, depending on whether cancel has taken effect
	// already.
	<-channel

	// The inventory should fall back to the RECEIVED_ZONES state.
	require.Eventually(t, func() bool {
		return inventory.getCurrentState().name == zoneInventoryStateReceivedZones
	}, time.Second, time.Millisecond)

	// Make sure that the inventory is in the correct state.
	require.Equal(t, zoneInventoryStateReceivedZones, inventory.getCurrentState().name)
}

// Test that an attempt to receive the zones from the inventory fails when
// the inventory wasn't populated.
func TestZoneInventoryReceiveZonesNotInited(t *testing.T) {
	// Create the inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), nil, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Collect the received zones in this slice.
	_, err := inventory.receiveZones(context.Background(), nil)
	require.Error(t, err)
	var notInitedError *zoneInventoryNotInitedError
	require.ErrorAs(t, err, &notInitedError)
	require.Equal(t, zoneInventoryStateInitial, inventory.getCurrentState().name)
}

// Test getting a selected zone from the inventory when the zones are stored in memory.
func TestZoneInventoryGetZoneInViewMemory(t *testing.T) {
	// Setup server response.
	randomZones := generateRandomZones(100)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": randomZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory in memory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones from the DNS server to the inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Wait for the completion.
	require.Eventually(t, func() bool {
		return inventory.getCurrentState().name == zoneInventoryStatePopulated
	}, time.Second, time.Millisecond)

	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Get a random zone from the existing ones.
	randomZone := randomZones[rand.Int64()%100]
	zone, err := inventory.getZoneInView("_default", randomZone.Name())
	require.NoError(t, err)
	require.NotNil(t, zone)
	require.Equal(t, randomZone.Name(), zone.Name())

	// Getting a non-existing zone should return nil.
	zone, err = inventory.getZoneInView("_default", "non.existing.zone")
	require.NoError(t, err)
	require.Nil(t, zone)
}

// Test getting a selected zone from the inventory when the zones are stored on disk.
func TestZoneInventoryGetZoneInViewDisk(t *testing.T) {
	// Setup server response.
	defaultZones := generateRandomZones(100)
	bindZones := generateRandomZones(200)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": defaultZones,
			},
			"_bind": map[string]any{
				"zones": bindZones,
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	// Create zone inventory in memory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
	require.NoError(t, err)
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)
	defer inventory.awaitBackgroundTasks()

	// Populate the zones from the DNS server to the inventory.
	done, err := inventory.populate(false)
	require.NoError(t, err)
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}

	// Wait for the completion.
	require.Eventually(t, func() bool {
		return inventory.getCurrentState().name == zoneInventoryStatePopulated
	}, time.Second, time.Millisecond)

	err = inventory.getCurrentState().err
	require.NoError(t, err)

	// Get a random zone from the existing ones.
	randomZone := bindZones[rand.Int64()%100]
	zone, err := inventory.getZoneInView("_bind", randomZone.Name())
	require.NoError(t, err)
	require.NotNil(t, zone)
	require.Equal(t, randomZone.Name(), zone.Name())

	// Getting a non-existing zone should return nil.
	zone, err = inventory.getZoneInView("_default", "non.existing.zone")
	require.NoError(t, err)
	require.Nil(t, zone)
}

// Benchmark measuring performance of loading the zones from disk to memory.
func BenchmarkZoneInventoryLoad(b *testing.B) {
	testCases := []int{10, 100, 1000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			// Setup server response.
			response := map[string]any{
				"views": map[string]any{
					"_default": map[string]any{
						"zones": generateRandomZones(testCase),
					},
				},
			}
			bind9StatsClient, off := setBenchmarkGetViewsResponseOK(b, response)
			defer off()

			// Create the inventory.
			sandbox := testutil.NewSandbox()
			defer sandbox.Close()
			storage, err := newZoneInventoryStorageMemoryDisk(sandbox.BasePath)
			if err != nil {
				b.Fatal(err)
			}
			inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)

			// Populate the zones to disk.
			done, err := inventory.populate(false)
			if err != nil {
				b.Fatal(err)
			}
			if inventory.getCurrentState().name == zoneInventoryStatePopulating {
				<-done
			}
			err = inventory.getCurrentState().err
			if err != nil {
				b.Fatal(err)
			}

			// Begin the actual benchmark.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				done, err := inventory.load(false)
				if err != nil {
					b.Fatal(err)
				}
				if inventory.getCurrentState().name == zoneInventoryStateLoading {
					<-done
				}
				err = inventory.getCurrentState().err
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark measuring performance of receiving the zones over the channel.
// Zones are stored in memory.
func BenchmarkZoneInventoryReceiveZonesMemory(b *testing.B) {
	testCases := []int{10, 100, 1000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			// Setup server response.
			response := map[string]any{
				"views": map[string]any{
					"_default": map[string]any{
						"zones": generateRandomZones(testCase),
					},
				},
			}
			bind9StatsClient, off := setBenchmarkGetViewsResponseOK(b, response)
			defer off()

			// Create the inventory.
			inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)

			// Populate the zones from the DNS server to the inventory.
			done, err := inventory.populate(false)
			if err != nil {
				b.Fatalf("error populating zone inventory %+v\n", err)
			}
			if inventory.getCurrentState().name == zoneInventoryStatePopulating {
				<-done
			}
			err = inventory.getCurrentState().err
			if err != nil {
				b.Fatalf("error populating zone inventory %+v\n", err)
			}

			// Begin the actual benchmark.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var receivedZones []*bind9stats.ExtendedZone
				channel, err := inventory.receiveZones(context.Background(), nil)
				if err != nil {
					b.Fatal(err)
				}
				if inventory.getCurrentState().name == zoneInventoryStateReceivingZones {
					for result := range channel {
						if result.err != nil {
							b.Fatal(err)
						}
						receivedZones = append(receivedZones, result.zone)
					}
				}
				if len(receivedZones) != testCase {
					b.Fatalf("not all zones have been transferred; only transferred %d zones", len(receivedZones))
				}
			}
		})
	}
}

// Benchmark measuring performance of receiving the zones over the channel.
// Zones are stored on disk.
func BenchmarkZoneInventoryReceiveZonesDisk(b *testing.B) {
	testCases := []int{10, 100, 1000, 10000, 100000}
	for _, testCase := range testCases {
		defaultZones := generateRandomZones(testCase)
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			// Setup server response.
			response := map[string]any{
				"views": map[string]any{
					"_default": map[string]any{
						"zones": defaultZones,
					},
				},
			}
			bind9StatsClient, off := setBenchmarkGetViewsResponseOK(b, response)
			defer off()

			// Create the inventory.
			sandbox := testutil.NewSandbox()
			defer sandbox.Close()
			storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
			if err != nil {
				b.Fatal(err)
			}
			inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)

			// Populate the zones from the DNS server to the inventory.
			done, err := inventory.populate(false)
			if err != nil {
				b.Fatalf("error populating zone inventory %+v\n", err)
			}
			if inventory.getCurrentState().name == zoneInventoryStatePopulating {
				<-done
			}
			err = inventory.getCurrentState().err
			if err != nil {
				b.Fatalf("error populating zone inventory %+v\n", err)
			}

			// Begin the actual benchmark.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var receivedZones []*bind9stats.ExtendedZone
				channel, err := inventory.receiveZones(context.Background(), nil)
				if err != nil {
					b.Fatal(err)
				}
				if inventory.getCurrentState().name == zoneInventoryStateReceivingZones {
					for result := range channel {
						if result.err != nil {
							b.Fatal(err)
						}
						receivedZones = append(receivedZones, result.zone)
					}
				}
				if len(receivedZones) != testCase {
					b.Fatalf("not all zones have been transferred; only transferred %d zones", len(receivedZones))
				}
			}
		})
	}
}

// Benchmark testing performance of getting a selected zone from memory.
func BenchmarkZoneInventoryGetZoneInView(b *testing.B) {
	// Setup server response.
	randomZones := generateRandomZones(100000)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": randomZones,
			},
		},
	}
	bind9StatsClient, off := setBenchmarkGetViewsResponseOK(b, response)
	defer off()

	// Create zone inventory in memory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), bind9StatsClient, "localhost", 5380)

	// Populate the zones from the DNS server to the inventory.
	done, err := inventory.populate(false)
	if err != nil {
		b.Fatal(err)
	}
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	if err != nil {
		b.Fatal(err)
	}

	// Begin the actual benchmark.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get a random zone from the existing ones.
		zone, err := inventory.getZoneInView("_default", randomZones[rand.Int64()%100000].Name())
		if err != nil {
			b.Fatal(err)
		}
		if zone == nil {
			b.Fatal("zone not found")
		}
	}
}

// Benchmark testing performance of getting a selected zone from disk.
func BenchmarkZoneInventoryGetZoneInViewDiskStorage(b *testing.B) {
	// Setup server response.
	randomZones := generateRandomZones(100000)
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": randomZones,
			},
		},
	}
	bind9StatsClient, off := setBenchmarkGetViewsResponseOK(b, response)
	defer off()

	// Create the inventory.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	storage, err := newZoneInventoryStorageDisk(sandbox.BasePath)
	if err != nil {
		b.Fatal(err)
	}
	inventory := newZoneInventory(storage, bind9StatsClient, "localhost", 5380)

	// Populate the zones into the inventory.
	done, err := inventory.populate(false)
	if err != nil {
		b.Fatal(err)
	}
	if inventory.getCurrentState().name == zoneInventoryStatePopulating {
		<-done
	}
	err = inventory.getCurrentState().err
	if err != nil {
		b.Fatal(err)
	}

	// Begin the actual benchmark.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get a random zone from the existing ones.
		zone, err := inventory.getZoneInView("_default", randomZones[rand.Int64()%100000].Name())
		if err != nil {
			b.Fatal(err)
		}
		if zone == nil {
			b.Fatal("zone not found")
		}
	}
}
