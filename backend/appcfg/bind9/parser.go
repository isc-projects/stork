package bind9config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/pkg/errors"
)

// Config is the root of the Bind9 configuration. It contains a list of
// top-level statements. The statements typically contain clauses with
// configuration elements.
type Config struct {
	// The absolute source path of the configuration file. Note that it
	// may be not set if getting the absolute path failed.
	sourcePath string
	// The configuration contains a list of Statements separated by semicolons.
	Statements []*Statement `parser:"( @@ ';'* )*"`
}

// Statement is a single top-level configuration element.
type Statement struct {
	// The "include statement is used to include another configuration file.
	Include *Include `parser:"'include' @@"`

	// The "acl" statement is used to define an access control list.
	ACL *ACL `parser:"| 'acl' @@"`

	// The "key" statement is used to define a secure key.
	Key *Key `parser:"| 'key' @@"`

	// The "view" statement is used to define a view (i.e., a logical
	// DNS server instance)
	View *View `parser:"| 'view' @@"`

	// The "zone" statement is used to define a DNS zone.
	Zone *Zone `parser:"| 'zone' @@"`

	// A generic catch-all named statement. It is used to parse any statement
	// not covered explicitly above and having the following format:
	//
	//	<identifier> <name> { <block> }
	NamedStatement *NamedStatement `parser:"| @@"`

	// A generic catch-all unnamed statement. It is used to parse any statement
	// not covered explicitly above and having the following format:
	//
	//	<identifier> { <block> }
	UnnamedStatement *UnnamedStatement `parser:"| @@"`
}

// Include is the statement used to include another configuration file.
// The included file can be parsed and its configuration statements expand
// the parent configuration. The "include" statement has the following format:
//
// include <filename>;
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#include-directive.
type Include struct {
	// Included file path.
	Path string `parser:"@String"`
}

