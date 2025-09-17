package bind9config

var _ formattedElement = (*UnixClause)(nil)

// A unix clause within the controls statement. The format of the unix clause is:
//
// unix <quoted_string> perm <integer> owner <integer> group <integer> [ keys { <string>; ... } ] [ read-only <boolean> ];
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#namedconf-statement-unix
type UnixClause struct {
	Path     string   `parser:"@String"`
	Perm     int64    `parser:"'perm' @Ident"`
	Owner    int64    `parser:"'owner' @Ident"`
	Group    int64    `parser:"'group' @Ident"`
	Keys     *Keys    `parser:"( 'keys' '{' @@ '}' )?"`
	ReadOnly *Boolean `parser:"( 'read-only' @Ident )?"`
}

// Returns the serialized BIND 9 configuration for the unix clause.
func (u *UnixClause) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClausef(`unix "%s" perm %04o owner %d group %d`, u.Path, u.Perm, u.Owner, u.Group)
	if u.Keys != nil {
		clause.add(u.Keys.getFormattedOutput(filter))
	}
	if u.ReadOnly != nil {
		switch *u.ReadOnly {
		case true:
			clause.addToken("read-only true")
		case false:
			clause.addToken("read-only false")
		}
	}
	return clause
}
