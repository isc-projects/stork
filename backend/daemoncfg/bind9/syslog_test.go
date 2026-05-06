package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the syslog clause is formatted correctly when facility is specified
// in quoted form.
func TestSyslogFormatWithQuotedFacility(t *testing.T) {
	syslog := &Syslog{
		Facility: &String{
			Quoted: storkutil.Ptr("local7"),
		},
	}
	output := syslog.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `syslog "local7";`, output)
}

// Test that the syslog clause is formatted correctly when facility is specified
// in unquoted form.
func TestSyslogFormatWithUnquotedFacility(t *testing.T) {
	syslog := &Syslog{
		Facility: &String{
			Unquoted: storkutil.Ptr("local7"),
		},
	}
	output := syslog.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `syslog local7;`, output)
}

// Test that the syslog clause is formatted correctly when facility is not specified.
func TestSyslogFormatWithoutFacility(t *testing.T) {
	syslog := &Syslog{}
	output := syslog.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `syslog;`, output)
}
