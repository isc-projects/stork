package testutil

import (
	"time"

	"isc.org/stork/daemondata/bind9xfr"
)

// Generates test zone transfer states used in different unit tests.
func GetTestZoneTransfers() []*bind9xfr.State {
	return []*bind9xfr.State{
		{
			ViewName:       "_default",
			ZoneName:       "good.example.org",
			Serial:         2026041600,
			Client:         "127.0.0.1",
			Server:         "192.5.5.241",
			MessagesCount:  79,
			RecordsCount:   24872,
			BytesCount:     1320233,
			Duration:       52 * time.Millisecond,
			Status:         bind9xfr.StatusCompleted,
			StartTime:      time.Date(2026, 4, 16, 10, 41, 27, 71000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 16, 10, 41, 27, 124000000, time.UTC),
		},
		{
			ViewName:       "_default",
			ZoneName:       "isc.example.org",
			Serial:         2026041601,
			Client:         "127.0.0.1",
			Server:         "192.5.5.241",
			MessagesCount:  179,
			RecordsCount:   24872,
			BytesCount:     1320233,
			Duration:       40 * time.Millisecond,
			Status:         bind9xfr.StatusMessage,
			StartTime:      time.Date(2026, 4, 16, 10, 41, 27, 71000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 16, 10, 41, 27, 124000000, time.UTC),
		},
		{
			ViewName:       "private",
			ZoneName:       "internal.example.org",
			Serial:         2026041602,
			Client:         "192.168.1.1",
			Server:         "192.168.1.2",
			MessagesCount:  1,
			RecordsCount:   1,
			BytesCount:     1,
			Duration:       1 * time.Second,
			Status:         bind9xfr.StatusMessage,
			StartTime:      time.Date(2026, 4, 16, 10, 41, 27, 71000000, time.UTC),
			CompletionTime: time.Date(2026, 4, 16, 10, 41, 27, 124000000, time.UTC),
		},
		{
			ViewName:  "public",
			ZoneName:  "public.example.org",
			Serial:    2026041603,
			Client:    "192.168.1.1",
			Server:    "192.168.1.2",
			Status:    bind9xfr.StatusStarted,
			StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000000, time.UTC),
		},
		{
			ViewName:  "_default",
			ZoneName:  "bad.example.org",
			Serial:    2026041604,
			Client:    "192.168.1.1",
			Server:    "192.168.1.2",
			Status:    bind9xfr.StatusMessage,
			Message:   "Transfer failed: AXFR timed out after 50 seconds (serial 2026041604)",
			StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000000, time.UTC),
		},
		{
			ViewName: "_default",
			ZoneName: "zero.example.org",
			Client:   "192.168.1.1",
			Status:   bind9xfr.StatusMessage,
			Message:  "Transfer failed: AXFR timed out after 0 seconds (serial 0)",
		},
	}
}
