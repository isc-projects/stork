package storkutil

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.EqualValues(t, HexToBytes("ffeeaa"), []byte{0xff, 0xee, 0xaa})
	require.Empty(t, HexToBytes("dog"))
}

// Test read a configuration without import statements.
func TestReadConfigurationWithoutIncludes(t *testing.T) {
	path := "configs/config-without-includes.json"
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
	path := "configs/config-with-includes.json"
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
	path := "configs/config-with-nested-includes.json"
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
	path := "configs/config-with-infinite-loop.json"
	raw, err := ReadFileWithIncludes(path)
	require.Empty(t, raw)
	require.Error(t, err)
}

// Test read a configuration with multiple the same import statements.
func TestReadConfigurationWithMultipleTheSameIncludes(t *testing.T) {
	path := "configs/config-with-multiple-the-same-includes.json"
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
	path := "configs/config-with-non-existing-includes.json"
	_, err := ReadFileWithIncludes(path)
	require.Error(t, err)
}

// Test that the sensitive data are hidden.
func TestHideSensitiveData(t *testing.T) {
	// Arrange
	data := map[string]interface{}{
		"foo":      "bar",
		"password": "xxx",
		"token":    "",
		"secret":   "aaa",
		"first": map[string]interface{}{
			"foo":      "baz",
			"Password": 42,
			"Token":    nil,
			"Secret":   "bbb",
			"second": map[string]interface{}{
				"foo":      "biz",
				"passworD": true,
				"tokeN":    "yyy",
				"secreT":   "ccc",
			},
		},
	}

	// Act
	HideSensitiveData(&data)

	// Assert
	// Top level
	require.EqualValues(t, "bar", data["foo"])
	require.EqualValues(t, nil, data["password"])
	require.EqualValues(t, nil, data["token"])
	require.EqualValues(t, nil, data["secret"])
	// First level of the nesting
	first := data["first"].(map[string]interface{})
	require.EqualValues(t, "baz", first["foo"])
	require.EqualValues(t, nil, first["Password"])
	require.EqualValues(t, nil, first["Token"])
	require.EqualValues(t, nil, first["Secret"])
	// Second level of the nesting
	second := first["second"].(map[string]interface{})
	require.EqualValues(t, "biz", second["foo"])
	require.EqualValues(t, nil, second["passworD"])
	require.EqualValues(t, nil, second["tokeN"])
	require.EqualValues(t, nil, second["secreT"])
}

// Function for a valid prefix should return no error.
func TestParseTimestampPrefixNoErrorForValid(t *testing.T) {
	// Arrange
	timestamp := time.Time{}.Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("%s_foo.ext", timestamp)

	// Act
	_, _, err := ParseTimestampPrefix(filename)

	// Assert
	require.NoError(t, err)
}

// Function for a missing delimiter in prefix should return error.
func TestParseTimestampPrefixErrorForNoDelimiter(t *testing.T) {
	// Arrange
	timestamp := time.Time{}.Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("%sfoo.ext", timestamp)

	// Act
	_, _, err := ParseTimestampPrefix(filename)

	// Assert
	require.Error(t, err)
}

// Function for a invalid prefix should return error.
func TestParseTimestampPrefixErrorForInvalid(t *testing.T) {
	// Arrange
	timestamp := "bar"
	filename := fmt.Sprintf("%s_foo.ext", timestamp)

	// Act
	_, _, err := ParseTimestampPrefix(filename)

	// Assert
	require.Error(t, err)
}

// Function for too short prefix should return error.
func TestParseTimestampPrefixTooShort(t *testing.T) {
	// Arrange
	timestamp := "2021-11-15T12:00:00"
	filename := fmt.Sprintf("%s_foo.ext", timestamp)

	// Act
	_, _, err := ParseTimestampPrefix(filename)

	// Assert
	require.Error(t, err)
}

// Function for a valid prefix should return rest of filename.
func TestParseTimestampPrefixRestOfFilenameForValid(t *testing.T) {
	// Arrange
	timestamp := time.Time{}.Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("%s_foo-bar.ext", timestamp)

	// Act
	_, prefix, _ := ParseTimestampPrefix(filename)

	// Assert
	require.EqualValues(t, "_foo-bar.ext", prefix)
}

// Function for a valid prefix should return the parsed timestamp.
func TestParseTimestampPrefixTimestampForValid(t *testing.T) {
	// Arrange
	timestamp := time.Time{}.Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("%s_foo.ext", timestamp)

	// Act
	timestampObj, _, _ := ParseTimestampPrefix(filename)

	// Assert
	require.EqualValues(t, time.Time{}, timestampObj)
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
