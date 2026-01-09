package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"isc.org/stork/daemondata/kea"
)

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
