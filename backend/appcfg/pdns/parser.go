package pdnsconfig

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

const (
	// Initial size of the buffer for a single line in the parser.
	minParserBufferSize = 512
	// Maximum size of the buffer for a single line in the parser.
	maxParserBufferSize = 16 * 1024
)

// Parser is a parser for PowerDNS configuration files using the
// key=values format.
type Parser struct{}

// Instantiates the parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parses the PowerDNS configuration from a reader.
func (p *Parser) Parse(reader io.Reader) (*Config, error) {
	// Define a map for storing key/values pairs. Note that there may be
	// multiple values separated by commas or spaces.
	parsedMap := make(map[string][]ParsedValue)

	// The default scanner splits the input into lines.
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, minParserBufferSize), maxParserBufferSize)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			// Skip empty lines and comments.
			continue
		}
		// Split the line into key and values.
		key, values, found := strings.Cut(text, "=")
		if !found {
			// If the line does not contain an equal sign, it is a boolean
			// value of true.
			parsedMap[text] = []ParsedValue{{boolValue: storkutil.Ptr(true)}}
			continue
		}
		// Remove leading and trailing whitespace from the key.
		// If it is empty, skip the line.
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		// Split the values into tokens separated by spaces or commas.
		fields := strings.FieldsFunc(values, func(r rune) bool {
			return unicode.IsSpace(r) || r == ','
		})
		parsedValues := make([]ParsedValue, 0, len(fields))
		for _, field := range fields {
			// Remove leading and trailing whitespace from the field.
			// If it is empty, skip the field.
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			var parsedValue ParsedValue
			switch field {
			case "yes":
				parsedValue.boolValue = storkutil.Ptr(true)
			case "no":
				parsedValue.boolValue = storkutil.Ptr(false)
			default:
				// If the field is a number, parse it as an integer.
				if intField, err := strconv.ParseInt(field, 10, 64); err == nil {
					parsedValue.int64Value = &intField
				} else {
					parsedValue.stringValue = &field
				}
			}
			parsedValues = append(parsedValues, parsedValue)
		}
		if len(parsedValues) > 0 {
			// Only set the key if there are any values.
			parsedMap[key] = parsedValues
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, errors.Wrapf(err, "encountered PowerDNS configuration line exceeding the maximum buffer size: %d", maxParserBufferSize)
		}
		return nil, errors.Wrap(err, "failed to parse PowerDNS configuration file")
	}
	return newConfig(parsedMap), nil
}

// Parses the PowerDNS configuration from a file.
func (p *Parser) ParseFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return p.Parse(file)
}