// ACL is the statement used to define an access control list.
// The "acl" statement has the following format:
//
// acl <name> { <address-match-list> };
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#acl-block-grammar.
type ACL struct {
	// The name of the ACL.
	Name string `parser:"( @String | @Ident )"`
	// The list of address match list elements between curly braces.
	AdressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// AddressMatchList is the list of address match list elements between curly braces.
// The address match list elements include but are not limited to: IP addresses,
// keys, or ACLs. The elements may also contain a negation sign. It is used to
// exclude certain clients from the ACLs. The address match list has the following
// format:
//
//	[ ! ] ( <ip_address> | <netprefix> | key <server_key> | <acl_name> | { address_match_list } )
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#term-address_match_element.
type AddressMatchList struct {
	Elements []*AddressMatchListElement `parser:"( @@ ';'* )*"`
}

// AddressMatchListElement is an element of an address match list.
type AddressMatchListElement struct {
	Negation  bool   `parser:"@('!')?"`
	ACL       *ACL   `parser:"( '{' @@ '}'"`
	KeyID     string `parser:"| ( 'key' ( @Ident | @String ) )"`
	IPAddress string `parser:"| ( @IPv4Address | @IPv6Address | @IPv4AddressQuoted | @IPv6AddressQuoted )"`
	ACLName   string `parser:"| ( @Ident | @String ) )"`
}

// Key is the statement used to define an algorithm and secret. It has the following
// format:
//
//	key <name> key <string> {
//		algorithm <string>;
//		secret <string>;
//	};
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#key-block-grammar.
type Key struct {
	// The name of the key statement.
	Name string `parser:"( @String | @Ident )"`
	// The list of clauses: an algorithm and secret. Note that they are defined
	// a list (rather than explicitly) because they can be defined in any order.
	Clauses []*KeyClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// KeyClause is a single clause of a key statement: an algorithm or secret.
type KeyClause struct {
	// The algorithm clause.
	Algorithm string `parser:"'algorithm' ( @Ident | @String )"`
	// The secret clause.
	Secret string `parser:"| 'secret' ( @Ident | @String )"`
}

// View is the statement used to define a DNS view. The view is a logical
// DNS server instance including its own set of zones. The view has the
// following format:
//
//	view <name> [ <class> ] {
//		<view-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#view-block-grammar.
type View struct {
	// The name of the view statement.
	Name string `parser:"( @String | @Ident )"`
	// An optional class of the view statement.
	Class string `parser:"( @String | @Ident )?"`
	// The list of clauses (e.g., match-clients, zone etc.).
	Clauses []*ViewClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// ViewClause is a single clause of a view statement.
type ViewClause struct {
	// The match-clients clause associating the view with ACLs.
	MatchClients *MatchClients `parser:"'match-clients' @@"`
	// The zone clause associating the zone with a view.
	Zone *Zone `parser:"| 'zone' @@"`
	// Any namedClause clause.
	NamedClause *NamedStatement `parser:"| @@"`
	// Any unnamedClause clause.
	UnnamedClause *UnnamedStatement `parser:"| @@"`
	// Any option clause.
	Option *Option `parser:"| @@"`
}

// Zone is the statement used to define a zone. The zone has the following format:
//
//	zone <name> [ <class> ] {
//		<zone-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#zone-block-grammar.
type Zone struct {
	// The name of the zone statement.
	Name string `parser:"( @String | @Ident )"`
	// The class of the zone statement.
	Class string `parser:"( @String | @Ident )?"`
	// The list of clauses (e.g., match-clients, zone etc.).
	Clauses []*ZoneClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// ZoneClause is a single clause of a zone statement.
type ZoneClause struct {
	// Any namedClause clause.
	NamedClause *NamedStatement `parser:"@@"`
	// Any unnamedClause clause.
	UnnamedClause *UnnamedStatement `parser:"| @@"`
	// Any option clause.
	Option *Option `parser:"| @@"`
}

// MatchClients is the clause for associations with ACLs. It can be used in the
// view to associate this view with specific ACLs.
type MatchClients struct {
	AdressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// NamedStatement is a generic catch-all named statement. It is used to parse
// any statement having the following format:
//
//	<identifier> <name> { <block> };
//
// The "tls" statement is an example:
// https://bind9.readthedocs.io/en/stable/reference.html#tls-block-grammar.
type NamedStatement struct {
	// The Identifier of the named statement.
	Identifier string `parser:"@Ident"`
	// The Name of the named statement.
	Name string `parser:"( @String | @Ident )"`
	// The Contents of the named statement.
	Contents *GenericClauseContents `parser:"'{' @@ '}'"`
}

// unnamedStatement is a generic catch-all unnamed statement. It is used to parse
// any statement having the following format:
//
//	<identifier> { <block> };
//
// The "managed-keys" statement is an example:
// /https://bind9.readthedocs.io/en/stable/reference.html#managed-keys-block-grammar.
type UnnamedStatement struct {
	// The Identifier of the unnamed statement.
	Identifier string `parser:"@Ident"`
	// The Contents of the unnamed statement.
	Contents *GenericClauseContents `parser:"'{' @@ '}'"`
}

// Option is a generic catch-all option clause. It is used to parse an option
// having the following format:
//
//	<identifier> <block>
//
// Many options in the options statement have this format.
type Option struct {
	Identifier string `parser:"@Ident"`
	Contents   string `parser:"( @String | @Ident )"`
}

// GenericClauseContents is used to parse any type of contents. It is
// used for parsing the configuration elements that are not explicitly
// supported in this parser. It consumes and discards all tokens until
// EOF or extraneous closing brace is found.
type GenericClauseContents struct{}

// Parses the contents of a generic clause.
func (b *GenericClauseContents) Parse(lex *lexer.PeekingLexer) error {
	cnt := 0
	for {
		// Get the next token without consuming it.
		token := lex.Peek()
		switch {
		case token.EOF():
			// The end of the statement contents.
			return nil
		case token.Value == "{":
			// Opening new sub-statement. Increase the
			// counter to keep track of the nesting level.
			cnt++
		case token.Value == "}":
			// Closing sub-statement. Decrease the counter
			// to keep track of the nesting level.
			cnt--
			if cnt < 0 {
				// Extraneous closing brace found.
				return nil
			}
		}
		// Consume the token.
		_ = lex.Next()
	}
}

// Parses the Bind9 configuration from a file using custom lexer.
func ParseFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open Bind9 config file: %s", filename)
	}
	defer file.Close()
	return Parse(filename, file)
}

// Parse the Bind9 configuration using custom lexer.
func Parse(filename string, fileReader io.Reader) (*Config, error) {
	// Define the custom lexer. It is used to tokenize the input stream
	// into tokens meaningful for named configuration parser. Note that
	// many of the rules below can be considered simplistic (e.g., the
	// IPv4 or IPv6 address matching rules). However, it is not the purpose
	// of this parser to validate the named configuration file syntax.
	// Bind is responsible for validating it. We just want to reliably
	// recognize the tokens in the named configuration file.
	lexer := lexer.MustSimple([]lexer.SimpleRule{
		// Comments can begin with either "//" or "#". They are elided from
		// the token stream.
		{Name: "Comment", Pattern: `(//|#)[^\n]*`},
		// C-style comments are also elided from the token stream.
		{Name: "CppStyleComment", Pattern: `\/\*([^*]|(\*+[^*\/]))*\*+\/`},
		// IPv4 addresses and subnets can be specified with or without quotes.
		// This variant assumes the lack of quotes.
		{Name: "IPv4Address", Pattern: `(?:([0-9]{1,3}\.){3}(?:[0-9]{1,3}))(?:/(?:[0-9]{1,2}))?`},
		// IPv6 addresses and subnets can be specified with or without quotes.
		// This variant assumes the lack of quotes.
		{Name: "IPv6Address", Pattern: `(?:([0-9a-fA-F]{1,4}:{1,2}){1,8})(?:/[0-9]{1,3})?`},
		// IPv4 addresses and subnets can be specified with quotes.
		{Name: "IPv4AddressQuoted", Pattern: `"(?:([0-9]{1,3}\.){3}(?:[0-9]{1,3}))(?:/(?:[0-9]{1,2}))?"`},
		// IPv6 addresses and subnets can be specified with quotes.
		{Name: "IPv6AddressQuoted", Pattern: `"(?:([0-9a-fA-F]{1,4}:{1,2}){1,8})(?:/[0-9]{1,3})?"`},
		// Strings are always quoted.
		{Name: "String", Pattern: `"(\\"|[^"])*"`},
		// Numbers.
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		// Identifiers are alphanumeric strings specified without quotes.
		// Note that the Bind9 configuration parser allows for specifying
		// configuration element names (and values) in quotes or without quotes.
		// The identifier handles this second case.
		{Name: "Ident", Pattern: `[0-9a-zA-Z-_]+`},
		// Punctuation characters.
		{Name: "Punct", Pattern: `[;,.{}!]`},
		// Whitespace characters.
		{Name: "Whitespace", Pattern: `[ \t\n\r]+`},
		// End of line characters.
		{Name: "EOL", Pattern: `[\n\r]+`},
	})

	parser := participle.MustBuild[Config](
		// Use custom lexer instead of the default one.
		participle.Lexer(lexer),
		// Remove quotes from the strings and other quoted tokens.
		participle.Unquote("String", "IPv4AddressQuoted", "IPv6AddressQuoted"),
		// Ignore whitespace and comments.
		participle.Elide("Whitespace", "Comment", "CppStyleComment"),
		// Use lookahead to improve the parsing accuracy.
		participle.UseLookahead(2),
	)
	// Run the parser.
	config, err := parser.Parse(filename, fileReader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Bind9 config file: %s", filename)
	}
	// Optionally set the absolute source path. If it may be used for detecting
	// cycles in the include statements.
	if sourcePath, err := filepath.Abs(filename); err == nil {
		config.sourcePath = sourcePath
	}
	return config, nil
}
