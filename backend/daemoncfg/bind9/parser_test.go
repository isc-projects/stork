package bind9config

import (
	"fmt"
	"iter"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test that the parsed configuration is the same as the original configuration
// stored in the testdata/dir/named.conf file.
func testParsedConfig(t *testing.T, cfg *Config) {
	require.Len(t, cfg.Statements, 12)

	next, stop := iter.Pull(slices.Values(cfg.Statements))
	defer stop()

	statement, _ := next()
	require.NotNil(t, statement.Key)
	require.Equal(t, "trusted-key", statement.Key.Name)
	require.Len(t, statement.Key.Clauses, 2)
	require.Equal(t, "hmac-sha256", statement.Key.Clauses[0].Algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", statement.Key.Clauses[1].Secret)

	statement, _ = next()
	require.NotNil(t, statement.Key)
	require.Equal(t, "guest-key", statement.Key.Name)
	require.Len(t, statement.Key.Clauses, 2)
	require.Equal(t, "hmac-sha256", statement.Key.Clauses[0].Algorithm)
	require.Equal(t, "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=", statement.Key.Clauses[1].Secret)

	statement, _ = next()
	require.NotNil(t, statement.ACL)
	require.Equal(t, "trusted-networks", statement.ACL.Name)
	require.Len(t, statement.ACL.AddressMatchList.Elements, 8)

	statement, _ = next()
	require.NotNil(t, statement.ACL)
	require.Equal(t, "guest-networks", statement.ACL.Name)
	require.Len(t, statement.ACL.AddressMatchList.Elements, 6)

	statement, _ = next()
	require.NotNil(t, statement.Controls)
	require.Len(t, statement.Controls.Clauses, 2)
	// Inet clause.
	require.NotNil(t, statement.Controls.Clauses[0].InetClause)
	require.Equal(t, "*", statement.Controls.Clauses[0].InetClause.Address)
	require.Empty(t, statement.Controls.Clauses[0].InetClause.Port)
	require.NotNil(t, statement.Controls.Clauses[0].InetClause.Allow)
	require.Len(t, statement.Controls.Clauses[0].InetClause.Allow.Elements, 1)
	require.NotNil(t, statement.Controls.Clauses[0].InetClause.Keys)
	require.Len(t, statement.Controls.Clauses[0].InetClause.Keys.KeyNames, 1)
	require.Nil(t, statement.Controls.Clauses[0].InetClause.ReadOnly)
	// Unix clause.
	require.NotNil(t, statement.Controls.Clauses[1].UnixClause)
	require.Equal(t, "/run/named/rndc.sock", statement.Controls.Clauses[1].UnixClause.Path)
	require.EqualValues(t, 0o600, statement.Controls.Clauses[1].UnixClause.Perm)
	require.EqualValues(t, 25, statement.Controls.Clauses[1].UnixClause.Owner)
	require.EqualValues(t, 26, statement.Controls.Clauses[1].UnixClause.Group)
	require.NotNil(t, statement.Controls.Clauses[1].UnixClause.Keys)
	require.Len(t, statement.Controls.Clauses[1].UnixClause.Keys.KeyNames, 1)

	statement, _ = next()
	require.NotNil(t, statement.StatisticsChannels)
	require.Len(t, statement.StatisticsChannels.Clauses, 1)
	require.NotNil(t, statement.StatisticsChannels.Clauses[0])
	require.Equal(t, "127.0.0.1", statement.StatisticsChannels.Clauses[0].Address)
	require.NotNil(t, statement.StatisticsChannels.Clauses[0].Port)
	require.Equal(t, "8053", *statement.StatisticsChannels.Clauses[0].Port)
	require.NotNil(t, statement.StatisticsChannels.Clauses[0].Allow)
	require.Len(t, statement.StatisticsChannels.Clauses[0].Allow.Elements, 1)
	require.Equal(t, "127.0.0.1", statement.StatisticsChannels.Clauses[0].Allow.Elements[0].IPAddressOrACLName)

	statement, _ = next()
	require.NotNil(t, statement.Option)
	require.Equal(t, "tls", statement.Option.Identifier)
	require.Len(t, statement.Option.Switches, 1)
	require.Equal(t, "domain.name", statement.Option.Switches[0].GetStringValue())

	statement, _ = next()
	require.NotNil(t, statement.Options)
	require.Len(t, statement.Options.Clauses, 10)
	require.NotNil(t, statement.Options.Clauses[0].Option)
	require.Equal(t, "allow-query", statement.Options.Clauses[0].Option.Identifier)
	require.NotNil(t, statement.Options.Clauses[1].AllowTransfer)
	require.Nil(t, statement.Options.Clauses[1].AllowTransfer.Port)
	require.Nil(t, statement.Options.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.Options.Clauses[1].AllowTransfer.AddressMatchList)
	require.Len(t, statement.Options.Clauses[1].AllowTransfer.AddressMatchList.Elements, 1)
	require.Equal(t, "any", statement.Options.Clauses[1].AllowTransfer.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.NotNil(t, statement.Options.Clauses[2].Option)
	require.Equal(t, "also-notify", statement.Options.Clauses[2].Option.Identifier)
	require.NotNil(t, statement.Options.Clauses[3].Option)
	require.Equal(t, "check-names", statement.Options.Clauses[3].Option.Identifier)
	require.NotNil(t, statement.Options.Clauses[4].Option)
	require.Equal(t, "dnssec-validation", statement.Options.Clauses[4].Option.Identifier)
	require.NotNil(t, statement.Options.Clauses[5].Option)
	require.Equal(t, "recursion", statement.Options.Clauses[5].Option.Identifier)
	require.NotNil(t, statement.Options.Clauses[6].ListenOn)
	require.NotNil(t, statement.Options.Clauses[6].ListenOn.Port)
	require.EqualValues(t, 8553, *statement.Options.Clauses[6].ListenOn.Port)
	require.NotNil(t, statement.Options.Clauses[6].ListenOn.Proxy)
	require.Equal(t, "plain", *statement.Options.Clauses[6].ListenOn.Proxy)
	require.NotNil(t, statement.Options.Clauses[6].ListenOn.TLS)
	require.Equal(t, "domain.name", *statement.Options.Clauses[6].ListenOn.TLS)
	require.NotNil(t, statement.Options.Clauses[6].ListenOn.HTTP)
	require.Equal(t, "myserver", *statement.Options.Clauses[6].ListenOn.HTTP)
	require.NotNil(t, statement.Options.Clauses[6].ListenOn.AddressMatchList)
	require.Len(t, statement.Options.Clauses[6].ListenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "127.0.0.1", statement.Options.Clauses[6].ListenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.NotNil(t, statement.Options.Clauses[7].ListenOn)
	require.NotNil(t, statement.Options.Clauses[7].ListenOn.AddressMatchList)
	require.Len(t, statement.Options.Clauses[7].ListenOn.AddressMatchList.Elements, 1)
	require.Equal(t, "::1", statement.Options.Clauses[7].ListenOn.AddressMatchList.Elements[0].IPAddressOrACLName)
	require.NotNil(t, statement.Options.Clauses[8].ResponsePolicy)
	require.Len(t, statement.Options.Clauses[8].ResponsePolicy.Zones, 2)
	require.Equal(t, "rpz.example.com", statement.Options.Clauses[8].ResponsePolicy.Zones[0].Zone)
	require.Len(t, statement.Options.Clauses[8].ResponsePolicy.Zones[0].Switches, 6)
	require.Equal(t, "db.local", statement.Options.Clauses[8].ResponsePolicy.Zones[1].Zone)
	require.Empty(t, statement.Options.Clauses[8].ResponsePolicy.Zones[1].Switches)
	require.Len(t, statement.Options.Clauses[8].ResponsePolicy.Switches, 2)
	require.Equal(t, "update-interval", statement.Options.Clauses[8].ResponsePolicy.Switches[0])
	require.Equal(t, "100", statement.Options.Clauses[8].ResponsePolicy.Switches[1])
	require.Equal(t, "deny-answer-aliases", statement.Options.Clauses[9].Option.Identifier)
	require.NotNil(t, statement.Options.Clauses[9].Option.Contents)
	require.Len(t, statement.Options.Clauses[9].Option.Suboptions, 1)
	require.NotNil(t, statement.Options.Clauses[9].Option.Suboptions[0].Contents)

	statement, _ = next()
	require.NotNil(t, statement.View)
	require.Equal(t, "trusted", statement.View.Name)
	require.Len(t, statement.View.Clauses, 8)
	require.NotNil(t, statement.View.Clauses[0].MatchClients)
	require.Len(t, statement.View.Clauses[0].MatchClients.AddressMatchList.Elements, 1)
	require.NotNil(t, statement.View.Clauses[1].AllowTransfer)
	require.EqualValues(t, 853, *statement.View.Clauses[1].AllowTransfer.Port)
	require.Equal(t, "tls", *statement.View.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.View.Clauses[1].AllowTransfer.AddressMatchList)
	require.NotNil(t, statement.View.Clauses[2].Option)
	require.Equal(t, "recursion", statement.View.Clauses[2].Option.Identifier)
	require.NotNil(t, statement.View.Clauses[3].ResponsePolicy)
	require.Len(t, statement.View.Clauses[3].ResponsePolicy.Zones, 1)
	require.NotNil(t, statement.View.Clauses[4].Zone)
	require.Equal(t, "bind9.example.com", statement.View.Clauses[4].Zone.Name)
	require.NotNil(t, statement.View.Clauses[5].Zone)
	require.Equal(t, "pdns.example.com", statement.View.Clauses[5].Zone.Name)
	require.NotNil(t, statement.View.Clauses[6].Zone)
	require.Equal(t, "rpz.example.com", statement.View.Clauses[6].Zone.Name)
	require.NotNil(t, statement.View.Clauses[7].Option)
	require.Equal(t, "allow-new-zones", statement.View.Clauses[7].Option.Identifier)
	require.Len(t, statement.View.Clauses[7].Option.Switches, 1)
	require.Equal(t, "no", statement.View.Clauses[7].Option.Switches[0].GetStringValue())

	statement, _ = next()
	require.NotNil(t, statement.View)
	require.Equal(t, "guest", statement.View.Name)
	require.Len(t, statement.View.Clauses, 4)
	require.NotNil(t, statement.View.Clauses[0].MatchClients)
	require.Len(t, statement.View.Clauses[0].MatchClients.AddressMatchList.Elements, 1)
	require.NotNil(t, statement.View.Clauses[1].AllowTransfer)
	require.Nil(t, statement.View.Clauses[1].AllowTransfer.Port)
	require.Equal(t, "tls", *statement.View.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.View.Clauses[1].AllowTransfer.AddressMatchList)
	require.NotNil(t, statement.View.Clauses[2].Zone)
	require.Equal(t, "bind9.example.org", statement.View.Clauses[2].Zone.Name)
	require.NotNil(t, statement.View.Clauses[3].Option)
	require.Equal(t, "deny-answer-addresses", statement.View.Clauses[3].Option.Identifier)
	require.NotNil(t, statement.View.Clauses[3].Option.Contents)
	require.Empty(t, statement.View.Clauses[3].Option.Suboptions)

	statement, _ = next()
	require.NotNil(t, statement.Zone)
	require.Equal(t, "nsd.example.com", statement.Zone.Name)
	require.Equal(t, "IN", statement.Zone.Class)
	require.NotNil(t, statement.Zone.Clauses)
	require.Len(t, statement.Zone.Clauses, 3)
	require.NotNil(t, statement.Zone.Clauses[0].Option)
	require.Equal(t, "type", statement.Zone.Clauses[0].Option.Identifier)
	require.Len(t, statement.Zone.Clauses[0].Option.Switches, 1)
	require.Equal(t, "master", statement.Zone.Clauses[0].Option.Switches[0].GetStringValue())
	require.NotNil(t, statement.Zone.Clauses[1].AllowTransfer)
	require.EqualValues(t, 853, *statement.Zone.Clauses[1].AllowTransfer.Port)
	require.Nil(t, statement.Zone.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.Zone.Clauses[1].AllowTransfer.AddressMatchList)
	require.NotNil(t, statement.Zone.Clauses[2].Option)
	require.Equal(t, "file", statement.Zone.Clauses[2].Option.Identifier)
	require.Len(t, statement.Zone.Clauses[2].Option.Switches, 1)
	require.Equal(t, "/etc/bind/db.nsd.example.com", statement.Zone.Clauses[2].Option.Switches[0].GetStringValue())

	statement, _ = next()
	require.NotNil(t, statement.Option)
	require.Equal(t, "logging", statement.Option.Identifier)
}

// Test successfully parsing the named configuration file.
func TestParseFile(t *testing.T) {
	// Parse the named configuration file without expanding includes.
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	configPathAbs, _ := filepath.Abs("testdata/dir/named.conf")
	require.Equal(t, configPathAbs, cfg.sourcePath)
	require.Empty(t, cfg.chrootDir)

	// Expand included files.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	testParsedConfig(t, cfg)
}

// Test successfully parsing the chroot named configuration file.
func TestParseFileChroot(t *testing.T) {
	// Parse the named configuration file without expanding includes.
	cfg, err := NewParser().ParseFile("dir/named.conf", "testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "/dir/named.conf", cfg.sourcePath)
	testdataPathAbs, _ := filepath.Abs("testdata")
	require.Equal(t, testdataPathAbs, cfg.chrootDir)

	// Expand included files.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	testParsedConfig(t, cfg)
}

// Test successfully parsing the chroot named configuration file.
// The include files are located in the root of the chroot.
func TestParseFileChrootNoSubdirectory(t *testing.T) {
	// Parse the named configuration file without expanding includes.
	cfg, err := NewParser().ParseFile("named.conf", "testdata/dir")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Expand included files.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	testParsedConfig(t, cfg)
}

// Test that a complex generic option is parsed correctly. The option contains
// multiple switches and the clause with generic contents. It also contains two
// suboptions. The switches are both strings and identifiers.
func TestParseGenericOption(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		options {
			test-option "192.168.1.1" 2001:db8:1::1 12 bar 1.1.1.1 {
				generic-content 1 2;
			} test-suboption1 192.0.2.1 {
				generic-content 3 4;
            } test-suboption2 smiley "123";
		}
	`))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// There is one top-level option.
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Options)
	require.Len(t, cfg.Statements[0].Options.Clauses, 1)

	// Validate the option.
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option)
	require.Equal(t, "test-option", cfg.Statements[0].Options.Clauses[0].Option.Identifier)
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Switches, 5)
	require.Equal(t, "192.168.1.1", cfg.Statements[0].Options.Clauses[0].Option.Switches[0].GetStringValue())
	require.Equal(t, "2001:db8:1::1", cfg.Statements[0].Options.Clauses[0].Option.Switches[1].GetStringValue())
	require.Equal(t, "12", cfg.Statements[0].Options.Clauses[0].Option.Switches[2].GetStringValue())
	require.Equal(t, "bar", cfg.Statements[0].Options.Clauses[0].Option.Switches[3].GetStringValue())
	require.Equal(t, "1.1.1.1", cfg.Statements[0].Options.Clauses[0].Option.Switches[4].GetStringValue())
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option.Contents)
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Contents.tokens, 4)
	require.Equal(t, "generic-content", cfg.Statements[0].Options.Clauses[0].Option.Contents.tokens[0])
	require.Equal(t, "1", cfg.Statements[0].Options.Clauses[0].Option.Contents.tokens[1])
	require.Equal(t, "2", cfg.Statements[0].Options.Clauses[0].Option.Contents.tokens[2])
	require.Equal(t, ";", cfg.Statements[0].Options.Clauses[0].Option.Contents.tokens[3])

	// The option contains two suboptions: test-suboption1 and test-suboption2.
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions, 2)

	// Validate the first suboption.
	require.Equal(t, "test-suboption1", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Identifier)
	require.Equal(t, "192.0.2.1", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Switches[0].GetStringValue())
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents)
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents.tokens, 4)
	require.Equal(t, "generic-content", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents.tokens[0])
	require.Equal(t, "3", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents.tokens[1])
	require.Equal(t, "4", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents.tokens[2])
	require.Equal(t, ";", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents.tokens[3])

	// Validate the second suboption.
	require.Equal(t, "test-suboption2", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[1].Identifier)
	require.Equal(t, "smiley", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[1].Switches[0].GetStringValue())
	require.Equal(t, "123", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[1].Switches[1].GetStringValue())
	require.Nil(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions[1].Contents)
}

// Test that the config can be serialized when filter is not set or when
// filter selects all types of statements.
func TestConfigGetFormattedString(t *testing.T) {
	// Parse the configuration file.
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Expand the included files.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	t.Run("no filtering", func(t *testing.T) {
		// Serialize the configuration without filtering.
		var builder strings.Builder
		for text, err := range cfg.GetFormattedTextIterator(1, nil) {
			require.NoError(t, err)
			builder.WriteString(text)
			builder.WriteString("\n")
		}
		// By parsing the output we ensure that the output configuration syntax
		// is valid.
		cfg2, err := NewParser().Parse("", "", strings.NewReader(builder.String()))
		require.NoError(t, err)
		require.NotNil(t, cfg2)
		// Verify that the parsed configuration is the same as the original configuration.
		testParsedConfig(t, cfg2)
	})

	t.Run("filtering", func(t *testing.T) {
		// Serialize the configuration with filtering.
		var builder strings.Builder
		for text, err := range cfg.GetFormattedTextIterator(0, NewFilter(FilterTypeConfig, FilterTypeView, FilterTypeZone, FilterTypeNoParse)) {
			require.NoError(t, err)
			builder.WriteString(text)
			builder.WriteString("\n")
		}
		// By parsing the output we ensure that the output configuration syntax
		// is valid.
		cfg2, err := NewParser().Parse("", "", strings.NewReader(builder.String()))
		require.NoError(t, err)
		require.NotNil(t, cfg2)
		// Verify that the parsed configuration is the same as the original
		// configuration.
		testParsedConfig(t, cfg2)
	})
}

// Test that the config can be serialized when filter is set to select
// specific types of statements.
func TestConfigGetFormattedStringFiltering(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	t.Run("view", func(t *testing.T) {
		var builder strings.Builder
		for text, err := range cfg.GetFormattedTextIterator(0, NewFilter(FilterTypeView)) {
			require.NoError(t, err)
			builder.WriteString(text)
			builder.WriteString("\n")
		}
		cfg2, err := NewParser().Parse("", "", strings.NewReader(builder.String()))
		require.NoError(t, err)
		require.NotNil(t, cfg2)
		require.Len(t, cfg2.Statements, 2)
		for _, statement := range cfg2.Statements {
			require.NotNil(t, statement.View)
		}
	})

	t.Run("zone", func(t *testing.T) {
		var builder strings.Builder
		for text, err := range cfg.GetFormattedTextIterator(0, NewFilter(FilterTypeZone)) {
			require.NoError(t, err)
			builder.WriteString(text)
			builder.WriteString("\n")
		}
		cfg2, err := NewParser().Parse("", "", strings.NewReader(builder.String()))
		require.NoError(t, err)
		require.NotNil(t, cfg2)
		require.Len(t, cfg2.Statements, 1)
		for _, statement := range cfg2.Statements {
			require.NotNil(t, statement.Zone)
		}
	})

	t.Run("config options", func(t *testing.T) {
		var builder strings.Builder
		for text, err := range cfg.GetFormattedTextIterator(0, NewFilter(FilterTypeConfig)) {
			require.NoError(t, err)
			builder.WriteString(text)
			builder.WriteString("\n")
		}
		cfg2, err := NewParser().Parse("", "", strings.NewReader(builder.String()))
		require.NoError(t, err)
		require.NotNil(t, cfg2)
		require.Len(t, cfg2.Statements, 8)
		for _, statement := range cfg2.Statements {
			require.Nil(t, statement.View)
			require.Nil(t, statement.Zone)
			require.Nil(t, statement.NoParse)
		}
	})
}

// Test an error is return upon trying to emit too long line under the
// no-parse directive.
func TestConfigGetFormattedTextTooLongLine(t *testing.T) {
	// Generate a too long line.
	longLine := strings.Repeat("a", maxFormatterLinesBufferSize+1)
	// Generate the configuration starting with valid configuration but ending
	// with two consecutive too long lines.
	configText := fmt.Sprintf(`
		zone "example.com" {
			type forward;
		};
		//@stork:no-parse:global
		%s;
		%s;
	`, longLine, longLine)

	// Parse the configuration.
	cfg, err := NewParser().Parse("", "", strings.NewReader(configText))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	var (
		lines  []string
		errors []error
	)
	// Serialize the configuration. This mechanism, when emitting the contents of the
	// no-parse directive, will allocate a buffer. If the buffer's capacity is exceeded,
	// it should return an error. The first lines should be emitted successfully.
	// The process should stop after the first error.
	for text, err := range cfg.GetFormattedTextIterator(0, nil) {
		if err != nil {
			errors = append(errors, err)
		} else {
			lines = append(lines, text)
		}
	}
	// Verify that the first lines were emitted successfully.
	require.Len(t, lines, 4)
	// Verify that the process stopped after the first error.
	require.Len(t, errors, 1)
}

// Test that the parser correctly handles the @stork:no-parse directive.
func TestNoParseSelectedZone(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		zone "example.com" {
			type forward;
		};
		//@stork:no-parse:scope
		zone "example.org" {
			type forward;
		};
		//@stork:no-parse:end
		zone "example.net" {
			type forward;
		};
	`))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 3)
	require.NotNil(t, cfg.Statements[0].Zone)
	require.Equal(t, "example.com", cfg.Statements[0].Zone.Name)
	require.NotNil(t, cfg.Statements[1].NoParse)
	require.False(t, cfg.Statements[1].NoParse.IsGlobal())
	require.Contains(t, cfg.Statements[1].NoParse.GetContentsString(),
		`zone "example.org" {
			type forward;
		};`)
	require.NotNil(t, cfg.Statements[2].Zone)
	require.Equal(t, "example.net", cfg.Statements[2].Zone.Name)
}

// Test selectively skipping parsing the inner contents of a zone definition.
func TestNoParseSelectedZoneOptions(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		zone "example.org" {
			//@stork:no-parse:scope
			type forward;
			//@stork:no-parse:end
			allow-transfer port 853 { any; };
		};
	`))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Zone)
	require.Equal(t, "example.org", cfg.Statements[0].Zone.Name)
	require.Len(t, cfg.Statements[0].Zone.Clauses, 2)
	require.NotNil(t, cfg.Statements[0].Zone.Clauses[0].NoParse)
	require.False(t, cfg.Statements[0].Zone.Clauses[0].NoParse.IsGlobal())
	require.Contains(t, cfg.Statements[0].Zone.Clauses[0].NoParse.GetContentsString(), "type forward;")
	require.NotNil(t, cfg.Statements[0].Zone.Clauses[1].AllowTransfer)
	require.EqualValues(t, 853, *cfg.Statements[0].Zone.Clauses[1].AllowTransfer.Port)
}

// Test that the parser correctly handles the @stork:no-parse directive
// for the view options.
func TestNoParseViewOptions(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		view "foo" {
			zone "example.com" {
				type primary;
			};
			//@stork:no-parse:scope
			zone "example.net" {
				type primary;
			};
			//@stork:no-parse:end
			zone "example.org" {
				type primary;
			};
		};
	`))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].View)
	require.Equal(t, "foo", cfg.Statements[0].View.Name)
	require.Len(t, cfg.Statements[0].View.Clauses, 3)
	require.NotNil(t, cfg.Statements[0].View.Clauses[0].Zone)
	require.Equal(t, "example.com", cfg.Statements[0].View.Clauses[0].Zone.Name)
	require.NotNil(t, cfg.Statements[0].View.Clauses[1].NoParse)
	require.False(t, cfg.Statements[0].View.Clauses[1].NoParse.IsGlobal())
	require.NotNil(t, cfg.Statements[0].View.Clauses[2].Zone)
	require.Equal(t, "example.org", cfg.Statements[0].View.Clauses[2].Zone.Name)
}

