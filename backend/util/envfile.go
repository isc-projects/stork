package storkutil

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// Stores key-value of the environment variable.
type KeyValuePair [2]string

// Returns the key of the key-value pair.
func (p KeyValuePair) GetKey() string {
	return p[0]
}

// Returns the value of the key-value pair.
func (p KeyValuePair) GetValue() string {
	return p[1]
}

// Defines an interface that accepts the environment variables.
type EnvironmentVariableSetter interface {
	Set(key, value string) error
}

// Setter that configures the environment variables for a current process.
type processEnvironmentVariableSetter struct{}

// Constructs the process environment variable setter.
func NewProcessEnvironmentVariableSetter() EnvironmentVariableSetter {
	return &processEnvironmentVariableSetter{}
}

// Implements the environment variable setter interface.
func (s *processEnvironmentVariableSetter) Set(key, value string) error {
	err := os.Setenv(key, value)
	err = errors.WithStack(err)
	return err
}

// Loads all entries from the environment file into one or multiple setters.
func LoadEnvironmentFileToSetter(path string, setters ...EnvironmentVariableSetter) error {
	data, err := LoadEnvironmentFile(path)
	if err != nil {
		return err
	}

	for _, pair := range data {
		for _, setter := range setters {
			err = setter.Set(pair.GetKey(), pair.GetValue())
			if err != nil {
				err = errors.WithMessagef(
					err,
					"cannot set '%s=%s' environment variable",
					pair.GetKey(),
					pair.GetValue(),
				)
				return err
			}
		}
	}

	return nil
}

// Loads all entries from the environment file.
func LoadEnvironmentFile(path string) ([]KeyValuePair, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open the '%s' environment file", path)
	}
	defer file.Close()
	return loadEnvironmentEntries(file)
}

// Loads all entries from a given reader.
func loadEnvironmentEntries(reader io.Reader) ([]KeyValuePair, error) {
	// The order of the entries is important.
	dataIndex := map[string]string{}
	scanner := bufio.NewScanner(reader)

	lineIdx := 0
	for scanner.Scan() {
		lineIdx++
		key, value, err := loadEnvironmentLine(scanner.Text())
		if err != nil {
			return nil, errors.WithMessagef(err, "invalid line %d of environment file", lineIdx)
		}
		if key == "" {
			// Comment.
			continue
		}

		dataIndex[key] = value
	}

	var data []KeyValuePair
	for key, value := range dataIndex {
		data = append(data, [2]string{key, value})
	}

	return data, nil
}

// Parses a line of the environment file.
func loadEnvironmentLine(line string) (string, string, error) {
	line = strings.TrimSpace(line)

	if line == "" {
		// Empty line - skip
		return "", "", nil
	}

	if strings.HasPrefix(line, "#") {
		// Comment - skip.
		return "", "", nil
	}

	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", errors.Errorf("line must contain the key and value separated by the '=' sign")
	}

	if key == "" {
		return "", "", errors.Errorf("key cannot be empty")
	}

	return key, value, nil
}
