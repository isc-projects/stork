package agent

import (
	"bufio"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Write the lines from input to the file in output one at a time, with a delay of 300 milliseconds between each write.
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
		time.Sleep(300 * time.Millisecond)
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

// Read up to `limit` Lease4 structs from `c`.  If the deadline in `ctx` expires before reading `limit` items, stop reading and return early.
func readChanToLimitWithTimeout[T any](c chan T, limit int, ctx context.Context) ([]T, bool) {
	if limit < 0 {
		panic("you want me to read a negative number of items from the channel?")
	}
	results := make([]T, 0, limit)
	didTimeOut := false
	for !didTimeOut && len(results) < limit {
		select {
		case lease, ok := <-c:
			if !ok {
				didTimeOut = true
				break
			}
			results = append(results, lease)
		case <-ctx.Done():
			didTimeOut = true
		}
	}
	return results, didTimeOut
}

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