// Test that the parser correctly handles the @stork:no-parse directive
// for the options.
func TestNoParseOptions(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		options {
			allow-transfer port 853 { any; };
			//@stork:no-parse:scope
			listen-on port 853 { 127.0.0.1; };
			//@stork:no-parse:end
			listen-on-v6 port 853 { ::1; };
		};
	`))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Options)
	require.Len(t, cfg.Statements[0].Options.Clauses, 3)
}

// Test that an error is returned when the @stork:no-parse:scope is not
// followed by the @stork:no-parse:end directive.
func TestNoParseNoEnd(t *testing.T) {
	_, err := NewParser().Parse("", "", strings.NewReader(`
		zone "example.com" {
			type forward;
		};
		//@stork:no-parse:scope
		zone "example.org" {
			type forward;
		};
		zone "example.net" {
			type forward;
		};
	`))
	require.Error(t, err)
	require.ErrorContains(t, err, `expected <noparsecontents> <noparseend>`)
}

// Test that the @stork:no-parse:global directive is correctly parsed
// and parsing the rest of the file is skipped.
func TestNoParseGlobal(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		zone "example.com" {
			type forward;
		};
		//@stork:no-parse:global
		zone "example.org" {
			type forward;
		};
	`))
	require.NoError(t, err)
	require.Len(t, cfg.Statements, 2)
	require.NotNil(t, cfg.Statements[0].Zone)
	require.Equal(t, "example.com", cfg.Statements[0].Zone.Name)
	require.Len(t, cfg.Statements[0].Zone.Clauses, 1)
	require.NotNil(t, cfg.Statements[1].NoParse)
	require.True(t, cfg.Statements[1].NoParse.IsGlobal())
	require.Contains(t, cfg.Statements[1].NoParse.GetContentsString(),
		`zone "example.org" {
			type forward;
		};`)
}

