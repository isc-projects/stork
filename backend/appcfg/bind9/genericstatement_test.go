package bind9config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that a real BIND 9 option parsed as a generic clause contents
// is formatted correctly.
func TestGenericClauseContentsGetFormatterOutput(t *testing.T) {
	text := `
		cert-file "/private/certs/domain.rsa.pem" ;
		key-file "/private/certs/domain.rsa.key" ;
		dhparam-file "/private/certs/dhparams4096.pem" ;
		ciphers "HIGH:!kRSA:!aNULL:!eNULL:!RC4:!3DES:!MD5:!EXP:!PSK:!SRP:!DSS:!SHA1:!SHA256:!SHA384" ;
		prefer-server-ciphers yes ;
		session-tickets no ;
	`
	contents := &GenericClauseContents{
		tokens: strings.Fields(text),
	}
	clause := contents.getFormattedOutput(nil)
	require.NotNil(t, clause)
	cfgScopeEq(t, text, clause)
}

// Test that a multiple scopes in a generic clause contents are formatted correctly.
func TestGenericClauseContentsGetFormatterOutputMultipleScopes(t *testing.T) {
	text := "foo { bar ; } baz { { cafe ; } abc bac { cab ; } xyz ; avg ; } ;"
	contents := &GenericClauseContents{
		tokens: strings.Fields(text),
	}
	clause := contents.getFormattedOutput(nil)
	require.NotNil(t, clause)
	cfgScopeEq(t, text, clause)
}

// Test that serializing a generic clause contents with nil values does not panic.
func TestGenericClauseContentsFormatNilValues(t *testing.T) {
	contents := &GenericClauseContents{}
	require.NotPanics(t, func() { contents.getFormattedOutput(nil) })
}
