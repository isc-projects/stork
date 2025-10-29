package pdnsconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test parsing a valid PowerDNS configuration file.
func TestParse(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.ParseFile("testdata/pdns.conf")
	require.NoError(t, err)

	allowAxfrIps := cfg.GetValues("allow-axfr-ips")
	require.Len(t, allowAxfrIps, 3)

	address := allowAxfrIps[0].GetString()
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1", *address)
	address = allowAxfrIps[1].GetString()
	require.NotNil(t, address)
	require.Equal(t, "::1", *address)
	address = allowAxfrIps[2].GetString()
	require.NotNil(t, address)
	require.Equal(t, "172.24.0.1", *address)

	api := cfg.GetBool("api")
	require.NotNil(t, api)
	require.True(t, *api)

	apiKey := cfg.GetString("api-key")
	require.NotNil(t, apiKey)
	require.Equal(t, "stork", *apiKey)

	bindConfig := cfg.GetString("bind-config")
	require.NotNil(t, bindConfig)
	require.Equal(t, "/etc/powerdns/named.conf", *bindConfig)

	launch := cfg.GetString("launch")
	require.NotNil(t, launch)
	require.Equal(t, "bind", *launch)

	localAddress := cfg.GetString("local-address")
	require.NotNil(t, localAddress)
	require.Equal(t, "0.0.0.0", *localAddress)

	webserver := cfg.GetBool("webserver")
	require.NotNil(t, webserver)
	require.False(t, *webserver)

	webserverAddress := cfg.GetString("webserver-address")
	require.NotNil(t, webserverAddress)
	require.Equal(t, "0.0.0.0:8085", *webserverAddress)
}

// Test that parser parses integer values.
func TestParseInteger(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		max-cache-entries = 10000
	`))
	require.NoError(t, err)

	maxCacheEntries := cfg.GetInt64("max-cache-entries")
	require.NotNil(t, maxCacheEntries)
	require.EqualValues(t, 10000, *maxCacheEntries)
}

// Check that parser trims whitespace from the key and values.
func TestParseTrimSpaces(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`api-key = stork`))
	require.NoError(t, err)

	apiKey := cfg.GetString("api-key")
	require.NotNil(t, apiKey)
	require.Equal(t, "stork", *apiKey)
}

// Check that parser splits values separated by commas and spaces.
func TestParseCommasAndWhitespaces(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		only-notify = 127.0.0.1, ::1 172.24.0.1
	`))
	require.NoError(t, err)

	onlyNotify := cfg.GetValues("only-notify")
	require.Len(t, onlyNotify, 3)

	address := onlyNotify[0].GetString()
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1", *address)
	address = onlyNotify[1].GetString()
	require.NotNil(t, address)
	require.Equal(t, "::1", *address)
	address = onlyNotify[2].GetString()
	require.NotNil(t, address)
	require.Equal(t, "172.24.0.1", *address)
}

// Test that parser excludes empty values.
func TestParseExcludeEmpty(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		only-notify = 127.0.0.1, ,172.24.0.1
	`))
	require.NoError(t, err)

	onlyNotify := cfg.GetValues("only-notify")
	require.Len(t, onlyNotify, 2)

	address := onlyNotify[0].GetString()
	require.NotNil(t, address)
	require.Equal(t, "127.0.0.1", *address)
	address = onlyNotify[1].GetString()
	require.NotNil(t, address)
	require.Equal(t, "172.24.0.1", *address)
}

// Test that parser excludes empty keys.
func TestParseExcludeEmptyKey(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(`
		 = 127.0.0.1
		api-key = stork
	`))
	require.NoError(t, err)

	require.Len(t, cfg.values, 1)

	apiKey := cfg.GetString("api-key")
	require.NotNil(t, apiKey)
	require.Equal(t, "stork", *apiKey)
}

// Test that parser returns an error when a line exceeds the maximum buffer size.
func TestParseTooLong(t *testing.T) {
	parser := NewParser()
	require.NotNil(t, parser)
	cfg, err := parser.Parse(strings.NewReader(strings.Repeat("a", maxParserBufferSize+1)))
	require.Error(t, err)
	require.ErrorContains(t, err, "encountered PowerDNS configuration line exceeding the maximum buffer size")
	require.Nil(t, cfg)
}