// Test that an error is returned when the @stork:no-parse:global directive
// is used in the middle of a statement.
func TestNoParseGlobalMidStatement(t *testing.T) {
	_, err := NewParser().Parse("", "", strings.NewReader(`
		zone "example.com" {
			//@stork:no-parse:global
			type forward;
		};
	`))
	require.Error(t, err)
	require.ErrorContains(t, err, `(expected "}")`)
}

// Test that the @stork:no-parse:end is ignored for the @stork:no-parse:global
// directive.
func TestNoParseGlobalExtraneousEnd(t *testing.T) {
	cfg, err := NewParser().Parse("", "", strings.NewReader(`
		zone "example.com" {
			type forward;
		};
		//@stork:no-parse:global
		zone "example.org" {
			type forward;
			allow-transfer port 853 { any; };
		};
		//@stork:no-parse:end
		zone "example.net" {
			type forward;
		};
	`))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 2)
	require.NotNil(t, cfg.Statements[0].Zone)
	require.Equal(t, "example.com", cfg.Statements[0].Zone.Name)
	require.NotNil(t, cfg.Statements[1].NoParse)
	require.True(t, cfg.Statements[1].NoParse.IsGlobal())
}

// Test that an error is returned when the @stork:no-parse:end directive
// is used without the @stork:no-parse:scope directive.
func TestNoParseOnlyEnd(t *testing.T) {
	_, err := NewParser().Parse("", "", strings.NewReader(`
		//@stork:no-parse:end
		zone "example.com" {
			type forward;
		};
	`))
	require.Error(t, err)
	require.ErrorContains(t, err, `unexpected token "end"`)
}

