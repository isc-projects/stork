package agent

import (
	"bufio"
	"context"
	"io"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestChangeDetectorRegistration(t *testing.T) {
	// Arrange
	callCounter1 := 0
	c1Mut := sync.Mutex{}
	callCounter2 := 0
	c2Mut := sync.Mutex{}
	count1 := func(s FSChangeState) {
		log.WithField("state", s).Info("Called count1")
		c1Mut.Lock()
		defer c1Mut.Unlock()
		callCounter1 += 1
	}
	count2 := func(s FSChangeState) {
		log.WithField("state", s).Info("Called count2")
		c2Mut.Lock()
		defer c2Mut.Unlock()
		callCounter2 += 1
	}
	read1 := func() int {
		c1Mut.Lock()
		defer c1Mut.Unlock()
		return callCounter1
	}
	read2 := func() int {
		c2Mut.Lock()
		defer c2Mut.Unlock()
		return callCounter2
	}
	tmpfile, err := os.CreateTemp("", "changeme-")
	require.Nil(t, err)
	defer tmpfile.Close()
	cd, err := NewFsNotifyChangeDetector(tmpfile.Name())
	require.Nil(t, err)

	// Act
	cd.RegisterListener(count1)
	cd.Start()
	defer cd.Stop()
	_, err = tmpfile.WriteString("Spotted Hyena: *Crocuta crocuta*\n")
	require.Nil(t, err)
	tmpfile.Sync() // Ensure write is flushed to disk and fsnotify sees it.
	c1SnapshotAfterRegisterWrite := read1()
	c2SnapshotAfterRegisterWrite := read2()

	cd.RegisterListener(count2)
	_, err = tmpfile.WriteString("Painted Wolf: *Lycaon pictus*\n")
	require.Nil(t, err)
	tmpfile.Sync()
	c1SnapshotAfterRegisterWriteRegisterWrite := read1()
	c2SnapshotAfterRegisterWriteRegisterWrite := read2()

	cd.UnregisterAllListeners()
	_, err = tmpfile.WriteString("Striped Hyena: *Hyaena hyaena*\n")
	require.Nil(t, err)
	tmpfile.Sync()

	// Assert
	require.Greater(t, c1SnapshotAfterRegisterWrite, 0, "Listener 1 wasn't called after the first write.")
	require.Equal(t, c2SnapshotAfterRegisterWrite, 0, "Listener 2 was called before it was registered??")
	require.Greater(t, c1SnapshotAfterRegisterWriteRegisterWrite, c1SnapshotAfterRegisterWrite, "Listener 1 wasn't called again after the second write.")
	require.Greater(t, c2SnapshotAfterRegisterWriteRegisterWrite, 0, "Listener 2 wasn't called after the second write.")
	require.Equal(t, c1SnapshotAfterRegisterWriteRegisterWrite, callCounter1, "Listener 1 was called again after being unregistered.")
	require.Equal(t, c2SnapshotAfterRegisterWriteRegisterWrite, callCounter2, "Listener 2 was called again after being unregistered.")
}

// Write the lines from input to the file in output one at a time, returning control to the Go scheduler after each write..
func slowlyWriteToLeasefile(input string, output io.WriteCloser) error {
	defer output.Close()
	infile, err := os.Open(input)
	if err != nil {
		return err
	}
	defer infile.Close()
	ln := []byte{'\n'}
	scanner := bufio.NewScanner(infile)
	for scanner.Scan() {
		output.Write(scanner.Bytes())
		output.Write(ln)
		runtime.Gosched()
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

// Read up to `limit` rows from `c`.  If the deadline in `ctx` expires before reading `limit` items, stop reading and return early.
func readChanToLimitWithTimeout(c chan []string, limit int, ctx context.Context) ([][]string, bool) {
	if limit < 0 {
		panic("you want me to read a negative number of items from the channel?")
	}
	results := make([][]string, 0, limit)
	didTimeOut := false
	for !didTimeOut && len(results) < limit {
		select {
		case row, ok := <-c:
			if !ok {
				didTimeOut = true
				break
			}
			results = append(results, row)
		case <-ctx.Done():
			didTimeOut = true
		}
	}
	return results, didTimeOut
}

func TestRowSourceExistingFile(t *testing.T) {
	// Arrange
	infile := "testdata/small-leases4.csv"
	want0 := []string{
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
	want1 := []string{
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
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	require.NoError(t, err)
	defer cancel()

	// Act
	channel := rowsource.Start()
	defer rowsource.Stop()
	got, timedOut := readChanToLimitWithTimeout(channel, 4, ctx)
	// Assert

	require.False(t, timedOut, "Timed out before getting 4 rows; got %d", len(got))
	require.Equal(t, want0, got[0])
	require.Equal(t, want1, got[1])
}

func TestRowSourceContinuesReadingOverTime(t *testing.T) {
	// Arrange
	leasefile, err := os.CreateTemp("", "leases4-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	defer os.Remove(leasefile.Name())

	infile := "testdata/small-leases4.csv"
	want0 := []string{
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
	want1 := []string{
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
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	require.NoError(t, err)
	defer cancel()

	// Act
	go slowlyWriteToLeasefile(infile, leasefile)
	channel := rowsource.Start()
	defer rowsource.Stop()
	got, timedOut := readChanToLimitWithTimeout(channel, 4, ctx)
	// Assert

	require.False(t, timedOut, "Timed out before getting 4 rows; got %d", len(got))
	require.Equal(t, want0, got[0])
	require.Equal(t, want1, got[1])
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
	wantRows := 8
	rowsource, err := NewRowSource(preCleanupName)
	require.NoError(t, err)

	// Act
	go slowlyWriteToLeasefileWithSwap(infile1, infile2, preCleanup, postCleanup)
	log.SetLevel(log.TraceLevel)
	results := rowsource.Start()

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		results,
		wantRows,
		ctx,
	)
	require.False(t, didTimeOut, "Did not read %d leases from the file before timing out; got %d", wantRows, len(parsedRows))
	require.Len(t, parsedRows, wantRows)
	require.EqualValues(t, expected1Addr, parsedRows[1][0])
	require.EqualValues(t, expected2Addr, parsedRows[5][0])
}

func TestParseRowAsLease4(t *testing.T) {
	// Arrange
	testCases := []struct {
		row     []string
		minCLTT uint64
		want    *Lease4
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
		},
	}
	for _, tc := range testCases {
		// Act
		got := ParseRowAsLease4(tc.row, tc.minCLTT)
		// Assert
		require.Equal(t, tc.want, got)
	}
}

func TestParseRowAsLease6(t *testing.T) {
	// Arrange
	testCases := []struct {
		row     []string
		minCLTT uint64
		want    *Lease6
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
		},
		// Valid IPv4 data but with CLTT too old, which it should refuse to parse.
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
		},
	}
	for _, tc := range testCases {
		// Act
		got := ParseRowAsLease6(tc.row, tc.minCLTT)
		// Assert
		require.Equal(t, tc.want, got)
	}
}

/*
// Test whether the Lease4 parser continues to read more data when Kea is writing into it over time.
func TestParseLease4(t *testing.T) {
	// Arrange
	leasefile, err := os.CreateTemp("", "leases4-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	defer os.Remove(leasefile.Name())

	infile := "testdata/small-leases4.csv"
	go slowlyWriteToLeasefile(infile, leasefile)
	parser, err := NewLease4Parser(leasefile.Name())
	require.NoError(t, err)
	expected1 := Lease4{
		"192.110.111.2",
		"03:00:00:00:00:00",
		1761257849,
		1761254249,
		3600,
		123,
		0,
	}

	// Act
	results := parser.StartParser(0)

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		results,
		3,
		ctx,
	)
	require.True(t, !didTimeOut, "Did not read 3 leases from the file before timing out")
	require.Len(t, parsedRows, 3)
	require.EqualValues(t, expected1, parsedRows[0])
}

// Test whether the Lease4 parser skips rows with CLTT lower than the filter parameter.
func TestParseLease4CLTT(t *testing.T) {
	// Arrange
	leasefile, err := os.CreateTemp("", "leases4-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	defer os.Remove(leasefile.Name())

	infile := "testdata/small-leases4.csv"
	go slowlyWriteToLeasefile(infile, leasefile)
	parser, err := NewLease4Parser(leasefile.Name())
	require.NoError(t, err)
	expected := Lease4{
		"192.110.111.3",
		"03:00:00:00:00:01",
		1761257851,
		1761254251,
		3600,
		123,
		0,
	}

	// Act
	filtered := parser.StartParser(1761254251)

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		filtered,
		2,
		ctx,
	)
	require.True(t, !didTimeOut, "Did not read 2 leases from the file before timing out")
	require.Len(t, parsedRows, 2)
	require.EqualValues(t, expected, parsedRows[0])
}

func TestParseLease4FileSwap(t *testing.T) {
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
	go slowlyWriteToLeasefileWithSwap(infile1, infile2, preCleanup, postCleanup)
	parser, err := NewLease4Parser(preCleanupName)
	require.NoError(t, err)
	expected1 := Lease4{
		"192.110.111.2",
		"03:00:00:00:00:00",
		1761257849,
		1761254249,
		3600,
		123,
		0,
	}
	expected2 := Lease4{
		"192.110.111.5",
		"03:00:00:00:00:03",
		1761257853,
		1761254253,
		3600,
		123,
		0,
	}

	// Act
	results := parser.StartParser(0)

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		results,
		6,
		ctx,
	)
	require.True(t, !didTimeOut, "Did not read 6 leases from the file before timing out")
	require.Len(t, parsedRows, 6)
	require.EqualValues(t, expected1, parsedRows[0])
	require.EqualValues(t, expected2, parsedRows[3])
}

// Test that the parser continues to read values as Kea writes them to the IPv6 leasefile.
func TestParseLease6(t *testing.T) {
	// Arrange
	leasefile, err := os.CreateTemp("", "leases6-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	defer os.Remove(leasefile.Name())

	infile := "testdata/small-leases6.csv"
	go slowlyWriteToLeasefile(infile, leasefile)
	parser, err := NewLease6Parser(leasefile.Name())
	require.NoError(t, err)
	expected1 := Lease6{
		"51a4:14ec:1::",
		"01:00:00:00:00:00",
		1761672649,
		1761669049,
		3600,
		123,
		2,
		128,
	}

	// Act
	results := parser.StartParser(0)

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		results,
		3,
		ctx,
	)
	require.True(t, !didTimeOut, "Did not read 3 leases from the file before timing out")
	require.Len(t, parsedRows, 3)
	require.EqualValues(t, expected1, parsedRows[0])
}

// Test whether the Lease6 parser skips rows with CLTT lower than the filter parameter.
func TestParseLease6CLTT(t *testing.T) {
	// Arrange
	leasefile, err := os.CreateTemp("", "leases6-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	defer os.Remove(leasefile.Name())

	infile := "testdata/small-leases6.csv"
	go slowlyWriteToLeasefile(infile, leasefile)
	parser, err := NewLease6Parser(leasefile.Name())
	require.NoError(t, err)
	expected2 := Lease6{
		"51a4:14ec:1::1",
		"01:00:00:00:00:01",
		1761672651,
		1761669051,
		3600,
		123,
		2,
		128,
	}

	// Act
	filtered := parser.StartParser(1761669051)

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		filtered,
		2,
		ctx,
	)
	require.True(t, !didTimeOut, "Did not read 2 leases from the file before timing out")
	require.Len(t, parsedRows, 2)
	require.EqualValues(t, expected2, parsedRows[0])
}

// Test that the Lease 6 parser follows file renames (see TestParseLease4FileSwap).
func TestParseLease6FileSwap(t *testing.T) {
	// Arrange
	preCleanup, err := os.CreateTemp("", "leases6-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}
	preCleanupName := preCleanup.Name()
	postCleanup, err := os.CreateTemp("", "leases6-")
	if err != nil {
		t.Errorf("unable to create temp leases file: %v", err)
	}

	infile1 := "testdata/small-leases6.csv"
	infile2 := "testdata/small2-leases6.csv"
	go slowlyWriteToLeasefileWithSwap(infile1, infile2, preCleanup, postCleanup)
	parser, err := NewLease6Parser(preCleanupName)
	require.NoError(t, err)
	expected1 := Lease6{
		"51a4:14ec:1::",
		"01:00:00:00:00:00",
		1761672649,
		1761669049,
		3600,
		123,
		2,
		128,
	}
	expected2 := Lease6{
		"51a4:14ec:1::3",
		"01:00:00:00:00:03",
		1761672653,
		1761669053,
		3600,
		123,
		2,
		128,
	}

	// Act
	results := parser.StartParser(0)

	// Assert
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()
	parsedRows, didTimeOut := readChanToLimitWithTimeout(
		results,
		6,
		ctx,
	)
	require.True(t, !didTimeOut, "Did not read 6 leases from the file before timing out")
	require.Len(t, parsedRows, 6)
	require.EqualValues(t, expected1, parsedRows[0])
	require.EqualValues(t, expected2, parsedRows[3])
}
*/
