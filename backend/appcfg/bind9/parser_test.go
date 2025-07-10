package bind9config

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test successfully parsing the named configuration file.
func TestParseFile(t *testing.T) {
	// Parse the named configuration file without expanding includes.
	cfg, err := NewParser().ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Expand included files.
	cfg, err = cfg.Expand("testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)

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
	require.NotNil(t, statement.UnnamedStatement)
	require.Equal(t, "controls", statement.UnnamedStatement.Identifier)

	statement, _ = next()
	require.NotNil(t, statement.UnnamedStatement)
	require.Equal(t, "statistics-channels", statement.UnnamedStatement.Identifier)

	statement, _ = next()
	require.NotNil(t, statement.NamedStatement)
	require.Equal(t, "tls", statement.NamedStatement.Identifier)
	require.Equal(t, "domain.name", statement.NamedStatement.Name)

	statement, _ = next()
	require.NotNil(t, statement.Options)
	require.Len(t, statement.Options.Clauses, 9)
	require.NotNil(t, statement.Options.Clauses[0].UnnamedClause)
	require.Equal(t, "allow-query", statement.Options.Clauses[0].UnnamedClause.Identifier)
	require.NotNil(t, statement.Options.Clauses[1].AllowTransfer)
	require.Nil(t, statement.Options.Clauses[1].AllowTransfer.Port)
	require.Nil(t, statement.Options.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.Options.Clauses[1].AllowTransfer.AddressMatchList)
	require.Len(t, statement.Options.Clauses[1].AllowTransfer.AddressMatchList.Elements, 1)
	require.Equal(t, "any", statement.Options.Clauses[1].AllowTransfer.AddressMatchList.Elements[0].ACLName)
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
	require.Equal(t, "127.0.0.1", statement.Options.Clauses[6].ListenOn.AddressMatchList.Elements[0].IPAddress)
	require.NotNil(t, statement.Options.Clauses[7].ListenOnV6)
	require.NotNil(t, statement.Options.Clauses[7].ListenOnV6.AddressMatchList)
	require.Len(t, statement.Options.Clauses[7].ListenOnV6.AddressMatchList.Elements, 1)
	require.Equal(t, "::1", statement.Options.Clauses[7].ListenOnV6.AddressMatchList.Elements[0].IPAddress)
	require.NotNil(t, statement.Options.Clauses[8].ResponsePolicy)
	require.Len(t, statement.Options.Clauses[8].ResponsePolicy.Zones, 2)
	require.Equal(t, "rpz.example.com", statement.Options.Clauses[8].ResponsePolicy.Zones[0].Zone)
	require.Len(t, statement.Options.Clauses[8].ResponsePolicy.Zones[0].Switches, 6)
	require.Equal(t, "db.local", statement.Options.Clauses[8].ResponsePolicy.Zones[1].Zone)
	require.Empty(t, statement.Options.Clauses[8].ResponsePolicy.Zones[1].Switches)
	require.Len(t, statement.Options.Clauses[8].ResponsePolicy.Switches, 2)
	require.Equal(t, "update-interval", statement.Options.Clauses[8].ResponsePolicy.Switches[0])
	require.Equal(t, "100", statement.Options.Clauses[8].ResponsePolicy.Switches[1])

	statement, _ = next()
	require.NotNil(t, statement.View)
	require.Equal(t, "trusted", statement.View.Name)
	require.Len(t, statement.View.Clauses, 7)
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

	statement, _ = next()
	require.NotNil(t, statement.View)
	require.Equal(t, "guest", statement.View.Name)
	require.Len(t, statement.View.Clauses, 3)
	require.NotNil(t, statement.View.Clauses[0].MatchClients)
	require.Len(t, statement.View.Clauses[0].MatchClients.AddressMatchList.Elements, 1)
	require.NotNil(t, statement.View.Clauses[1].AllowTransfer)
	require.Nil(t, statement.View.Clauses[1].AllowTransfer.Port)
	require.Equal(t, "tls", *statement.View.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.View.Clauses[1].AllowTransfer.AddressMatchList)
	require.NotNil(t, statement.View.Clauses[2].Zone)
	require.Equal(t, "bind9.example.org", statement.View.Clauses[2].Zone.Name)

	statement, _ = next()
	require.NotNil(t, statement.Zone)
	require.Equal(t, "nsd.example.com", statement.Zone.Name)
	require.Equal(t, "IN", statement.Zone.Class)
	require.NotNil(t, statement.Zone.Clauses)
	require.Len(t, statement.Zone.Clauses, 3)
	require.NotNil(t, statement.Zone.Clauses[0].Option)
	require.Equal(t, "type", statement.Zone.Clauses[0].Option.Identifier)
	require.Len(t, statement.Zone.Clauses[0].Option.SwitchesBeforeCurlyBrackets, 1)
	require.Equal(t, "master", statement.Zone.Clauses[0].Option.SwitchesBeforeCurlyBrackets[0])
	require.NotNil(t, statement.Zone.Clauses[1].AllowTransfer)
	require.EqualValues(t, 853, *statement.Zone.Clauses[1].AllowTransfer.Port)
	require.Nil(t, statement.Zone.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, statement.Zone.Clauses[1].AllowTransfer.AddressMatchList)
	require.NotNil(t, statement.Zone.Clauses[2].Option)
	require.Equal(t, "file", statement.Zone.Clauses[2].Option.Identifier)
	require.Len(t, statement.Zone.Clauses[2].Option.SwitchesBeforeCurlyBrackets, 1)
	require.Equal(t, "/etc/bind/db.nsd.example.com", statement.Zone.Clauses[2].Option.SwitchesBeforeCurlyBrackets[0])

	statement, _ = next()
	require.NotNil(t, statement.UnnamedStatement)
	require.Equal(t, "logging", statement.UnnamedStatement.Identifier)
}

// Test that an attempt to parse a non-existent file returns an error.
func TestParseFileError(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/non-existent.conf")
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
	cfg, err := NewParser().ParseFile(topLevelPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Statements, 2)
	require.NotNil(t, cfg.Statements[0].Include)
	require.Equal(t, "1.conf", cfg.Statements[0].Include.Path)
	require.NotNil(t, cfg.Statements[1].Include)
	require.Equal(t, includedPath, cfg.Statements[1].Include.Path)

	// Expand the includes.
	cfg, err = cfg.Expand(sandbox.BasePath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The new statements must be ACLs.
	require.Len(t, cfg.Statements, 2)
	acl1 := cfg.GetACL("test1")
	require.NotNil(t, acl1)
	require.Equal(t, "test1", acl1.Name)
	require.Len(t, acl1.AddressMatchList.Elements, 1)
	require.Equal(t, "1.2.3.4", acl1.AddressMatchList.Elements[0].IPAddress)

	acl2 := cfg.GetACL("test2")
	require.NotNil(t, acl2)
	require.Equal(t, "test2", acl2.Name)
	require.Len(t, acl2.AddressMatchList.Elements, 1)
	require.Equal(t, "0.0.0.0", acl2.AddressMatchList.Elements[0].IPAddress)
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
	cfg, err := NewParser().ParseFile(topLevelPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Expand the includes.
	cfg, err = cfg.Expand(sandbox.BasePath)
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
	require.Equal(t, "1.2.3.4", cfg.Statements[1].ACL.AddressMatchList.Elements[0].IPAddress)
}

// Test that the parser doesn't fail when parsing the query-source option.
func TestParseQuerySource(t *testing.T) {
	t.Run("IP address only", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				query-source 1.2.3.4;
				query-source-v6 2001:db8::1;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address keyword", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				query-source address 1.2.3.4;
				query-source-v6 address 2001:db8::1;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address keyword with port", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				query-source address 1.2.3.4 port 53;
				query-source-v6 address 2001:db8::1 port 53;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address keyword with port asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				query-source address 1.2.3.4 port *;
				query-source-v6 address 2001:db8::1 port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address with asterisk and port with asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				query-source * port *;
				query-source-v6 * port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address none", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
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
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				notify-source 1.2.3.4;
				notify-source-v6 2001:db8::1;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address and port", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				notify-source address 1.2.3.4 port 53;
				notify-source-v6 address 2001:db8::1 port 53;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address and port asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				notify-source address 1.2.3.4 port *;
				notify-source-v6 address 2001:db8::1 port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address with asterisk and port with asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				notify-source * port *;
				notify-source-v6 * port *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("address with asterisk", func(t *testing.T) {
		cfg, err := NewParser().Parse(" ", strings.NewReader(`
			options {
				notify-source *;
				notify-source-v6 *;
			}
		`))
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}
