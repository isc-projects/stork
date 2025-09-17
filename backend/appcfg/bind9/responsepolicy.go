package bind9config

import "strings"

var (
	_ formattedElement = (*ResponsePolicy)(nil)
	_ formattedElement = (*ResponsePolicyZone)(nil)
)

// ResponsePolicy is the clause specifying the response policy zones.
// See https://bind9.readthedocs.io/en/latest/reference.html#namedconf-statement-response-policy
type ResponsePolicy struct {
	Zones    []*ResponsePolicyZone `parser:"'{' ( @@ ';'+ )* '}'"`
	Switches []string              `parser:"( @String | @Ident )*"`
}

// Checks if the zone is a RPZ by running a case-insensitive comparison
// of the zone name with the zone names in the response-policy clause.
func (rp *ResponsePolicy) IsRPZ(zoneName string) bool {
	for _, zone := range rp.Zones {
		if strings.EqualFold(zone.Zone, zoneName) {
			return true
		}
	}
	return false
}

// Returns the serialized BIND 9 configuration for the response-policy statement.
func (rp *ResponsePolicy) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause("response-policy")
	scope := clause.addScope()
	for _, zone := range rp.Zones {
		scope.add(zone.getFormattedOutput(filter))
	}
	for _, sw := range rp.Switches {
		clause.addToken(sw)
	}
	return clause
}

// ResponsePolicyZone is a single response policy zone entry.
type ResponsePolicyZone struct {
	Zone     string   `parser:"'zone' ( @String | @Ident )"`
	Switches []string `parser:"( @String | @Ident )*"`
}

// Returns the serialized BIND 9 configuration for the response-policy zone.
func (rpz *ResponsePolicyZone) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClausef(`zone "%s"`, rpz.Zone)
	for _, sw := range rpz.Switches {
		clause.addToken(sw)
	}
	return clause
}
