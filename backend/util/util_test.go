package storkutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"path"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"isc.org/stork/testutil"
)

// Test that HostWithPort function generates proper output.
func TestHostWithPortURL(t *testing.T) {
	require.Equal(t, "http://localhost:1000/", HostWithPortURL("localhost", 1000, false))
	require.Equal(t, "http://192.0.2.0:1/", HostWithPortURL("192.0.2.0", 1, false))
	require.Equal(t, "https://localhost:1000/", HostWithPortURL("localhost", 1000, true))
	require.Equal(t, "https://192.0.2.0:1/", HostWithPortURL("192.0.2.0", 1, true))
}

// Test parsing URL into host and port.
func TestParseURL(t *testing.T) {
	host, port, secure := ParseURL("https://xyz:8080/")
	require.Equal(t, "xyz", host)
	require.EqualValues(t, 8080, port)
	require.True(t, secure)

	host, port, secure = ParseURL("https://[2001:db8:1::]:8080")
	require.Equal(t, "2001:db8:1::", host)
	require.EqualValues(t, 8080, port)
	require.True(t, secure)

	host, port, secure = ParseURL("http://host.example.org/")
	require.Equal(t, "host.example.org", host)
	require.EqualValues(t, 80, port)
	require.False(t, secure)

	host, port, secure = ParseURL("https://host.example.org/")
	require.Equal(t, "host.example.org", host)
	require.EqualValues(t, 443, port)
	require.True(t, secure)
}

// Test conversion of a string consisting of a string of hexadecimal
// digits with and without whitespace and with and without colons
// is successful. Also test that conversion of a string having
// invalid format is unsuccessful.
func TestFormatMACAddress(t *testing.T) {
	// Whitespace.
	formatted, ok := FormatMACAddress("01 02 03 04 05 06")
	require.True(t, ok)
	require.Equal(t, "01:02:03:04:05:06", formatted)

	// Correct format already.
	formatted, ok = FormatMACAddress("01:02:03:04:05:06")
	require.True(t, ok)
	require.Equal(t, "01:02:03:04:05:06", formatted)

	// No separator.
	formatted, ok = FormatMACAddress("aabbccddeeff")
	require.True(t, ok)
	require.Equal(t, "aa:bb:cc:dd:ee:ff", formatted)

	// Non-hexadecimal digits present.
	_, ok = FormatMACAddress("ab:cd:ef:gh")
	require.False(t, ok)

	// Invalid separator.
	_, ok = FormatMACAddress("01,02,03,04,05,06")
	require.False(t, ok)
}

// Test detection whether the text comprises an identifier
// consisting of hexadecimal digits and optionally a whitespace
// or colons.
func TestIsHexIdentifier(t *testing.T) {
	require.True(t, IsHexIdentifier("01:02:03"))
	require.True(t, IsHexIdentifier("01 e2 03"))
	require.True(t, IsHexIdentifier("abcdef "))
	require.True(t, IsHexIdentifier("12"))
	require.True(t, IsHexIdentifier(" abcd:ef"))
	require.False(t, IsHexIdentifier(" "))
	require.False(t, IsHexIdentifier("1234gh"))
	require.False(t, IsHexIdentifier("12:56:"))
	require.False(t, IsHexIdentifier("12:56:9"))
	require.False(t, IsHexIdentifier("ab,cd"))
	require.False(t, IsHexIdentifier("ab: cd"))
	require.False(t, IsHexIdentifier("abcde"))
}

// Test splitting an identifier into a slice of bytes.
func TestCountHexIdentifierBytes(t *testing.T) {
	require.Equal(t, 3, CountHexIdentifierBytes("01:02:03"))
	require.Equal(t, 3, CountHexIdentifierBytes("01::02::03"))
	require.Equal(t, 3, CountHexIdentifierBytes("01 e2 03"))
	require.Equal(t, 3, CountHexIdentifierBytes("abcdef "))
	require.Equal(t, 1, CountHexIdentifierBytes("12"))
	require.Equal(t, 3, CountHexIdentifierBytes(" abcd:ef"))

	// Invalid output for invalid input.
	require.Zero(t, CountHexIdentifierBytes(" "))
	require.Zero(t, CountHexIdentifierBytes(""))
	require.Zero(t, CountHexIdentifierBytes("1234gh"))
	require.Zero(t, CountHexIdentifierBytes("12:56:"))
	require.Zero(t, CountHexIdentifierBytes("12:56:9"))
	require.Zero(t, CountHexIdentifierBytes("ab,cd"))
	require.Zero(t, CountHexIdentifierBytes("ab: cd"))
	require.Zero(t, CountHexIdentifierBytes("abcde"))
}

// Check if BytesToHex works.
func TestBytesToHex(t *testing.T) {
	bytesArray := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	str := BytesToHex(bytesArray)
	require.Equal(t, "0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F20", str)
}