// Test that an attempt to parse a non-existent file returns an error.
func TestParseFileError(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/non-existent.conf", "")
	require.Error(t, err)
	require.Nil(t, cfg)
}

// Test that the parser correctly handles the include statements.
func TestParseIncludes(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	topLevelPath, err := sandbox.Join("top-level.conf")
	require.NoError(t, err)
	includedPath, err := sandbox.Join("2.conf")
	require.NoError(t, err)

	// Create the parent file with the include statements. The path
	// to the first file is relative. The path to the second file is
	// absolute.
	sandbox.Write("top-level.conf", fmt.Sprintf(`
		include "1.conf";
		include "%s";
	`, includedPath))

	// Create the first included file.
	sandbox.Write("1.conf", `
		acl test1 {
			1.2.3.4;
		};
	`)

	// Create the second included file.
	sandbox.Write("2.conf", `
		acl test2 {
			0.0.0.0;
		};
	`)

	// Parse the parent file without expanding includes. All
	// statements must be includes.
	cfg, err := NewParser().ParseFile(topLevelPath, "")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 2)
	require.NotNil(t, cfg.Statements[0].Include)
	require.Equal(t, "1.conf", cfg.Statements[0].Include.Path)
	require.NotNil(t, cfg.Statements[1].Include)
	require.Equal(t, includedPath, cfg.Statements[1].Include.Path)

	// Expand the includes.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The new statements must be ACLs.
	require.Len(t, cfg.Statements, 2)
	acl1 := cfg.GetACL("test1")
	require.NotNil(t, acl1)
	require.Equal(t, "test1", acl1.Name)
	require.Len(t, acl1.AddressMatchList.Elements, 1)
	require.Equal(t, "1.2.3.4", acl1.AddressMatchList.Elements[0].IPAddressOrACLName)

	acl2 := cfg.GetACL("test2")
	require.NotNil(t, acl2)
	require.Equal(t, "test2", acl2.Name)
	require.Len(t, acl2.AddressMatchList.Elements, 1)
	require.Equal(t, "0.0.0.0", acl2.AddressMatchList.Elements[0].IPAddressOrACLName)
}

