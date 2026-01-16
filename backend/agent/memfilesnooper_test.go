package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
)

//go:generate mockgen -source memfilesnooper.go -package=agent -destination=memfilesnoopermock_test.go -mock_names=RowSource=MockRowSource,MemfileSnooper=MockMemfileSnooper isc.org/agent RowSource,MemfileSnooper

// Write the lines from input to the file in output one at a time, syncing the
// file to encourage the changes to reach the disk and trigger a filesystem
// event.
func slowlyWriteToLeasefile(input string, output *os.File) error {
	defer output.Close()
	infile, err := os.Open(input)
	if err != nil {
		return err
	}
	defer infile.Close()
	lf := []byte{'\n'}
	scanner := bufio.NewScanner(infile)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		line := string(bytes)
		output.Write(bytes)
		output.Write(lf)
		output.Sync()
		log.WithField("line", line).Trace("Wrote line")
	}
	return nil
}

// Write the lines from input1 to the file in output1 one at a time, with a
// delay of 300 milliseconds between each write. Then rename output1 to output2,
// reopen output1, and write the lines from input2 to output1 (as before).
func slowlyWriteToLeasefileWithSwapAndDelay(input1 string, input2 string, output1 *os.File, output2 *os.File, delay time.Duration) error {
	output1Name := output1.Name()
	output2Name := output2.Name()
	err := slowlyWriteToLeasefile(input1, output1)
	if err != nil {
		return err
	}
	output2.Close()
	os.Remove(output2Name)
	os.Rename(output1Name, output2Name)
	if delay != 0 {
		time.Sleep(delay)
	}
	output1Again, err := os.OpenFile(output1Name, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	err = slowlyWriteToLeasefile(input2, output1Again)
	return err
}

var (
	ErrInvalidLimit = errors.New("invalid limit parameter; it is not possible to read a negative number of items from the channel")
	ErrChanClosed   = errors.New("channel closed unexpectedly")
	ErrTimedOut     = errors.New("timed out while waiting for enough rows")
)

// Read up to `limit` rows from `c`, stopping after the provided timeout.  If
// the timeout expires before reading `limit` items, return immediately and signal
// an error.
func readChanToLimitWithTimeout(c chan []string, limit int, ctx context.Context, timeout time.Duration) ([][]string, error) {
	timeoutCtx, cancelFn := context.WithTimeout(ctx, timeout)
	defer cancelFn()
	if limit < 0 {
		return nil, ErrInvalidLimit
	}
	results := make([][]string, 0, limit)
	var didTimeOut error = nil
	for didTimeOut == nil && len(results) < limit {
		select {
		case row, ok := <-c:
			if !ok {
				didTimeOut = ErrChanClosed
				break
			}
			results = append(results, row)
		case <-timeoutCtx.Done():
			didTimeOut = ErrTimedOut
		}
	}
	return results, didTimeOut
}

// Confirm that RowSource reads the expected amount of data from a file in the
// simple case where the data is already there and no following is necessary.
func TestRowSourceExistingFile(t *testing.T) {
	// Arrange
	infile := "testdata/small-leases4.csv"
	expected0 := []string{
		"address",
		"hwaddr",
		"client_id",
		"valid_lifetime",
		"expire",
		"subnet_id",
		"fqdn_fwd",
		"fqdn_rev",
		"hostname",
		"state",
		"user_context",
		"pool_id",
	}
	expected1 := []string{
		"192.110.111.2",
		"03:00:00:00:00:00",
		"01:03:00:00:00:00:00",
		"3600",
		"1761257849",
		"123",
		"0",
		"0",
		"",
		"0",
		"",
		"0",
	}
	rowsource, err := NewRowSource(infile)
	require.NoError(t, err)

	// Act
	channel := rowsource.Start()
	defer rowsource.Stop()
	actual, err := readChanToLimitWithTimeout(channel, 4, t.Context(), 250*time.Millisecond)
	// Assert

	require.NoError(t, err, "Got error while reading channel")
	require.Equal(t, expected0, actual[0])
	require.Equal(t, expected1, actual[1])
}

// Confirm that RowSource continues reading rows from a file that is actively
// being written to.
func TestRowSourceContinuesReadingOverTime(t *testing.T) {
	// Arrange
	tmpdir := t.TempDir()
	err := os.MkdirAll(tmpdir, 0o750)
	require.NoError(t, err, "unable to create temporary directory for test execution")

	leasefileName := filepath.Join(tmpdir, "kea-leases4.csv")
	leasefile, err := os.Create(leasefileName)
	require.NoError(t, err, "unable to create temporary leases file")
	defer os.Remove(leasefileName)

	infile := "testdata/small-leases4.csv"
	expected0 := []string{
		"address",
		"hwaddr",
		"client_id",
		"valid_lifetime",
		"expire",
		"subnet_id",
		"fqdn_fwd",
		"fqdn_rev",
		"hostname",
		"state",
		"user_context",
		"pool_id",
	}
	expected1 := []string{
		"192.110.111.2",
		"03:00:00:00:00:00",
		"01:03:00:00:00:00:00",
		"3600",
		"1761257849",
		"123",
		"0",
		"0",
		"",
		"0",
		"",
		"0",
	}
	rowsource, err := NewRowSource(leasefileName)
	require.NoError(t, err)

	// Act
	go slowlyWriteToLeasefile(infile, leasefile)
	channel := rowsource.Start()
	defer rowsource.Stop()
	actual, err := readChanToLimitWithTimeout(channel, 4, t.Context(), 250*time.Millisecond)
	// Assert

	require.NoError(t, err, "Got an error when trying to read rows")
	require.Equal(t, expected0, actual[0])
	require.Equal(t, expected1, actual[1])
}

// Confirm that RowSource continues producing rows after the file is swapped
// (a la kea-lfc). Additionally, confirm that RowSource will wait for the Create
// event (and not just error out) if it takes some time for kea-lfc to create
// the new leasefile.
func TestRowSourceFollowsAcrossFileSwap(t *testing.T) {
	delaySettings := []time.Duration{0, 50 * time.Millisecond}

	for _, delay := range delaySettings {
		t.Run(fmt.Sprintf("Delay=%d", delay), func(t *testing.T) {
			// Arrange
			log.SetLevel(log.TraceLevel)
			log.Trace("Start of test run")
			tmpdir := t.TempDir()
			err := os.MkdirAll(tmpdir, 0o750)
			require.NoError(t, err, "unable to create temporary directory for test execution")

			preCleanupName := filepath.Join(tmpdir, "kea-leases4.csv")
			preCleanup, err := os.Create(preCleanupName)
			require.NoError(t, err, "unable to create temporary leases file (pre-cleanup)")

			postCleanupName := filepath.Join(tmpdir, "kea-leases4.csv.1")
			postCleanup, err := os.Create(postCleanupName)
			require.NoError(t, err, "unable to create temporary leases file (post-cleanup)")

			infile1 := "testdata/small-leases4.csv"
			infile2 := "testdata/small2-leases4.csv"
			expected1Addr := "192.110.111.2"
			expected2Addr := "192.110.111.5"
			expectedRows := 8
			rowsource, err := NewRowSource(preCleanupName)
			require.NoError(t, err)

			// Act
			go slowlyWriteToLeasefileWithSwapAndDelay(
				infile1,
				infile2,
				preCleanup,
				postCleanup,
				delay,
			)
			results := rowsource.Start()
			defer rowsource.Stop()

			// Assert
			parsedRows, err := readChanToLimitWithTimeout(
				results,
				expectedRows,
				t.Context(),
				250*time.Millisecond+delay,
			)
			require.NoError(t, err, "Got error while reading channel")
			require.Len(t, parsedRows, expectedRows)
			fileBytes, err := os.ReadFile(preCleanupName)
			require.NoError(t, err)
			fileContents := string(fileBytes)
			require.EqualValues(t, expected1Addr, parsedRows[1][0], "complete parse results: %v; file contents: '%s'", parsedRows, fileContents)
			require.EqualValues(t, expected2Addr, parsedRows[5][0], "complete parse results: %v; file contents: '%s'", parsedRows, fileContents)
		})
	}
}

// Confirm that RowSource conitnues producing rows from a file when EnsureWatching is called with the same file path.
func TestRowSourceEnsureWatchingNoChange(t *testing.T) {
	// Arrange
	tmpdir := t.TempDir()
	err := os.MkdirAll(tmpdir, 0o750)
	require.NoError(t, err, "unable to create temporary directory for test execution")

	leasefileName := filepath.Join(tmpdir, "kea-leases4.csv")
	leasefile, err := os.Create(leasefileName)
	require.NoError(t, err, "unable to create temporary leases file")
	defer os.Remove(leasefileName)

	infile := "testdata/small-leases4.csv"
	infile2 := "testdata/small2-leases4.csv"
	expected0 := []string{
		"address",
		"hwaddr",
		"client_id",
		"valid_lifetime",
		"expire",
		"subnet_id",
		"fqdn_fwd",
		"fqdn_rev",
		"hostname",
		"state",
		"user_context",
		"pool_id",
	}
	expected1 := []string{
		"192.110.111.2",
		"03:00:00:00:00:00",
		"01:03:00:00:00:00:00",
		"3600",
		"1761257849",
		"123",
		"0",
		"0",
		"",
		"0",
		"",
		"0",
	}
	expected5 := []string{
		"192.110.111.5",
		"03:00:00:00:00:03",
		"01:03:00:00:00:00:03",
		"3600",
		"1761257853",
		"123",
		"0",
		"0",
		"",
		"0",
		"",
		"0",
	}
	rowsource, err := NewRowSource(leasefileName)
	require.NoError(t, err)
	batch1, err := os.ReadFile(infile)
	require.NoError(t, err)
	_, err = leasefile.Write(batch1)
	require.NoError(t, err)

	// Act
	channel := rowsource.Start()
	defer rowsource.Stop()
	actual, errRead1 := readChanToLimitWithTimeout(channel, 4, t.Context(), 250*time.Millisecond)
	errEnsure := rowsource.EnsureWatching(leasefileName)
	go func() {
		err = slowlyWriteToLeasefile(infile2, leasefile)
		require.NoError(t, err, "Got an error when trying to WRITE the second set of rows")
	}()

	actualAfter, errRead2 := readChanToLimitWithTimeout(channel, 4, t.Context(), 250*time.Millisecond)

	// Assert
	require.NoError(t, errRead1, "Got an error when trying to read the first set of rows")
	require.NoError(t, errEnsure, "Got an error when trying to EnsureWatching")
	require.NoError(t, errRead2, "Got an error when trying to read the second set of rows")
	require.Equal(t, expected0, actual[0])
	require.Equal(t, expected1, actual[1])
	require.Equal(t, expected0, actualAfter[0])
	require.Equal(t, expected5, actualAfter[1])
}

// Confirm that RowSource produces rows from a new file if EnsureWatching is
// called with a new path.
func TestRowSourceEnsureWatchingWithChange(t *testing.T) {
	// Arrange
	infile := "testdata/small-leases4.csv"
	infile2 := "testdata/small2-leases4.csv"
	expected0 := []string{
		"address",
		"hwaddr",
		"client_id",
		"valid_lifetime",
		"expire",
		"subnet_id",
		"fqdn_fwd",
		"fqdn_rev",
		"hostname",
		"state",
		"user_context",
		"pool_id",
	}
	expected1 := []string{
		"192.110.111.2",
		"03:00:00:00:00:00",
		"01:03:00:00:00:00:00",
		"3600",
		"1761257849",
		"123",
		"0",
		"0",
		"",
		"0",
		"",
		"0",
	}
	expected5 := []string{
		"192.110.111.5",
		"03:00:00:00:00:03",
		"01:03:00:00:00:00:03",
		"3600",
		"1761257853",
		"123",
		"0",
		"0",
		"",
		"0",
		"",
		"0",
	}
	rowsource, err := NewRowSource(infile)
	require.NoError(t, err)

	// Act
	channel := rowsource.Start()
	defer rowsource.Stop()
	actual, errRead1 := readChanToLimitWithTimeout(channel, 4, t.Context(), 250*time.Millisecond)

	errEnsure := rowsource.EnsureWatching(infile2)

	actualAfter, errRead2 := readChanToLimitWithTimeout(channel, 4, t.Context(), 250*time.Millisecond)

	// Assert
	require.NoError(t, errRead1, "Got an error when trying to read the first set of rows")
	require.NoError(t, errEnsure, "Got an error when trying to EnsureWatching the new file")
	require.NoError(t, errRead2, "Got an error when trying to read the second set of rows")
	require.Equal(t, expected0, actual[0])
	require.Equal(t, expected1, actual[1])
	require.Equal(t, expected0, actualAfter[0])
	require.Equal(t, expected5, actualAfter[1])
}

func TestRowSourcePreconditions(t *testing.T) {
	t.Run("Calling .Stop() on a RowSource that isn't running should not error out", func(t *testing.T) {
		rowsource, err := NewRowSource("testdata/small-leases4.csv")
		require.NoError(t, err)

		rowsource.Stop()
	})

	t.Run("Calling .Start() on a RowSource that is already running should not start a second goroutine", func(t *testing.T) {
		rowsource, err := NewRowSource("testdata/small-leases4.csv")
		require.NoError(t, err)

		_ = rowsource.Start()
		grCount := runtime.NumGoroutine()
		_ = rowsource.Start()
		grCount2 := runtime.NumGoroutine()
		require.Equal(t, grCount, grCount2, "the number of goroutines running changed after calling .Start() a second time")
	})
}

// Confirm that ParseRowAsLease4 handles the various kinds of rows as expected.
func TestParseRowAsLease4(t *testing.T) {
	// Arrange
	testCases := []struct {
		desc        string
		row         []string
		minCLTT     uint64
		expected    *keadata.Lease
		expectedErr string // Zero value means no expected error.
	}{
		{
			"Empty slice, which it should skip",
			[]string{},
			0,
			nil,
			"empty",
		},
		{
			"Headers, which it should skip.",
			[]string{
				"address",
				"hwaddr",
				"client_id",
				"valid_lifetime",
				"expire",
				"subnet_id",
				"fqdn_fwd",
				"fqdn_rev",
				"hostname",
				"state",
				"user_context",
				"pool_id",
			},
			0,
			nil,
			"headers",
		},
		{
			"Valid IPv4 data, which it should parse.",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"3600",
				"1761257849",
				"123",
				"0",
				"0",
				"",
				"0",
				"",
				"0",
			},
			0,
			&keadata.Lease{
				IPVersion:     keadata.LeaseIPv4,
				IPAddress:     "192.110.111.2",
				HWAddress:     "03:00:00:00:00:00",
				CLTT:          1761254249,
				ValidLifetime: 3600,
				SubnetID:      123,
				State:         0,
			},
			"",
		},
		{
			"Valid IPv6 data, which it should refuse to parse.",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"1761672649",
				"123",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			0,
			nil,
			"IPv6",
		},
		{
			"Valid IPv4 data but with CLTT too old, which it should refuse to parse.",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"3600",
				"1761257849",
				"123",
				"0",
				"0",
				"",
				"0",
				"",
				"0",
			},
			1761254250,
			nil,
			"",
		},
		{
			"Non-integer expiry timestamp, which it should refuse to parse.",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"3600",
				"CA55E11E",
				"123",
				"0",
				"0",
				"",
				"0",
				"",
				"0",
			},
			0,
			nil,
			"expiry",
		},
		{
			"Non-integer valid lifetime, which it should refuse to parse.",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"CA55E11E",
				"1761257849",
				"123",
				"0",
				"0",
				"",
				"0",
				"",
				"0",
			},
			0,
			nil,
			"valid_lifetime",
		},
		{
			"Non-integer subnet ID, which it should refuse to parse.",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"3600",
				"1761257849",
				"CA55E11E",
				"0",
				"0",
				"",
				"0",
				"",
				"0",
			},
			0,
			nil,
			"subnet ID",
		},
		{
			"Non-integer lease state, which it should refuse to parse.",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"3600",
				"1761257849",
				"123",
				"0",
				"0",
				"",
				"CA55E11E",
				"",
				"0",
			},
			0,
			nil,
			"lease state",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Act
			actual, err := ParseRowAsLease4(tc.row, tc.minCLTT)
			// Assert
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			}
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestParseRowAsLease6(t *testing.T) {
	// Arrange
	testCases := []struct {
		desc        string
		row         []string
		minCLTT     uint64
		expected    *keadata.Lease
		expectedErr string // Zero value means no expected error.
	}{
		{
			"Empty slice, which it should skip",
			[]string{},
			0,
			nil,
			"empty",
		},
		{
			"Headers, which it should skip",
			[]string{
				"address",
				"hwaddr",
				"client_id",
				"valid_lifetime",
				"expire",
				"subnet_id",
				"fqdn_fwd",
				"fqdn_rev",
				"hostname",
				"state",
				"user_context",
				"pool_id",
			},
			0,
			nil,
			"headers",
		},
		{
			"Valid IPv4 data, which it should refuse to parse",
			[]string{
				"192.110.111.2",
				"03:00:00:00:00:00",
				"01:03:00:00:00:00:00",
				"3600",
				"1761257849",
				"123",
				"0",
				"0",
				"",
				"0",
				"",
				"0",
			},
			0,
			nil,
			"IPv4",
		},
		{
			"Valid IPv6 data, which it should parse",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"1761672649",
				"123",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			0,
			&keadata.Lease{
				IPVersion:     keadata.LeaseIPv6,
				IPAddress:     "51a4:14ec:1::",
				DUID:          "01:00:00:00:00:00",
				CLTT:          1761669049,
				ValidLifetime: 3600,
				SubnetID:      123,
				State:         2,
				PrefixLength:  128,
			},
			"",
		},
		{
			"Valid IPv6 data but with CLTT too old, which it should refuse to parse",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"1761672649",
				"123",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			1761669050,
			nil,
			"",
		},
		{
			"Non-integer expiry timestamp, which it should refuse to parse.",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"CA55E11E",
				"123",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			0,
			nil,
			"expiry",
		},
		{
			"Non-integer valid lifetime, which it should refuse to parse.",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"CA55E11E",
				"1761672649",
				"123",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			0,
			nil,
			"valid_lifetime",
		},
		{
			"Non-integer subnet ID, which it should refuse to parse.",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"1761672649",
				"CA55E11E",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			0,
			nil,
			"subnet ID",
		},
		{
			"Non-integer lease state, which it should refuse to parse.",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"1761672649",
				"123",
				"2250",
				"0",
				"1",
				"128",
				"0",
				"0",
				"",
				"",
				"CA55E11E",
				"",
				"",
				"",
				"0",
			},
			0,
			nil,
			"lease state",
		},
		{
			"Non-integer prefix length, which it should refuse to parse.",
			[]string{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				"3600",
				"1761672649",
				"123",
				"2250",
				"0",
				"1",
				"CA55E11E",
				"0",
				"0",
				"",
				"",
				"2",
				"",
				"",
				"",
				"0",
			},
			0,
			nil,
			"prefix length",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Act
			actual, err := ParseRowAsLease6(tc.row, tc.minCLTT)
			// Assert
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			}
			require.Equal(t, tc.expected, actual)
		})
	}
}

