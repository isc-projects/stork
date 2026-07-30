//nolint:goconst
package testutil

import (
	"time"

	"isc.org/stork/daemondata/bind9xfr"
)

// Generates test zone transfer states used in different unit tests.
func GetTestZoneTransfers() []*bind9xfr.State {
	// Cannot use storkutil.Ptr() as it would cause import cycle.
	serial1 := int64(2026041600)
	serial2 := int64(2026041601)
	serial3 := int64(2026041602)
	serial4 := int64(2026041603)
	serial5 := int64(2026041604)
	serial6 := int64(0)
	return []*bind9xfr.State{
		{
			ViewName:       "_default",
			ZoneName:       "good.example.org",
			Serial:         &serial1,
			Client:         "127.0.0.1",
			Server:         "192.5.5.241",
			MessagesCount:  79,
			RecordsCount:   24872,
			BytesCount:     1320233,
			Duration:       52 * time.Millisecond,
			Status:         bind9xfr.StatusCompleted,
			StartTime:      time.Date(2026, 4, 16, 10, 41, 27, 71000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 16, 10, 45, 11, 124000000, time.UTC),
		},
		{
			ViewName:       "_default",
			ZoneName:       "isc.example.org",
			Serial:         &serial2,
			Client:         "127.0.0.1",
			Server:         "192.5.5.241",
			MessagesCount:  179,
			RecordsCount:   24872,
			BytesCount:     1320233,
			Duration:       40 * time.Millisecond,
			Status:         bind9xfr.StatusMessage,
			StartTime:      time.Date(2026, 4, 16, 11, 42, 30, 71000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 16, 11, 44, 30, 124000000, time.UTC),
		},
		{
			ViewName:       "private",
			ZoneName:       "internal.example.org",
			Serial:         &serial3,
			Client:         "192.168.1.1",
			Server:         "192.168.1.2",
			MessagesCount:  1,
			RecordsCount:   1,
			BytesCount:     1,
			Duration:       1 * time.Second,
			Status:         bind9xfr.StatusMessage,
			StartTime:      time.Date(2026, 4, 16, 12, 32, 13, 50000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 16, 12, 33, 34, 71000000, time.UTC),
		},
		{
			ViewName:  "public",
			ZoneName:  "public.example.org",
			Serial:    &serial4,
			Client:    "192.168.1.1",
			Server:    "192.168.1.2",
			Status:    bind9xfr.StatusStarted,
			StartTime: time.Date(2026, 4, 23, 13, 12, 34, 50000000, time.UTC),
		},
		{
			ViewName:  "_default",
			ZoneName:  "bad.example.org",
			Serial:    &serial5,
			Client:    "192.168.1.1",
			Server:    "192.168.1.2",
			Status:    bind9xfr.StatusMessage,
			Message:   "Transfer failed: AXFR timed out after 50 seconds (serial 2026041604)",
			StartTime: time.Date(2026, 4, 25, 1, 2, 0, 13000000, time.UTC),
		},
		{
			ZoneName: "zero.example.org",
			Status:   bind9xfr.StatusMessage,
			Message:  "Transfer failed: AXFR timed out after 0 seconds (serial 0)",
		},
		{
			ZoneName:       ".",
			Status:         bind9xfr.StatusCompleted,
			Serial:         &serial6,
			Client:         "2001:db8::1",
			Server:         "2001:db8::2",
			StartTime:      time.Date(2026, 4, 26, 1, 2, 0, 13000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 26, 1, 2, 0, 13000000, time.UTC),
		},
	}
}

// Generates test local zone transfer states (i.e., zone transfers where
// the client and server are running on the same machine).
func GetTestLocalZoneTransfers() []*bind9xfr.State {
	return []*bind9xfr.State{
		{
			ZoneName: "local.example.org",
			Client:   "127.0.0.1",
			Server:   "127.0.0.1",
			Status:   bind9xfr.StatusMessage,
			Message:  "Transfer failed: AXFR timed out after 0 seconds (serial 0)",
		},
	}
}