// Test that the parser correctly handles the include statements in chroot
// environment.
func TestParseIncludesChroot(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	chrootPath, err := sandbox.JoinDir("chroot")
	require.NoError(t, err)

	// Create the parent file with the include statements. The path
	// to the first file is relative. The path to the second file is
	// absolute but to the chroot.
	sandbox.Write("chroot/etc/bind/top-level.conf", `
		include "1.conf";
		include "/etc/bind/2.conf";
	`)
	require.NoError(t, err)

	// Create the first included file.
	sandbox.Write("chroot/etc/bind/1.conf", `
		acl test1 {
			1.2.3.4;
		};
	`)

	// Create the second included file.
	sandbox.Write("chroot/etc/bind/2.conf", `
		acl test2 {
			0.0.0.0;
		};
	`)

	// Parse the parent file without expanding includes. All
	// statements must be includes.
	cfg, err := NewParser().ParseFile("/etc/bind/top-level.conf", chrootPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 2)
	require.NotNil(t, cfg.Statements[0].Include)
	require.Equal(t, "1.conf", cfg.Statements[0].Include.Path)
	require.NotNil(t, cfg.Statements[1].Include)
	require.Equal(t, "/etc/bind/2.conf", cfg.Statements[1].Include.Path)

	// Expand the includes.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The new statements must be ACLs.
	require.Len(t, cfg.Statements, 2)
	acl1 := cfg.GetACL("test1")
	require.NotNil(t, acl1)
	require.Equal(t, "test1", acl1.Name)
	require.Len(t, acl1.AddressMatchList.Elements, 1)
	require.Equal(t, "1.2.3.4", acl1.AddressMatchList.Elements[0].IPAddressOrACLName)

	acl2 := cfg.GetACL("test2")
	require.NotNil(t, acl2)
	require.Equal(t, "test2", acl2.Name)
	require.Len(t, acl2.AddressMatchList.Elements, 1)
	require.Equal(t, "0.0.0.0", acl2.AddressMatchList.Elements[0].IPAddressOrACLName)
}