// Create a mock function which emits the given list of rows, one at a time,
// into the channel.  It signals the wait group when complete.
func mockEmitRows(rows [][]string, wg *sync.WaitGroup) func() chan []string {
	return func() chan []string {
		channel := make(chan []string)
		go func() {
			for _, row := range rows {
				channel <- row
			}
			// This is a load-bearing sleep.  Without it, this goroutine exits promptly and
			// control is almost always transferred back to the main test, rather than to
			// the MemfileSnooper's goroutine.  If the MemfileSnooper doesn't run again
			// before the GetSnapshot call, the fifth lease will not be appended before
			// GetSnapshot takes the lock.
			time.Sleep(time.Millisecond)
			wg.Done()
		}()
		return channel
	}
}

// Create a mock RowSource which will emit the list of rows, one at a time, into
// the channel when started.  It will signal the wait group when this is complete.
func makeMockRowSource(ctrl *gomock.Controller, rows [][]string) (RowSource, *sync.WaitGroup) {
	rowSource := NewMockRowSource(ctrl)
	wg := sync.WaitGroup{}
	wg.Add(1)
	rowSource.EXPECT().Start().DoAndReturn(mockEmitRows(rows, &wg))
	rowSource.EXPECT().Stop()
	return rowSource, &wg
}

// Ensure that the MemfileSnooper collects all the leases from the RowSource.
func TestMemfileSnooperCollectsLeases(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rows := [][]string{
		{"192.168.1.1", "00:00:00:00:00:00", "01:03:00:00:00:00:00", "3600", "1761257849", "123", "0", "0", "", "0", "", "0"},
		{"192.168.1.2", "00:00:00:00:00:01", "01:03:00:00:00:00:01", "3600", "1761257850", "123", "0", "0", "", "0", "", "0"},
		{"192.168.1.3", "00:00:00:00:00:02", "01:03:00:00:00:00:02", "3600", "1761257851", "123", "0", "0", "", "0", "", "0"},
		{"192.168.1.4", "00:00:00:00:00:03", "01:03:00:00:00:00:03", "3600", "1761257852", "123", "0", "0", "", "0", "", "0"},
		{"192.168.1.5", "00:00:00:00:00:04", "01:03:00:00:00:00:04", "3600", "1761257853", "123", "0", "0", "", "0", "", "0"},
	}
	rowSource, wg := makeMockRowSource(ctrl, rows)

	memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv4, rowSource)
	require.NoError(t, err)

	// Act
	memfileSnooper.Start()
	wg.Wait()
	memfileSnooper.Stop()

	// Assert
	snapshot := memfileSnooper.GetSnapshot()
	require.Len(t, snapshot, 5)
}

