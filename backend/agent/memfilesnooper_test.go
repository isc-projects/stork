package agent

import (
	"bufio"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Write the lines from input to the file in output one at a time, returning control to the Go scheduler after each write..
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
func slowlyWriteToLeasefileWithSwap(input1 string, input2 string, output1 *os.File, output2 *os.File) error {
	output1Name := output1.Name()
	output2Name := output2.Name()
	err := slowlyWriteToLeasefile(input1, output1)
	if err != nil {
		return err
	}
	output2.Close()
	os.Remove(output2Name)
	os.Rename(output1Name, output2Name)
	output1Again, err := os.OpenFile(output1Name, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	err = slowlyWriteToLeasefile(input2, output1Again)
	return err
}

var (
	ErrInvalidLimit = errors.New("invalid limit parameter; it is not possible to read a negative number of items from the channel")
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
				didTimeOut = ErrTimedOut
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
	leasefile, err := os.CreateTemp("", "leases4-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	defer os.Remove(leasefile.Name())

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
	rowsource, err := NewRowSource(leasefile.Name())
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

func TestRowSourceFollowsAcrossFileSwap(t *testing.T) {
	// Arrange
	preCleanup, err := os.CreateTemp("", "leases4-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	preCleanupName := preCleanup.Name()
	postCleanup, err := os.CreateTemp("", "leases4-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}

	infile1 := "testdata/small-leases4.csv"
	infile2 := "testdata/small2-leases4.csv"
	expected1Addr := "192.110.111.2"
	expected2Addr := "192.110.111.5"
	expectedRows := 8
	rowsource, err := NewRowSource(preCleanupName)
	require.NoError(t, err)

	// Act
	go slowlyWriteToLeasefileWithSwap(infile1, infile2, preCleanup, postCleanup)
	results := rowsource.Start()

	// Assert
	parsedRows, err := readChanToLimitWithTimeout(
		results,
		expectedRows,
		t.Context(),
		250*time.Millisecond,
	)
	require.NoError(t, err, "Got error while reading channel")
	require.Len(t, parsedRows, expectedRows)
	require.EqualValues(t, expected1Addr, parsedRows[1][0])
	require.EqualValues(t, expected2Addr, parsedRows[5][0])
}

// Confirm that ParseRowAsLease4 handles the various kinds of rows as expected.
func TestParseRowAsLease4(t *testing.T) {
	// Arrange
	testCases := []struct {
		row         []string
		minCLTT     uint64
		expected    *Lease4
		expectedErr error
	}{
		// Headers, which it should skip.
		{
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
			ErrHeaders,
		},
		// Valid IPv4 data, which it should parse.
		{
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
			&Lease4{
				"192.110.111.2",
				"03:00:00:00:00:00",
				1761257849,
				1761254249,
				3600,
				123,
				0,
			},
			nil,
		},
		// Valid IPv6 data, which it should refuse to parse.
		{
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
			ErrUnexpectedV6,
		},
		// Valid IPv4 data but with CLTT too old, which it should refuse to parse.
		{
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
			ErrCLTTTooOld,
		},
	}
	for _, tc := range testCases {
		// Act
		actual, err := ParseRowAsLease4(tc.row, tc.minCLTT)
		// Assert
		if tc.expectedErr != nil {
			require.ErrorIs(t, err, tc.expectedErr)
		}
		require.Equal(t, tc.expected, actual)
	}
}

func TestParseRowAsLease6(t *testing.T) {
	// Arrange
	testCases := []struct {
		row         []string
		minCLTT     uint64
		expected    *Lease6
		expectedErr error
	}{
		// Headers, which it should skip.
		{
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
			ErrHeaders,
		},
		// Valid IPv4 data, which it should refuse to parse.
		{
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
			ErrUnexpectedV4,
		},
		// Valid IPv6 data, which it should parse.
		{
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
			&Lease6{
				"51a4:14ec:1::",
				"01:00:00:00:00:00",
				1761672649,
				1761669049,
				3600,
				123,
				2,
				128,
			},
			nil,
		},
		// Valid IPv6 data but with CLTT too old, which it should refuse to parse.
		{
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
			ErrCLTTTooOld,
		},
	}
	for _, tc := range testCases {
		// Act
		actual, err := ParseRowAsLease6(tc.row, tc.minCLTT)
		// Assert
		if tc.expectedErr != nil {
			require.ErrorIs(t, err, tc.expectedErr)
		}
		require.Equal(t, tc.expected, actual)
	}
}