// Test the case when the configuration file includes itself.
func TestParseIncludeSelf(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	topLevelPath, err := sandbox.Join("top-level.conf")
	require.NoError(t, err)

	sandbox.Write("top-level.conf", `
		include "top-level.conf";
		acl test {
			1.2.3.4;
		};
	`)

	// Parse the configuration file without expanding includes.
	cfg, err := NewParser().ParseFile(topLevelPath, "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Expand the includes.
	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The ACL statement should be correctly parsed. The include statement
	// should exist but not be expanded.
	require.Len(t, cfg.Statements, 2)
	require.NotNil(t, cfg.Statements[0].Include)
	require.Equal(t, "top-level.conf", cfg.Statements[0].Include.Path)
	require.NotNil(t, cfg.Statements[1].ACL)
	require.Equal(t, "test", cfg.Statements[1].ACL.Name)
	require.Len(t, cfg.Statements[1].ACL.AddressMatchList.Elements, 1)
	require.Equal(t, "1.2.3.4", cfg.Statements[1].ACL.AddressMatchList.Elements[0].IPAddressOrACLName)
}

// Test that the parser doesn't fail when parsing the query-source option.
func TestParseQuerySource(t *testing.T) {
	t.Run("IP address only", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				query-source 1.2.3.4;
				query-source-v6 2001:db8::1;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address keyword", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				query-source address 1.2.3.4;
				query-source-v6 address 2001:db8::1;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address keyword with port", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				query-source address 1.2.3.4 port 53;
				query-source-v6 address 2001:db8::1 port 53;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address keyword with port asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				query-source address 1.2.3.4 port *;
				query-source-v6 address 2001:db8::1 port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address with asterisk and port with asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				query-source * port *;
				query-source-v6 * port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address none", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				query-source none;
				query-source-v6 none;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}

// Test that the parser doesn't fail when parsing the notify-source option.
func TestParseNotifySource(t *testing.T) {
	t.Run("IP address only", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				notify-source 1.2.3.4;
				notify-source-v6 2001:db8::1;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address and port", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				notify-source address 1.2.3.4 port 53;
				notify-source-v6 address 2001:db8::1 port 53;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address and port asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				notify-source address 1.2.3.4 port *;
				notify-source-v6 address 2001:db8::1 port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address with asterisk and port with asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				notify-source * port *;
				notify-source-v6 * port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address with asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			options {
				notify-source *;
				notify-source-v6 *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}

// Test that the parser correctly handles the notify-source option in the zone and view blocks.
func TestParseNotifySourceZoneView(t *testing.T) {
	t.Run("zone", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			zone "example.com" {
				notify-source 1.2.3.4;
				notify-source-v6 2001:db8::1;
			};
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
	t.Run("view", func(t *testing.T) {
		cfg, err := NewParser().Parse("", "", strings.NewReader(`
			view "foo" {
				notify-source address 1.2.3.4 port 53;
				notify-source-v6 address 2001:db8::1 port 53;
			};
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}

// Test parsing option with a single switch.
func TestParseBasicOption(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{
			name:  "IPv4 address",
			value: "1.2.3.4",
		},
		{
			name:  "IPv6 address range",
			value: "2001:db8::/32",
		},
		{
			name:  "IPv6 address",
			value: "2001:db8::1",
		},
		{
			name:  "IPv4 address quoted",
			value: `"1.2.3.4"`,
		},
		{
			name:  "IPv6 address range quoted",
			value: `"2001:db8::/32"`,
		},
		{
			name:  "IPv6 address quoted",
			value: `"2001:db8::1"`,
		},
		{
			name:  "string",
			value: `"bar"`,
		},
		{
			name:  "ident",
			value: "bar-abc",
		},
		{
			name:  "number",
			value: "123",
		},
		{
			name:  "asterisk",
			value: "*",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfgText := fmt.Sprintf(`
				options {
					foo %s;
				}
			`, testCase.value)
			cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.Len(t, cfg.Statements, 1)
			require.NotNil(t, cfg.Statements[0].Options)
			require.Len(t, cfg.Statements[0].Options.Clauses, 1)
			require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option)
			//			require.Equal(t, "foo", cfg.Statements[0].Options.Clauses[0].Option.Identifier)
			//			require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Switches, 1)
			//			require.Equal(t, strings.Trim(testCase.value, `"`), cfg.Statements[0].Options.Clauses[0].Option.Switches[0])
			//			require.Nil(t, cfg.Statements[0].Options.Clauses[0].Option.Contents)
			require.Empty(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions)
		})
	}
}

// Test parsing option including curly brackets and no suboptions.
func TestParseOptionWithCurlyBrackets(t *testing.T) {
	cfgText := `
		options {
			foo 123	abc { city "paris"; };
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Options)
	require.Len(t, cfg.Statements[0].Options.Clauses, 1)
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option)
	require.Equal(t, "foo", cfg.Statements[0].Options.Clauses[0].Option.Identifier)
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Switches, 2)
	require.Equal(t, "123", cfg.Statements[0].Options.Clauses[0].Option.Switches[0].GetStringValue())
	require.Equal(t, "abc", cfg.Statements[0].Options.Clauses[0].Option.Switches[1].GetStringValue())
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option.Contents)
	require.Empty(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions)
}

// Test parsing option with suboptions.
func TestParseOptionWithSuboptions(t *testing.T) {
	cfgText := `
		options {
			foo 123	{ city "paris"; } except-from { "abc"; } update 100;
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Options)
	require.Len(t, cfg.Statements[0].Options.Clauses, 1)
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option)
	require.Equal(t, "foo", cfg.Statements[0].Options.Clauses[0].Option.Identifier)
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Switches, 1)
	require.Equal(t, "123", cfg.Statements[0].Options.Clauses[0].Option.Switches[0].GetStringValue())
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option.Contents)
	require.Len(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions, 2)
	require.Equal(t, "except-from", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Identifier)
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option.Suboptions[0].Contents)
	require.Equal(t, "update", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[1].Identifier)
	require.Equal(t, "100", cfg.Statements[0].Options.Clauses[0].Option.Suboptions[1].Switches[0].GetStringValue())
}

// Test parsing ACL with negated key.
func TestParseACLWithNegatedKey(t *testing.T) {
	cfgText := `
		acl "trusted-networks" {
			!key guest-key;
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].ACL)
	require.Equal(t, "trusted-networks", cfg.Statements[0].ACL.Name)
	require.NotNil(t, cfg.Statements[0].ACL.AddressMatchList)
	require.Len(t, cfg.Statements[0].ACL.AddressMatchList.Elements, 1)
	require.True(t, cfg.Statements[0].ACL.AddressMatchList.Elements[0].Negation)
	require.Equal(t, "guest-key", cfg.Statements[0].ACL.AddressMatchList.Elements[0].KeyID)
}

// Test parsing ACL with a key.
func TestParseACLWithKey(t *testing.T) {
	cfgText := `
		acl "guest-networks" {
			key "guest-key";
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].ACL)
	require.Equal(t, "guest-networks", cfg.Statements[0].ACL.Name)
	require.NotNil(t, cfg.Statements[0].ACL.AddressMatchList)
	require.Len(t, cfg.Statements[0].ACL.AddressMatchList.Elements, 1)
	require.False(t, cfg.Statements[0].ACL.AddressMatchList.Elements[0].Negation)
	require.Equal(t, "guest-key", cfg.Statements[0].ACL.AddressMatchList.Elements[0].KeyID)
}

// Test parsing ACL with an unquoted ACL name.
func TestParseACLWithUnquotedACLName(t *testing.T) {
	cfgText := `
		acl "trusted-networks" {
			localnets;
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].ACL)
	require.Equal(t, "trusted-networks", cfg.Statements[0].ACL.Name)
	require.NotNil(t, cfg.Statements[0].ACL.AddressMatchList)
	require.Len(t, cfg.Statements[0].ACL.AddressMatchList.Elements, 1)
	require.Equal(t, "localnets", cfg.Statements[0].ACL.AddressMatchList.Elements[0].IPAddressOrACLName)
}

// Test parsing ACL with a quoted ACL name.
func TestParseACLWithQuotedACLName(t *testing.T) {
	cfgText := `
		acl "trusted-networks" {
			"localhosts";
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].ACL)
	require.Equal(t, "trusted-networks", cfg.Statements[0].ACL.Name)
	require.NotNil(t, cfg.Statements[0].ACL.AddressMatchList)
	require.Len(t, cfg.Statements[0].ACL.AddressMatchList.Elements, 1)
	require.Equal(t, "localhosts", cfg.Statements[0].ACL.AddressMatchList.Elements[0].IPAddressOrACLName)
}

