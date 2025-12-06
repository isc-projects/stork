package bind9config

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/pkg/errors"
)

// Boolean is a boolean set to true if the value is "true", "yes", or "1".
type Boolean bool

// Capture captures the boolean value from the token stream.
func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true" || values[0] == "yes" || values[0] == "1"
	return nil
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
			// The [\s\S]* matches whitespace and non-whitespace characters, so
			// includes new lines.
			{Name: "NoParseContents", Pattern: `[\S\s]*?//@stork:no-parse:`, Action: lexer.Pop()},
			lexer.Return(),
		},
		"NoParseGlobal": {
			// The [\s\S]* matches whitespace and non-whitespace characters, so
			// includes new lines.
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
func (p *Parser) parse(filename string, chrootDir string, fileReader io.Reader, parser *participle.Parser[Config]) (*Config, error) {
	// Run the parser.
	configPath := filepath.Join(chrootDir, filename)
	config, err := parser.Parse(configPath, fileReader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Bind9 config file: %s", filename)
	}
	// Optionally set the absolute source path. If it may be used for detecting
	// cycles in the include statements.
	config.sourcePath = filepath.Clean(filename)
	if sourcePath, err := filepath.Abs(configPath); err == nil {
		// Strip the chroot directory from the source path.
		if chrootDir != "" {
			config.chrootDir = chrootDir
			chrootDirAbs, err := filepath.Abs(chrootDir)
			if err == nil {
				config.chrootDir = chrootDirAbs
				sourcePath = strings.TrimPrefix(sourcePath, chrootDirAbs)
			}
		}
		config.sourcePath = sourcePath
	}
	return config, nil
}

// Parses the BIND 9 configuration from a file. It accepts a path to the file
// and a root of chroot if the configuration resides in a chroot.
func (p *Parser) ParseFile(filename string, chrootDir string) (*Config, error) {
	file, err := os.Open(path.Join(chrootDir, filename))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open BIND 9 config file: %s", filename)
	}
	defer file.Close()
	return p.Parse(filename, chrootDir, file)
}

// Parses the BIND 9 configuration.
func (p *Parser) Parse(filename string, chrootDir string, fileReader io.Reader) (*Config, error) {
	return p.parse(filename, chrootDir, fileReader, bind9Parser)
}