// Ensure that the MemfileSnooper responds appropriately in various error
// conditions, documented within.
func TestMemfileSnooperErrorConditions(t *testing.T) {
	// The snooper should skip a header row in the input without complaining (because that's normal and will happen every time it starts fresh).
	t.Run("header row in input", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rows := [][]string{
			{"address", "hwaddr", "client_id", "valid_lifetime", "expire", "subnet_id", "fqdn_fwd", "fqdn_rev", "hostname", "state", "user_context", "pool_id"},
		}
		rowSource, wg := makeMockRowSource(ctrl, rows)

		memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv4, rowSource)
		require.NoError(t, err)

		// Act
		memfileSnooper.Start()
		wg.Wait()
		memfileSnooper.Stop()

		// Assert
		snapshot := memfileSnooper.GetSnapshot()
		require.Len(t, snapshot, 0)
	})
	// The snooper should skip lease updates with CLTTs older than the most recent
	// one it has seen.
	t.Run("old CLTT", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rows := [][]string{
			{"192.168.1.5", "00:00:00:00:00:04", "01:03:00:00:00:00:04", "3600", "1761257853", "123", "0", "0", "", "0", "", "0"},
			{"192.168.1.4", "00:00:00:00:00:03", "01:03:00:00:00:00:03", "3600", "1761257852", "123", "0", "0", "", "0", "", "0"},
		}
		rowSource, wg := makeMockRowSource(ctrl, rows)

		memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv4, rowSource)
		require.NoError(t, err)

		// Act
		memfileSnooper.Start()
		wg.Wait()
		memfileSnooper.Stop()

		// Assert
		snapshot := memfileSnooper.GetSnapshot()
		require.Len(t, snapshot, 1)
	})
	// The snooper should clean itself up appropriately when the input channel from
	// the RowSource is closed.
	t.Run("channel closed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		wg := sync.WaitGroup{}
		wg.Add(1)
		rowSource := NewMockRowSource(ctrl)
		rowSource.EXPECT().Start().DoAndReturn(func() chan []string {
			c := make(chan []string)
			go func() {
				time.Sleep(time.Millisecond)
				close(c)
				wg.Done()
			}()
			return c
		})
		rowSource.EXPECT().Stop()
		memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv6, rowSource)
		require.NoError(t, err)

		// Act
		memfileSnooper.Start()
		wg.Wait()
		memfileSnooper.Stop()

		// Assert
		snapshot := memfileSnooper.GetSnapshot()
		require.Len(t, snapshot, 0)
	})
	// The snooper should return an error if it is asked to snoop for anything other
	// than kea-dhcp4 or kea-dhcp6.
	t.Run("invalid daemon name", func(t *testing.T) {
		ms1, err := NewMemfileSnooper(10, daemonname.CA, nil)
		require.Nil(t, ms1)
		require.ErrorContains(t, err, "daemons other than DHCPv4 and DHCPv6")

		ms2 := RealMemfileSnooper{
			kind:         daemonname.D2,
			rs:           nil,
			leaseUpdates: make([]*keadata.Lease, 0),
			stop:         make(chan bool, 1),
		}
		snapshot := ms2.GetSnapshot()
		require.Empty(t, snapshot)
	})
	// The snooper should not error or panic if .Stop() is called when it is already
	// stopped.
	t.Run(".Stop() on stopped snooper", func(t *testing.T) {
		snooper, err := NewMemfileSnooper(10, daemonname.DHCPv4, nil)
		require.NoError(t, err)

		snooper.Stop()
	})
	// The snooper should not start a second goroutine if .Start() is called more
	// than once.
	t.Run(".Start() on running snooper", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rows := [][]string{}
		rowSource, _ := makeMockRowSource(ctrl, rows)

		memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv4, rowSource)
		require.NoError(t, err)

		memfileSnooper.Start()
		defer memfileSnooper.Stop()
		grCount := runtime.NumGoroutine()
		memfileSnooper.Start()
		grCount2 := runtime.NumGoroutine()
		require.Equal(t, grCount, grCount2, "the number of goroutines running changed after calling .Start() a second time")
	})
	t.Run("needs lease compacting to fit in limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		expected0 := uint64(1761257858)
		expected1 := uint64(1761257859)
		rows := [][]string{
			{"192.168.1.4", "00:00:00:00:00:03", "01:03:00:00:00:00:03", "3600", "1761257852", "123", "0", "0", "", "0", "", "0"},
			{"192.168.1.4", "00:00:00:00:00:03", "01:03:00:00:00:00:03", "3600", strconv.FormatUint(expected0, 10), "123", "0", "0", "", "2", "", "0"},
			{"192.168.1.5", "00:00:00:00:00:04", "01:03:00:00:00:00:04", "3600", strconv.FormatUint(expected1, 10), "123", "0", "0", "", "0", "", "0"},
		}
		rowSource, wg := makeMockRowSource(ctrl, rows)

		memfileSnooper, err := NewMemfileSnooper(2, daemonname.DHCPv4, rowSource)
		require.NoError(t, err)

		// Act
		memfileSnooper.Start()
		wg.Wait()
		memfileSnooper.Stop()

		// Assert
		snapshot := memfileSnooper.GetSnapshot()
		require.Len(t, snapshot, 2)
		require.Equal(t, expected0-3600, snapshot[0].CLTT)
		require.Equal(t, expected1-3600, snapshot[1].CLTT)
	})
	t.Run("cannot fit in limit, doesn't update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		expected0 := uint64(1761257851)
		expected1 := uint64(1761257852)
		rows := [][]string{
			{"192.168.1.3", "00:00:00:00:00:02", "01:03:00:00:00:00:02", "3600", strconv.FormatUint(expected0, 10), "123", "0", "0", "", "0", "", "0"},
			{"192.168.1.4", "00:00:00:00:00:03", "01:03:00:00:00:00:03", "3600", strconv.FormatUint(expected1, 10), "123", "0", "0", "", "0", "", "0"},
			{"192.168.1.5", "00:00:00:00:00:04", "01:03:00:00:00:00:04", "3600", "1761257853", "123", "0", "0", "", "0", "", "0"},
		}
		rowSource, wg := makeMockRowSource(ctrl, rows)

		memfileSnooper, err := NewMemfileSnooper(2, daemonname.DHCPv4, rowSource)
		require.NoError(t, err)

		// Act
		memfileSnooper.Start()
		wg.Wait()
		memfileSnooper.Stop()

		// Assert
		snapshot := memfileSnooper.GetSnapshot()
		require.Len(t, snapshot, 2)
		require.Equal(t, expected0-3600, snapshot[0].CLTT)
		require.Equal(t, expected1-3600, snapshot[1].CLTT)
	})
}