// Test parsing ACL with a quoted IPv4 address.
func TestParseACLWithQuotedIPv4Address(t *testing.T) {
	cfgText := `
		acl "trusted-networks" {
			"10.0.0.1";
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].ACL)
	require.Equal(t, "trusted-networks", cfg.Statements[0].ACL.Name)
	require.NotNil(t, cfg.Statements[0].ACL.AddressMatchList)
	require.Len(t, cfg.Statements[0].ACL.AddressMatchList.Elements, 1)
	require.Equal(t, "10.0.0.1", cfg.Statements[0].ACL.AddressMatchList.Elements[0].IPAddressOrACLName)
}

// Test parsing the dyndb statement.
func TestParseDynDB(t *testing.T) {
	cfgText := `
		dyndb "ipa" "/usr/lib64/bind/ldap.so" {
			uri "ldapi://%2fvar%2frun%2fslapd-EXAMPLE-CA.socket";
			base "cn=dns,dc=example,dc=ca";
			server_id "host.example.ca";
			auth_method "sasl";
			sasl_mech "EXTERNAL";
			krb5_keytab "FILE:/etc/named.keytab";
			library "ldap.so";
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Option)
	require.Equal(t, "dyndb", cfg.Statements[0].Option.Identifier)
	require.Len(t, cfg.Statements[0].Option.Switches, 2)
	require.Equal(t, "ipa", cfg.Statements[0].Option.Switches[0].GetStringValue())
	require.Equal(t, "/usr/lib64/bind/ldap.so", cfg.Statements[0].Option.Switches[1].GetStringValue())
	require.NotNil(t, cfg.Statements[0].Option.Contents)
	require.Empty(t, cfg.Statements[0].Option.Suboptions)
}

