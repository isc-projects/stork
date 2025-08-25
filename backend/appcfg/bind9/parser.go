package bind9config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	// A Stork-specific annotation to skip parsing statements between the
	// @stork:no-parse:scope and @stork:no-parse:end directives, or after
	// the @stork:no-parse:global directive.
	NoParse *NoParse `parser:"@@"`

	// The "include statement is used to include another configuration file.
	Include *Include `parser:"| 'include' @@"`

	// The "acl" statement is used to define an access control list.
	ACL *ACL `parser:"| 'acl' @@"`

	// The "key" statement is used to define a secure key.
	Key *Key `parser:"| 'key' @@"`

	// The "options" statement is used to define global options.
	Options *Options `parser:"| 'options' @@"`

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

// A Stork-specific annotation to skip parsing statements between the
// @stork:no-parse:scope and @stork:no-parse:end directives, or after
// the @stork:no-parse:global directive.
type NoParse struct {
	NoParseScope  *NoParseScope  `parser:"( @@"`
	NoParseGlobal *NoParseGlobal `parser:"| @@ )"`
}

// Checks if the @stork:no-parse:global directive was used.
func (n *NoParse) IsGlobal() bool {
	return n.NoParseGlobal != nil
}

// Returns the unparsed contents within the @stork:no-parse:scope
// and @stork:no-parse:end directives, or after the @stork:no-parse:global
// directive.
func (n *NoParse) GetContentsString() string {
	switch {
	case n.NoParseScope != nil:
		return n.NoParseScope.Contents.GetString()
	case n.NoParseGlobal != nil:
		return n.NoParseGlobal.Contents.GetString()
	default:
		return ""
	}
}

// Represents the @stork:no-parse:scope/@stork:no-parse:end directives.
type NoParseScope struct {
	Preamble string      `parser:"@NoParseScope"`
	Contents RawContents `parser:"@NoParseContents"`
	End      string      `parser:"@NoParseEnd"`
}

// Represents the @stork:no-parse:global directive.
type NoParseGlobal struct {
	Preamble string      `parser:"@NoParseGlobal"`
	Contents RawContents `parser:"@NoParseGlobalContents"`
}

// Unparsed contents between the @stork:no-parse:scope and @stork:no-parse:end
// directives, or after the @stork:no-parse:global directive.
type RawContents string