// Ensure that GetSnapshot correctly de-duplicates lease updates by CLTT,
// discarding older updates in favor of newer ones.
func TestMemfileSnooperGetSnapshotDeduplicates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rows := [][]string{
		{
			"51a4:14ec:1::",
			"01:00:00:00:00:00",
			"3600",
			"1761672649",
			"123",
			"2250",
			"0",
			"1",
			"128",
			"0",
			"0",
			"",
			"",
			"2",
			"",
			"",
			"",
			"0",
		},
		{
			"51a4:14ec:1::",
			"01:00:00:00:00:00",
			"3600",
			"1761672650",
			"123",
			"2250",
			"0",
			"1",
			"128",
			"0",
			"0",
			"",
			"",
			"0",
			"",
			"",
			"",
			"0",
		},
		{
			"51a4:14ec:1::",
			"01:00:00:00:00:00",
			"3600",
			"1761672648",
			"123",
			"2250",
			"0",
			"1",
			"128",
			"0",
			"0",
			"",
			"",
			"1",
			"",
			"",
			"",
			"0",
		},
	}
	rowSource, wg := makeMockRowSource(ctrl, rows)

	memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv6, rowSource)
	require.NoError(t, err)

	// Act
	memfileSnooper.Start()
	wg.Wait()
	memfileSnooper.Stop()

	// Assert
	snapshot := memfileSnooper.GetSnapshot()
	require.Len(t, snapshot, 1)
	require.Equal(t, 0, snapshot[0].State)
}

// Ensure that EnsureWatching calls EnsureWatching on the underlying RowSource.
func TestMemfileSnooperEnsureWatchingCallsRowSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rowSource := NewMockRowSource(ctrl)
	rowSource.EXPECT().EnsureWatching("foo").Times(1)
	memfileSnooper, err := NewMemfileSnooper(10, daemonname.DHCPv6, rowSource)
	require.NoError(t, err)

	// Act
	memfileSnooper.EnsureWatching("foo")

	// Assert (ctrl.Finish does the work)
}