// Test parsing the deny-answer-aliases option.
func TestParseDenyAnswerAliases(t *testing.T) {
	cfgText := `
		options {
			deny-answer-aliases {
				name "secure.example.com";
				exclude { internal-nets; };
			} except-from { "trusted.example.net"; };
		}
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(cfgText))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 1)
	require.NotNil(t, cfg.Statements[0].Options)
	require.Len(t, cfg.Statements[0].Options.Clauses, 1)
	require.NotNil(t, cfg.Statements[0].Options.Clauses[0].Option)
}

// A benchmark that measures the performance of the @stork:no-parse directive.
// It creates a set of zones and runs two independent checks. First, how long
// it takes to parse the zones. Second, how long it takes to process the config
// when @stork:no-parse elides the zones.
//
// For 10000 we've got the following results:
//
// BenchmarkNoParseZones/No_parse-12         	       2	 774297854 ns/op	 1344496 B/op	     122 allocs/op
// BenchmarkNoParseZones/NoParseGlobal-12    	       2	 717727792 ns/op	 1306192 B/op	     106 allocs/op
// BenchmarkNoParseZones/Parse-12            	       1	6126068000 ns/op	2984310664 B/op	10009899 allocs/op
// PASS
// ok  	isc.org/stork/daemoncfg/bind9	12.047s
//
// Clearly, skipping the zones during parsing significantly improves the
// configuration file parsing performance.
func BenchmarkNoParseZones(b *testing.B) {
	zones := testutil.GenerateRandomZones(10000)
	zoneTemplate := `
		zone "%s" {
			type master;
			allow-transfer port 853 { any; };
			file "/etc/bind/db.%s";
		};
	`
	builder := strings.Builder{}
	for _, zone := range zones {
		zoneText := fmt.Sprintf(zoneTemplate, zone.Name, zone.Name)
		builder.WriteString(zoneText)
	}
	parser := NewParser()
	require.NotNil(b, parser)

	b.Run("No parse", func(b *testing.B) {
		// Surround the zones with the @stork:no-parse:begin/end directscope
		noParseBuilder := strings.Builder{}
		noParseBuilder.WriteString("//@stork:no-parse:scope\n")
		noParseBuilder.WriteString(builder.String())
		noParseBuilder.WriteString("//@stork:no-parse:end\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.Parse("", "", strings.NewReader(noParseBuilder.String()))
			require.NoError(b, err)
		}
	})

	b.Run("NoParseGlobal", func(b *testing.B) {
		// Precede the zones with the @stork:no-parse:global directive.
		noParseBuilder := strings.Builder{}
		noParseBuilder.WriteString("//@stork:no-parse:global\n")
		noParseBuilder.WriteString(builder.String())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.Parse("", "", strings.NewReader(noParseBuilder.String()))
			require.NoError(b, err)
		}
	})

	b.Run("Parse", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.Parse("", "", strings.NewReader(builder.String()))
			require.NoError(b, err)
		}
	})
}

// Test parsing the inet clause of the controls statement.
func TestParseControlsInet(t *testing.T) {
	tests := []struct {
		name             string
		source           string
		expectedAddress  string
		expectedPort     *string
		expectedAllow    []string
		expectedKeys     []string
		expectedReadOnly *bool
	}{
		{
			name:            "address only",
			source:          "inet 127.0.0.1;",
			expectedAddress: "127.0.0.1",
		},
		{
			name:            "address and port",
			source:          "inet 127.0.0.1 port 53;",
			expectedAddress: "127.0.0.1",
			expectedPort:    storkutil.Ptr("53"),
		},
		{
			name:            "address and port and allow",
			source:          "inet 127.0.0.1 port 53 allow { 1.2.3.4; };",
			expectedAddress: "127.0.0.1",
			expectedPort:    storkutil.Ptr("53"),
			expectedAllow:   []string{"1.2.3.4"},
		},
		{
			name:            "address and port and allow and keys",
			source:          "inet 127.0.0.1 port 53 allow { 1.2.3.4; } keys { rndc-key; };",
			expectedAddress: "127.0.0.1",
			expectedPort:    storkutil.Ptr("53"),
			expectedAllow:   []string{"1.2.3.4"},
			expectedKeys:    []string{"rndc-key"},
		},
		{
			name:             "address and port and allow and keys and read-only",
			source:           "inet 127.0.0.1 port 53 allow { 1.2.3.4; } keys { rndc-key; } read-only true;",
			expectedAddress:  "127.0.0.1",
			expectedPort:     storkutil.Ptr("53"),
			expectedAllow:    []string{"1.2.3.4"},
			expectedKeys:     []string{"rndc-key"},
			expectedReadOnly: storkutil.Ptr(true),
		},
		{
			name:             "address and read-only false",
			source:           "inet 127.0.0.1 read-only false;",
			expectedAddress:  "127.0.0.1",
			expectedReadOnly: storkutil.Ptr(false),
		},
		{
			name:             "address and read-only yes",
			source:           "inet 127.0.0.1 read-only yes;",
			expectedAddress:  "127.0.0.1",
			expectedReadOnly: storkutil.Ptr(true),
		},
		{
			name:             "address and read-only no",
			source:           "inet 127.0.0.1 read-only no;",
			expectedAddress:  "127.0.0.1",
			expectedReadOnly: storkutil.Ptr(false),
		},
		{
			name:             "address and read-only 1",
			source:           "inet 127.0.0.1 read-only 1;",
			expectedAddress:  "127.0.0.1",
			expectedReadOnly: storkutil.Ptr(true),
		},
		{
			name:             "address and read-only 0",
			source:           "inet 127.0.0.1 read-only 0;",
			expectedAddress:  "127.0.0.1",
			expectedReadOnly: storkutil.Ptr(false),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controlsSource := fmt.Sprintf("controls { %s };", test.source)
			parser := NewParser()
			cfg, err := parser.Parse("", "", strings.NewReader(controlsSource))
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.Len(t, cfg.Statements, 1)
			require.NotNil(t, cfg.Statements[0].Controls)
			require.Len(t, cfg.Statements[0].Controls.Clauses, 1)
			require.NotNil(t, cfg.Statements[0].Controls.Clauses[0].InetClause)
			require.Equal(t, test.expectedAddress, cfg.Statements[0].Controls.Clauses[0].InetClause.Address)
			if test.expectedPort != nil {
				require.NotNil(t, cfg.Statements[0].Controls.Clauses[0].InetClause.Port)
				require.Equal(t, *test.expectedPort, *cfg.Statements[0].Controls.Clauses[0].InetClause.Port)
			}
			if test.expectedAllow != nil {
				for _, element := range cfg.Statements[0].Controls.Clauses[0].InetClause.Allow.Elements {
					require.Contains(t, test.expectedAllow, element.IPAddressOrACLName)
				}
			}
			if test.expectedKeys != nil {
				require.ElementsMatch(t, test.expectedKeys, cfg.Statements[0].Controls.Clauses[0].InetClause.Keys.KeyNames)
			}
			if test.expectedReadOnly != nil {
				require.NotNil(t, cfg.Statements[0].Controls.Clauses[0].InetClause.ReadOnly)
				require.EqualValues(t, *test.expectedReadOnly, *cfg.Statements[0].Controls.Clauses[0].InetClause.ReadOnly)
			}
		})
	}
}

// Test parsing the unix clause of the controls statement.
func TestParseControlsUnix(t *testing.T) {
	tests := []struct {
		name             string
		source           string
		expectedPath     string
		expectedPerm     int64
		expectedOwner    int64
		expectedGroup    int64
		expectedKeys     []string
		expectedReadOnly *bool
	}{
		{
			name:          "path and perm and owner and group",
			source:        `unix "/var/run/rndc.sock" perm 0600 owner 25 group 26;`,
			expectedPath:  "/var/run/rndc.sock",
			expectedPerm:  0o600,
			expectedOwner: 25,
			expectedGroup: 26,
		},
		{
			name:          "path and perm and owner and group and keys",
			source:        `unix "/var/run/rndc.sock" perm 0600 owner 25 group 26 keys { rndc-key; };`,
			expectedPath:  "/var/run/rndc.sock",
			expectedPerm:  0o600,
			expectedOwner: 25,
			expectedGroup: 26,
			expectedKeys:  []string{"rndc-key"},
		},
		{
			name:             "path and perm and owner and group and keys and read-only",
			source:           `unix "/var/run/rndc.sock" perm 0600 owner 25 group 26 keys { rndc-key; } read-only true;`,
			expectedPath:     "/var/run/rndc.sock",
			expectedPerm:     0o600,
			expectedOwner:    25,
			expectedGroup:    26,
			expectedKeys:     []string{"rndc-key"},
			expectedReadOnly: storkutil.Ptr(true),
		},
		{
			name:             "path and perm and owner and group and read-only false",
			source:           `unix "/var/run/rndc.sock" perm 0600 owner 25 group 26 read-only false;`,
			expectedPath:     "/var/run/rndc.sock",
			expectedPerm:     0o600,
			expectedOwner:    25,
			expectedGroup:    26,
			expectedReadOnly: storkutil.Ptr(false),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controlsSource := fmt.Sprintf("controls { %s };", test.source)
			parser := NewParser()
			cfg, err := parser.Parse("", "", strings.NewReader(controlsSource))
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.Len(t, cfg.Statements, 1)
			require.NotNil(t, cfg.Statements[0].Controls)
			require.Len(t, cfg.Statements[0].Controls.Clauses, 1)
			require.NotNil(t, cfg.Statements[0].Controls.Clauses[0].UnixClause)
			require.Equal(t, test.expectedPath, cfg.Statements[0].Controls.Clauses[0].UnixClause.Path)
			require.EqualValues(t, test.expectedPerm, cfg.Statements[0].Controls.Clauses[0].UnixClause.Perm)
			require.EqualValues(t, test.expectedOwner, cfg.Statements[0].Controls.Clauses[0].UnixClause.Owner)
			require.EqualValues(t, test.expectedGroup, cfg.Statements[0].Controls.Clauses[0].UnixClause.Group)
			if test.expectedKeys != nil {
				require.ElementsMatch(t, test.expectedKeys, cfg.Statements[0].Controls.Clauses[0].UnixClause.Keys.KeyNames)
			}
			if test.expectedReadOnly != nil {
				require.NotNil(t, cfg.Statements[0].Controls.Clauses[0].UnixClause.ReadOnly)
				require.EqualValues(t, *test.expectedReadOnly, *cfg.Statements[0].Controls.Clauses[0].UnixClause.ReadOnly)
			}
		})
	}
}
