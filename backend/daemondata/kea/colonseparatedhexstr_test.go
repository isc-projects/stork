package keadata

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	storktest "isc.org/stork/server/test"
)

// TestConstructCSHR verifies that [NewColonSeparatedHexStr] properly constructs a new
// [ColonSeparatedHexStr].
func TestConstructCSHR(t *testing.T) {
	// Act
	exampleStr := "01:02:03"
	exampleCSHR := NewColonSeparatedHexStr(&exampleStr)
	nilCSHR := NewColonSeparatedHexStr(nil)
	zeroCSHR := NewColonSeparatedHexStrZero()

	// Assert
	require.EqualValues(t, exampleStr, exampleCSHR.String)
	require.Nil(t, nilCSHR)
	empty := ""
	require.EqualValues(t, empty, zeroCSHR.String)
}

// TestCSHRAppendValue verifies that [AppendValue] correctly converts the conventional
// colon-separated format into the `\x0123456789abcdef` format which PostgreSQL
// expects.
func TestCSHRAppendValue(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		withQuotes  int
		expected    []byte
	}{
		{
			description: "Zero with no quotes",
			input:       "00",
			withQuotes:  0,
			expected:    []byte("\\x00"),
		},
		{
			description: "Zero with quotes",
			input:       "00",
			withQuotes:  1,
			expected:    []byte("'\\x00'"),
		},
		{
			description: "All valid hex digits",
			input:       "01:23:45:67:89:ab:cd:ef",
			withQuotes:  1,
			expected:    []byte("'\\x0123456789abcdef'"),
		},
	}
	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			cshr := NewColonSeparatedHexStr(&tc.input)

			bytes, err := cshr.AppendValue([]byte{}, tc.withQuotes)

			require.NoError(t, err)
			require.EqualValues(t, tc.expected, bytes)
		})
	}
}

// TestCSHRToString verifies that [ToString] returns the inner string as expected, or
// returns "" if the receiver is nil.
func TestCSHRToString(t *testing.T) {
	hexstr := "01:23:45:67:89:ab:cd:ef"
	example := NewColonSeparatedHexStr(&hexstr)
	var isNil *ColonSeparatedHexStr
	require.EqualValues(t, hexstr, example.ToString())
	require.EqualValues(t, "", isNil.ToString())
}

// TestCSHRScanValue verifies that [ScanValue] reads PostgreSQL's serialized format
// and converts it back to a conventional colon-separated hex string.
func TestCSHRScanValue(t *testing.T) {
	testCases := []struct {
		description         string
		input               []byte
		inputErr            error
		expectedResult      string
		expectedErrContains string
	}{
		{
			description:         "Zero",
			input:               []byte{},
			expectedResult:      "",
			expectedErrContains: "",
		},
		{
			description:         "One byte",
			input:               []byte("\\x00"),
			expectedResult:      "00",
			expectedErrContains: "",
		},
		{
			description:         "All valid hex digits",
			input:               []byte("\\x0123456789abcdef"),
			expectedResult:      "01:23:45:67:89:ab:cd:ef",
			expectedErrContains: "",
		},
		{
			description:         "Error",
			input:               []byte("\\x0001"),
			inputErr:            errors.New("foobar2000"),
			expectedResult:      "",
			expectedErrContains: "foobar2000",
		},
	}
	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			reader := storktest.NewPoolReaderMock(tc.input, tc.inputErr)
			cshr := NewColonSeparatedHexStrZero()

			err := cshr.ScanValue(reader, len(tc.input))

			if tc.expectedErrContains != "" {
				require.ErrorContains(t, err, tc.expectedErrContains)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, tc.expectedResult, cshr.String)
			}
		})
	}
}

// TestCSHRUnmarshalJSON verifies that [UnmarshalJSON] reads a plain JSON string into
// a [ColonSeparatedHexStr].
func TestCSHRUnmarshalJSON(t *testing.T) {
	actual := NewColonSeparatedHexStrZero()
	expected := "aa:bb:cc:dd"
	input := []byte("\"aa:bb:cc:dd\"")

	err := actual.UnmarshalJSON(input)

	require.NoError(t, err)
	require.EqualValues(t, expected, actual.String)
}

// TestCSHRMarshalJSON verifies that [MarshalJSON] writes a [ColonSeparatedHexStr] as
// a plain JSON string (rather than an object).
func TestCSHRMarshalJSON(t *testing.T) {
	inputStr := "00:11:22:33"
	input := NewColonSeparatedHexStr(&inputStr)
	expected := []byte("\"00:11:22:33\"")

	actual, err := input.MarshalJSON()
	require.NoError(t, err)
	require.EqualValues(t, expected, actual)
}