// Test conversion from hex to bytes.
func TestHexToBytes(t *testing.T) {
	require.EqualValues(t, HexToBytes("00:01:02:03:04:05:06"), []byte{0, 1, 2, 3, 4, 5, 6})
	require.EqualValues(t, HexToBytes("00-01-02-03-04-05-06"), []byte{0, 1, 2, 3, 4, 5, 6})
	require.EqualValues(t, HexToBytes("00 01 02 03 04 05 06"), []byte{0, 1, 2, 3, 4, 5, 6})
	require.EqualValues(t,
		HexToBytes("aaBB cC:Dd-Ee Ff"),
		[]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	)
	require.EqualValues(t, HexToBytes("ffeeaa"), []byte{0xff, 0xee, 0xaa})
	require.Empty(t, HexToBytes("dog"))
}

// Test read a configuration without import statements.
func TestReadConfigurationWithoutIncludes(t *testing.T) {
	path := "testdata/configs/config-without-includes.json"
	raw, err := ReadFileWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})

	require.Contains(t, data, "foo", "bar", "baz")
	foo := int(data["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := data["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := data["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with include statements.
func TestReadFileWithIncludes(t *testing.T) {
	path := "testdata/configs/config-with-includes.json"
	raw, err := ReadFileWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})
	require.Contains(t, data, "biz", "buz", "boz")

	// Non-imported content
	biz := data["biz"].(string)
	require.EqualValues(t, biz, "zib")
	boz := data["boz"].(string)
	require.EqualValues(t, boz, "zob")

	// Imported content
	buz := data["buz"].(map[string]interface{})
	require.Contains(t, buz, "foo", "bar", "baz")
	foo := int(buz["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := buz["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := buz["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with include statements without JSON extension.
func TestReadFileWithIncludesNonJSONExtension(t *testing.T) {
	path := "testdata/configs/config-with-non-json-includes.json"
	raw, err := ReadFileWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})
	require.Contains(t, data, "biz", "buz", "boz")

	// Non-imported content
	biz := data["biz"].(string)
	require.EqualValues(t, biz, "zib")
	boz := data["boz"].(string)
	require.EqualValues(t, boz, "zob")

	// Imported content
	buz := data["buz"].(map[string]interface{})
	require.Contains(t, buz, "foo", "bar", "baz")
	foo := int(buz["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := buz["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := buz["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with nested import statements.
func TestReadConfigurationWithNestedIncludes(t *testing.T) {
	path := "testdata/configs/config-with-nested-includes.json"
	raw, err := ReadFileWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	json.Unmarshal([]byte(raw), &content)
	data := content.(map[string]interface{})
	require.Contains(t, data, "ban")

	// Non-imported content
	ban := data["ban"].([]interface{})
	require.EqualValues(t, len(ban), 5)
	require.Equal(t, ban[0], float64(0))
	require.Equal(t, ban[1], float64(1))
	require.Equal(t, ban[2], float64(2))
	require.Equal(t, ban[4], float64(4))

	// 1-level nesting
	firstNest := ban[3].(map[string]interface{})
	require.Contains(t, firstNest, "biz", "buz", "boz")
	biz := firstNest["biz"].(string)
	require.EqualValues(t, biz, "zib")
	boz := firstNest["boz"].(string)
	require.EqualValues(t, boz, "zob")

	// 2-level nesting
	buz := firstNest["buz"].(map[string]interface{})
	require.Contains(t, buz, "foo", "bar", "baz")
	foo := int(buz["foo"].(float64))
	require.EqualValues(t, foo, 42)
	bar := buz["bar"].(string)
	require.EqualValues(t, bar, "24")
	baz := buz["baz"].(bool)
	require.EqualValues(t, baz, true)
}

// Test read a configuration with an infinite loop.
func TestReadConfigurationWithInfiniteLoop(t *testing.T) {
	path := "testdata/configs/config-with-infinite-loop.json"
	raw, err := ReadFileWithIncludes(path)
	require.Empty(t, raw)
	require.Error(t, err)
}

// Test read a configuration with multiple the same import statements.
func TestReadConfigurationWithMultipleTheSameIncludes(t *testing.T) {
	path := "testdata/configs/config-with-multiple-the-same-includes.json"
	raw, err := ReadFileWithIncludes(path)
	require.NoError(t, err)

	var content interface{}
	err = json.Unmarshal([]byte(raw), &content)
	require.NoError(t, err)
	data := content.(map[string]interface{})
	require.Contains(t, data, "biz", "buz", "boz")

	for _, key := range []string{"biz", "buz", "boz"} {
		nested := data[key].(map[string]interface{})
		require.Contains(t, nested, "foo", "bar", "baz")
		foo := int(nested["foo"].(float64))
		require.EqualValues(t, foo, 42)
		bar := nested["bar"].(string)
		require.EqualValues(t, bar, "24")
		baz := nested["baz"].(bool)
		require.EqualValues(t, baz, true)
	}
}

// Test read a configuration with an import statement related to a non-existing file.
func TestReadConfigurationWithNonExistingIncludes(t *testing.T) {
	path := "testdata/configs/config-with-non-existing-includes.json"
	_, err := ReadFileWithIncludes(path)
	require.Error(t, err)
}

// Test that function returns true for proper filename.
func TestIsValidFilenameForProperFilename(t *testing.T) {
	// Arrange
	filenames := []string{
		// Standard letter
		"foo.bar",
		// Numbers
		"12345667890",
		// Standard keyboard characters without *, \ and /
		"!@#$%^&()_+.{}|:\"<>?`~-=[];',.",
		// Unicode character
		"πœę©ß←↓→óþąśðæŋ’ə…ł´^¨~ ̣≥≤µń”„„ćźżż",
		// Backslash
		"\\",
	}

	for _, filename := range filenames {
		// Act
		ok := IsValidFilename(filename)
		// Assert
		require.True(t, ok)
	}
}

// Test that function returns false for invalid filenames.
func TestIsValidFilenameForInvalidFilename(t *testing.T) {
	// Arrange
	filenames := []string{
		// Asterisk
		"*",
		// Slash
		"/",
	}

	for _, filename := range filenames {
		// Act
		ok := IsValidFilename(filename)
		// Assert
		require.False(t, ok, filename)
	}
}

// Test that singular and plural noun form is returned depending
// on the count.
func TestFormatNoun(t *testing.T) {
	require.Equal(t, "0 subnets", FormatNoun(0, "subnet", "s"))
	require.Equal(t, "1 shared network", FormatNoun(1, "shared network", "s"))
	require.Equal(t, "2 parameters", FormatNoun(2, "parameter", "s"))
	require.Equal(t, "-2 parameters", FormatNoun(-2, "parameter", "s"))
	require.Equal(t, "-1 subnet", FormatNoun(-1, "subnet", "s"))
}

// Test that a nil pointer assigned to an interface is correctly
// recognized as nil. It compares the helper function and standard
// nil checking.
func TestIsNilPtr(t *testing.T) {
	// Arrange
	var iface io.Reader
	nilPtr := (*bytes.Reader)(nil)

	// Act
	iface = nilPtr

	// Assert
	require.Nil(t, nilPtr)
	require.NotEqualValues(t, iface, nil)
	require.True(t, IsNilPtr(iface))
}

// Test that a not-nil pointer assigned to an interface is correctly
// recognized as not nil.
func TestIsNotNilPtr(t *testing.T) {
	// Arrange
	var iface io.Reader
	ptr := bytes.NewReader([]byte{})

	// Act
	iface = ptr

	// Assert
	require.NotNil(t, ptr)
	require.NotEqualValues(t, iface, nil)
	require.False(t, IsNilPtr(iface))
}

// Test creating a pointer from a literal.
func TestPtr(t *testing.T) {
	// Test string.
	ptr0 := Ptr("string")
	require.NotNil(t, ptr0)
	require.Equal(t, "string", *ptr0)
	// Test int64.
	ptr := Ptr[int64](64)
	require.Equal(t, int64(64), *ptr)
}

// Test the function that checks if a specified value is a whole number.
func TestIsWholeNumber(t *testing.T) {
	// Signed integers.
	require.True(t, IsWholeNumber(int8(100)))
	require.True(t, IsWholeNumber(int16(100)))
	require.True(t, IsWholeNumber(int32(100)))
	require.True(t, IsWholeNumber(int64(100)))
	require.True(t, IsWholeNumber(int(100)))
	// Unsigned integers.
	require.True(t, IsWholeNumber(uint8(100)))
	require.True(t, IsWholeNumber(uint16(100)))
	require.True(t, IsWholeNumber(uint32(100)))
	require.True(t, IsWholeNumber(uint64(100)))
	require.True(t, IsWholeNumber(uint(100)))
	// Not whole numbers.
	require.False(t, IsWholeNumber(1.1))
	require.False(t, IsWholeNumber("foo"))
	require.False(t, IsWholeNumber(struct{}{}))
	require.False(t, IsWholeNumber(interface{}(nil)))
	u8 := uint8(123)
	require.False(t, IsWholeNumber(&u8))
}

// Test that the system command executor is constructed properly.
func TestNewSystemCommandExecutor(t *testing.T) {
	// Arrange & Act
	executor := NewSystemCommandExecutor()

	// Assert
	require.NotNil(t, executor)

	lsPath, err := executor.LookPath("ls")
	require.NotNil(t, lsPath)
	require.Nil(t, err)
	require.True(t, executor.IsFileExist(lsPath))
	sb := testutil.NewSandbox()
	defer sb.Close()
	require.False(t, executor.IsFileExist(path.Join(sb.BasePath, "not-exists")))
}

// Tests if the SET_LOG_LEVEL environment variable is used correctly to set
// logging level. Tests positive and negative cases.
func TestLoggingLevel(t *testing.T) {
	type testCase struct {
		env string
		lv  log.Level
	}

	testCases := []testCase{
		// positive cases
		{env: "DEBUG", lv: log.DebugLevel},
		{env: "INFO", lv: log.InfoLevel},
		{env: "WARN", lv: log.WarnLevel},
		{env: "ERROR", lv: log.ErrorLevel},

		// negative: if set to empty, garbage or unset at all, use the default (info)
		{env: "", lv: log.InfoLevel},
		{env: "Garbage", lv: log.InfoLevel},
		{env: "-", lv: log.InfoLevel},
	}

	// Let's remember state of the environment and revert to it after test.
	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	for _, test := range testCases {
		t.Run(test.env, func(t *testing.T) {
			if test.env != "-" {
				// special case "-" means to unset the variable
				os.Setenv("STORK_LOG_LEVEL", test.env)
			} else {
				os.Unsetenv("STORK_LOG_LEVEL")
			}
			SetupLoggingLevel()

			require.Equal(t, test.lv, log.GetLevel())
		})
	}
}

// Test that the errors are combined properly.
func TestCombineErrors(t *testing.T) {
	// Arrange
	err1 := errors.New("foo")
	err2 := errors.New("bar")
	var err3 error

	// Act
	combinedErr := CombineErrors("baz", []error{err1, err2, err3})

	// Assert
	require.ErrorContains(t, combinedErr, "baz")
	require.ErrorContains(t, combinedErr, "bar")
	require.ErrorContains(t, combinedErr, "foo")
}

// Test that the nil is returned on empty error list.
func TestCombineErrorsForEmptyList(t *testing.T) {
	// Arrange & Act
	combinedErr := CombineErrors("baz", []error{})

	// Assert
	require.Nil(t, combinedErr)
}

// Test that the nil is returned if the error list contains only nil values.
func TestCombineErrorsForListOfNils(t *testing.T) {
	// Arrange
	// Arrange & Act
	combinedErr := CombineErrors("baz", []error{nil, nil, nil})

	// Assert
	require.Nil(t, combinedErr)
}

// Test that the socket is recognized properly.
func TestSocketIsSocket(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	socketName, _ := sb.Join("socket")
	os.Remove(socketName)

	listener, _ := net.Listen("unix", socketName)
	defer listener.Close()

	// Act & Assert
	require.True(t, IsSocket(socketName))
}

// Test that the regular file is not recognized as a socket.
func TestRegularFileIsNotSocket(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	fileName, _ := sb.Join("file")

	// Act & Assert
	require.False(t, IsSocket(fileName))
}

// Test that non-existing file is not recognized as a socket.
func TestNonExistingFileIsNotSocket(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	fileName := path.Join(sb.BasePath, "not-exists")

	// Act & Assert
	require.False(t, IsSocket(fileName))
}

// Test that an empty string is converted to a nil pointer.
func TestNullifyEmptyString(t *testing.T) {
	// Non-empty string should be returned as is.
	in := "foo"
	out := NullifyEmptyString(&in)
	require.NotNil(t, out)
	require.Equal(t, "foo", *out)

	// An empty string should be converted to nil.
	in = ""
	out = NullifyEmptyString(&in)
	require.Nil(t, out)

	// nil is also legal as an input parameter and is returned as is.
	out = NullifyEmptyString(nil)
	require.Nil(t, out)
}

// Test that the boolean flag is parsed properly.
func TestParseBoolFlag(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("true")

		// Assert
		require.NoError(t, err)
		require.True(t, value)
	})

	t.Run("TRUE", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("TRUE")

		// Assert
		require.NoError(t, err)
		require.True(t, value)
	})

	t.Run("1", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("1")

		// Assert
		require.NoError(t, err)
		require.True(t, value)
	})

	t.Run("false", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("false")

		// Assert
		require.NoError(t, err)
		require.False(t, value)
	})

	t.Run("FALSE", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("FALSE")

		// Assert
		require.NoError(t, err)
		require.False(t, value)
	})

	t.Run("0", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("0")

		// Assert
		require.NoError(t, err)
		require.False(t, value)
	})

	t.Run("unknown", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("unknown")

		// Assert
		require.Error(t, err)
		require.False(t, value)
	})

	t.Run("empty", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("")

		// Assert
		require.Error(t, err)
		require.False(t, value)
	})

	t.Run("2", func(t *testing.T) {
		// Arrange & Act
		value, err := ParseBoolFlag("2")

		// Assert
		require.Error(t, err)
		require.False(t, value)
	})
}
