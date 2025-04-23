package bind9config

import (
	"fmt"
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

	require.Len(t, cfg.Statements, 11)

	require.NotNil(t, cfg.Statements[0].Key)
	require.Equal(t, "trusted-key", cfg.Statements[0].Key.Name)
	require.Len(t, cfg.Statements[0].Key.Clauses, 2)
	require.Equal(t, "hmac-sha256", cfg.Statements[0].Key.Clauses[0].Algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", cfg.Statements[0].Key.Clauses[1].Secret)

	require.NotNil(t, cfg.Statements[1].Key)
	require.Equal(t, "guest-key", cfg.Statements[1].Key.Name)
	require.Len(t, cfg.Statements[1].Key.Clauses, 2)
	require.Equal(t, "hmac-sha256", cfg.Statements[1].Key.Clauses[0].Algorithm)
	require.Equal(t, "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=", cfg.Statements[1].Key.Clauses[1].Secret)

	require.NotNil(t, cfg.Statements[2].ACL)
	require.Equal(t, "trusted-networks", cfg.Statements[2].ACL.Name)
	require.Len(t, cfg.Statements[2].ACL.AdressMatchList.Elements, 8)

	require.NotNil(t, cfg.Statements[3].ACL)
	require.Equal(t, "guest-networks", cfg.Statements[3].ACL.Name)
	require.Len(t, cfg.Statements[3].ACL.AdressMatchList.Elements, 6)

	require.NotNil(t, cfg.Statements[4].UnnamedStatement)
	require.Equal(t, "controls", cfg.Statements[4].UnnamedStatement.Identifier)

	require.NotNil(t, cfg.Statements[5].UnnamedStatement)
	require.Equal(t, "statistics-channels", cfg.Statements[5].UnnamedStatement.Identifier)

	require.NotNil(t, cfg.Statements[6].Options)
	require.Len(t, cfg.Statements[6].Options.Clauses, 8)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[0].UnnamedClause)
	require.Equal(t, "allow-query", cfg.Statements[6].Options.Clauses[0].UnnamedClause.Identifier)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[1].AllowTransfer)
	require.Nil(t, cfg.Statements[6].Options.Clauses[1].AllowTransfer.Port)
	require.Nil(t, cfg.Statements[6].Options.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[1].AllowTransfer.AdressMatchList)
	require.Len(t, cfg.Statements[6].Options.Clauses[1].AllowTransfer.AdressMatchList.Elements, 1)
	require.Equal(t, "any", cfg.Statements[6].Options.Clauses[1].AllowTransfer.AdressMatchList.Elements[0].ACLName)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[2].Option)
	require.Equal(t, "also-notify", cfg.Statements[6].Options.Clauses[2].Option.Identifier)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[3].Option)
	require.Equal(t, "check-names", cfg.Statements[6].Options.Clauses[3].Option.Identifier)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[4].Option)
	require.Equal(t, "dnssec-validation", cfg.Statements[6].Options.Clauses[4].Option.Identifier)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[5].Option)
	require.Equal(t, "recursion", cfg.Statements[6].Options.Clauses[5].Option.Identifier)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[6].ListenOn)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[6].ListenOn.Port)
	require.EqualValues(t, 8553, *cfg.Statements[6].Options.Clauses[6].ListenOn.Port)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[6].ListenOn.Proxy)
	require.Equal(t, "plain", *cfg.Statements[6].Options.Clauses[6].ListenOn.Proxy)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[6].ListenOn.TLS)
	require.Equal(t, "ephemeral", *cfg.Statements[6].Options.Clauses[6].ListenOn.TLS)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[6].ListenOn.HTTP)
	require.Equal(t, "myserver", *cfg.Statements[6].Options.Clauses[6].ListenOn.HTTP)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[6].ListenOn.AdressMatchList)
	require.Len(t, cfg.Statements[6].Options.Clauses[6].ListenOn.AdressMatchList.Elements, 1)
	require.Equal(t, "127.0.0.1", cfg.Statements[6].Options.Clauses[6].ListenOn.AdressMatchList.Elements[0].IPAddress)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[7].ListenOnV6)
	require.NotNil(t, cfg.Statements[6].Options.Clauses[7].ListenOnV6.AdressMatchList)
	require.Len(t, cfg.Statements[6].Options.Clauses[7].ListenOnV6.AdressMatchList.Elements, 1)
	require.Equal(t, "::1", cfg.Statements[6].Options.Clauses[7].ListenOnV6.AdressMatchList.Elements[0].IPAddress)
	require.NotNil(t, cfg.Statements[7].View)
	require.Equal(t, "trusted", cfg.Statements[7].View.Name)
	require.Len(t, cfg.Statements[7].View.Clauses, 5)
	require.NotNil(t, cfg.Statements[7].View.Clauses[0].MatchClients)
	require.Len(t, cfg.Statements[7].View.Clauses[0].MatchClients.AdressMatchList.Elements, 1)
	require.NotNil(t, cfg.Statements[7].View.Clauses[1].AllowTransfer)
	require.EqualValues(t, 853, *cfg.Statements[7].View.Clauses[1].AllowTransfer.Port)
	require.Equal(t, "tls", *cfg.Statements[7].View.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, cfg.Statements[7].View.Clauses[1].AllowTransfer.AdressMatchList)
	require.NotNil(t, cfg.Statements[7].View.Clauses[2].Option)
	require.Equal(t, "recursion", cfg.Statements[7].View.Clauses[2].Option.Identifier)
	require.NotNil(t, cfg.Statements[7].View.Clauses[3].Zone)
	require.Equal(t, "bind9.example.com", cfg.Statements[7].View.Clauses[3].Zone.Name)
	require.NotNil(t, cfg.Statements[7].View.Clauses[4].Zone)
	require.Equal(t, "pdns.example.com", cfg.Statements[7].View.Clauses[4].Zone.Name)

	require.NotNil(t, cfg.Statements[8].View)
	require.Equal(t, "guest", cfg.Statements[8].View.Name)
	require.Len(t, cfg.Statements[8].View.Clauses, 3)
	require.NotNil(t, cfg.Statements[8].View.Clauses[0].MatchClients)
	require.Len(t, cfg.Statements[8].View.Clauses[0].MatchClients.AdressMatchList.Elements, 1)
	require.NotNil(t, cfg.Statements[8].View.Clauses[1].AllowTransfer)
	require.Nil(t, cfg.Statements[8].View.Clauses[1].AllowTransfer.Port)
	require.Equal(t, "tls", *cfg.Statements[8].View.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, cfg.Statements[8].View.Clauses[1].AllowTransfer.AdressMatchList)
	require.NotNil(t, cfg.Statements[8].View.Clauses[2].Zone)
	require.Equal(t, "bind9.example.org", cfg.Statements[8].View.Clauses[2].Zone.Name)

	require.NotNil(t, cfg.Statements[9].Zone)
	require.Equal(t, "nsd.example.com", cfg.Statements[9].Zone.Name)
	require.Equal(t, "IN", cfg.Statements[9].Zone.Class)
	require.NotNil(t, cfg.Statements[9].Zone.Clauses)
	require.Len(t, cfg.Statements[9].Zone.Clauses, 3)
	require.NotNil(t, cfg.Statements[9].Zone.Clauses[0].Option)
	require.Equal(t, "type", cfg.Statements[9].Zone.Clauses[0].Option.Identifier)
	require.Len(t, cfg.Statements[9].Zone.Clauses[0].Option.Switches, 1)
	require.Equal(t, "master", cfg.Statements[9].Zone.Clauses[0].Option.Switches[0])
	require.NotNil(t, cfg.Statements[9].Zone.Clauses[1].AllowTransfer)
	require.EqualValues(t, 853, *cfg.Statements[9].Zone.Clauses[1].AllowTransfer.Port)
	require.Nil(t, cfg.Statements[9].Zone.Clauses[1].AllowTransfer.Transport)
	require.NotNil(t, cfg.Statements[9].Zone.Clauses[1].AllowTransfer.AdressMatchList)
	require.NotNil(t, cfg.Statements[9].Zone.Clauses[2].Option)
	require.Equal(t, "file", cfg.Statements[9].Zone.Clauses[2].Option.Identifier)
	require.Len(t, cfg.Statements[9].Zone.Clauses[2].Option.Switches, 1)
	require.Equal(t, "/etc/bind/db.nsd.example.com", cfg.Statements[9].Zone.Clauses[2].Option.Switches[0])

	require.NotNil(t, cfg.Statements[10].UnnamedStatement)
	require.Equal(t, "logging", cfg.Statements[10].UnnamedStatement.Identifier)
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
	require.Len(t, acl1.AdressMatchList.Elements, 1)
	require.Equal(t, "1.2.3.4", acl1.AdressMatchList.Elements[0].IPAddress)

	acl2 := cfg.GetACL("test2")
	require.NotNil(t, acl2)
	require.Equal(t, "test2", acl2.Name)
	require.Len(t, acl2.AdressMatchList.Elements, 1)
	require.Equal(t, "0.0.0.0", acl2.AdressMatchList.Elements[0].IPAddress)
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
	require.Len(t, cfg.Statements[1].ACL.AdressMatchList.Elements, 1)
	require.Equal(t, "1.2.3.4", cfg.Statements[1].ACL.AdressMatchList.Elements[0].IPAddress)
}