// Captures the unparsed contents between the @stork:no-parse:scope
// and @stork:no-parse:end directives and removes the trailing
// @stork:no-parse: suffix which is appended by the lexer.
func (c *RawContents) Capture(values []string) error {
	if len(values) == 0 {
		return nil
	}
	values[len(values)-1] = strings.TrimSuffix(values[len(values)-1], "//@stork:no-parse:")
	*c = RawContents(strings.Join(values, ""))
	return nil
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
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
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
	Negation           bool   `parser:"@('!')?"`
	ACL                *ACL   `parser:"( '{' @@ '}'"`
	KeyID              string `parser:"| ( 'key' ( @Ident | @String ) )"`
	IPAddressOrACLName string `parser:"| ( @Ident | @String ) )"`
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

// Options is the statement used to define global options.
// This section has the following format:
//
//	options {
//		<option-clauses> ...
//	};
//
// See: https://bind9.readthedocs.io/en/stable/reference.html#options-block-grammar.
type Options struct {
	// Cache the response-policy only once.
	responsePolicyOnce sync.Once
	// The response-policy clause cache for better access performance.
	responsePolicy *ResponsePolicy
	// The list of clauses (e.g., allow-transfer, listen-on, response-policy etc.).
	Clauses []*OptionClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// OptionClause is a single clause of an options statement.
type OptionClause struct {
	// A Stork-specific annotation to skip parsing statements between the
	// @stork:no-parse:scope and @stork:no-parse:end directives, or after
	// the @stork:no-parse:global directive.
	NoParse *NoParse `parser:"@@"`
	// The allow-transfer clause restricting who can perform AXFR.
	AllowTransfer *AllowTransfer `parser:"| 'allow-transfer' @@"`
	// The listen-on clause specifying the addresses the server listens
	// on the DNS requests.
	ListenOn *ListenOn `parser:"| 'listen-on' @@"`
	// The listen-on-v6 clause specifying the addresses the server listens
	// on the DNS requests.
	ListenOnV6 *ListenOn `parser:"| 'listen-on-v6' @@"`
	// The response-policy clause specifying the response policy zones.
	ResponsePolicy *ResponsePolicy `parser:"| 'response-policy' @@"`
	// Any option clause.
	Option *Option `parser:"| @@"`
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
	// Cache the response-policy only once.
	responsePolicyOnce sync.Once
	// The response-policy clause cache for better access performance.
	responsePolicy *ResponsePolicy
	// The name of the view statement.
	Name string `parser:"( @String | @Ident )"`
	// An optional class of the view statement.
	Class string `parser:"( @String | @Ident )?"`
	// The list of clauses (e.g., match-clients, zone etc.).
	Clauses []*ViewClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// ViewClause is a single clause of a view statement.
type ViewClause struct {
	// A Stork-specific annotation to skip parsing statements between the
	// @stork:no-parse:scope and @stork:no-parse:end directives, or after
	// the @stork:no-parse:global directive.
	NoParse *NoParse `parser:"@@"`
	// The match-clients clause associating the view with ACLs.
	MatchClients *MatchClients `parser:"| 'match-clients' @@"`
	// The allow-transfer clause restricting who can perform AXFR.
	AllowTransfer *AllowTransfer `parser:"| 'allow-transfer' @@"`
	// The response-policy clause specifying the response policy zones.
	ResponsePolicy *ResponsePolicy `parser:"| 'response-policy' @@"`
	// The zone clause associating the zone with a view.
	Zone *Zone `parser:"| 'zone' @@"`
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
	// The list of clauses (e.g., match-clients, allow-transfer etc.).
	// This is made optional to allow quicker parsing of the zone definition,
	// with the zone-level options elided.
	Clauses []*ZoneClause `parser:"( '{' ( @@ ';'* )* '}' )?"`
}

// ZoneClause is a single clause of a zone statement.
type ZoneClause struct {
	NoParse *NoParse `parser:"@@"`
	// The allow-transfer clause restricting who can perform AXFR.
	AllowTransfer *AllowTransfer `parser:"| 'allow-transfer' @@"`
	// Any option clause.
	Option *Option `parser:"| @@"`
}

// MatchClients is the clause for associations with ACLs. It can be used in the
// view to associate this view with specific ACLs.
type MatchClients struct {
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// AllowTransfer is the clause for restricting who can perform AXFR
// globally, for a particular view or zone.
type AllowTransfer struct {
	Port             *int64            `parser:"( 'port' @Ident )?"`
	Transport        *string           `parser:"( 'transport' ( @String | @Ident ) )?"`
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// ListenOn is the clause specifying the addresses the servers listens on the
// DNS requests. It also contains additional options.
type ListenOn struct {
	Port             *int64            `parser:"( 'port' @Ident )?"`
	Proxy            *string           `parser:"( 'proxy' ( @String | @Ident ) )?"`
	TLS              *string           `parser:"( 'tls' ( @String | @Ident ) )?"`
	HTTP             *string           `parser:"( 'http' ( @String | @Ident ) )?"`
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// ResponsePolicy is the clause specifying the response policy zones.
type ResponsePolicy struct {
	Zones    []*ResponsePolicyZone `parser:"'{' ( @@ ';'+ )* '}'"`
	Switches []string              `parser:"( @String | @Ident )*"`
}

// ResponsePolicyZone is a single response policy zone entry.
type ResponsePolicyZone struct {
	Zone     string   `parser:"'zone' ( @String | @Ident )"`
	Switches []string `parser:"( @String | @Ident )*"`
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
	Identifier string                 `parser:"@Ident"`
	Switches   []string               `parser:"( @String | @Ident )*"`
	Contents   *GenericClauseContents `parser:"( '{' @@ '}' )?"`
	Suboptions []Suboption            `parser:"( @@ )*"`
}

// Suboption is a generic catch-all clause being an optional part of an
// option. Suboptions can appear after curly braces in the option.
type Suboption struct {
	Identifier string                 `parser:"@Ident"`
	Switches   []string               `parser:"( @String | @Ident )*"`
	Contents   *GenericClauseContents `parser:"( '{' @@ '}' )?"`
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

var (
	// Custom lexer. It is used to tokenize the input stream into tokens
	// meaningful for the named configuration parser. It drops the comments
	// and whitespace. It also drops the configuration parts annotated with
	// the @stork:no-parse directives. For example, to skip parsing a given
	// zone definition, annotate it with:
	//
	//	//@stork:no-parse:scope
	//	zone "example.com" {
	//		type master;
	//		allow-transfer port 853 { any; };
	//		file "/etc/bind/db.example.com";
	//	};
	//	//@stork:no-parse:end
	//
	// If only specific parts of the zone definition should be skipped, one
	// can do:
	//
	//	zone "example.com" {
	//		//@stork:no-parse:scope
	//		type master;
	//		file "/etc/bind/db.example.com";
	//		//@stork:no-parse:end
	//		allow-transfer port 853 { any; };
	//	};
	//
	// The @stork:no-parse directive can be used for other statements as well.
	// It is not limited to the zone definition. For example, it can be used
	// to skip parsing an included file, options, views and the inner statements
	// within these configuration elements.
	//
	// If the interesting configuration part is at the beginning of a file and
	// the parse to be skipped is at the end, use the @stork:no-parse:global
	// directive to annotate the rest of the file to be skipped.
	//nolint:gochecknoglobals
	bind9Lexer = lexer.MustStateful(lexer.Rules{
		"Root": {
			{Name: "noParse", Pattern: `//@stork:no-parse:`, Action: lexer.Push("NoParse")},
			{Name: "comment", Pattern: `(//|#)[^\n]*`},
			{Name: "cppStyleComment", Pattern: `\/\*([^*]|(\*+[^*\/]))*\*+\/`},
			{Name: "String", Pattern: `"(\\"|[^"])*"`},
			{Name: "Ident", Pattern: `[0-9a-zA-Z-_\.\:\/\*]+`},
			{Name: "whitespace", Pattern: `[ \t\n\r]+`},
			{Name: "Punct", Pattern: `[;,{}!]`},
		},
		"NoParse": {
			{Name: "NoParseScope", Pattern: `scope`, Action: lexer.Push("NoParseScope")},
			{Name: "NoParseGlobal", Pattern: `global`, Action: lexer.Push("NoParseGlobal")},
			{Name: "NoParseEnd", Pattern: `end`, Action: lexer.Pop()},
			lexer.Return(),
		},
		"NoParseScope": {
			{Name: "NoParseContents", Pattern: `[\S\s]*?//@stork:no-parse:`, Action: lexer.Pop()},
			lexer.Return(),
		},
		"NoParseGlobal": {
			{Name: "NoParseGlobalContents", Pattern: `[\s\S]*`},
		},
	})

	// The parser uses the custom lexer.
	//nolint:gochecknoglobals
	bind9Parser = participle.MustBuild[Config](
		// Use custom lexer instead of the default one.
		participle.Lexer(bind9Lexer),
		// Remove quotes from the strings and other quoted tokens.
		participle.Unquote("String"),
	)
)

// Parser is a parser for the BIND 9 configuration.
type Parser struct{}

// Instantiates a new parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parses the BIND 9 configuration from a file using a custom parser.
func (p *Parser) parse(filename string, fileReader io.Reader, parser *participle.Parser[Config]) (*Config, error) {
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

// Parses the BIND 9 configuration from a file.
func (p *Parser) ParseFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open BIND 9 config file: %s", filename)
	}
	defer file.Close()
	return p.Parse(filename, file)
}

// Parses the BIND 9 configuration.
func (p *Parser) Parse(filename string, fileReader io.Reader) (*Config, error) {
	return p.parse(filename, fileReader, bind9Parser)
}
