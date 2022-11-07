package storkutil

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// Loads all entries from the environment file and uses them as environment
// variables for a current process.
func LoadEnvironmentFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "cannot open the '%s' environment file", path)
	}
	defer file.Close()
	return loadEnvironmentEntries(file)
}

// Loads all entries from a given reader and uses them as environment variable
// for a current process.
func loadEnvironmentEntries(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)

	lineIdx := 0
	for scanner.Scan() {
		lineIdx++
		err := loadEnvironmentLine(scanner.Text())
		if err != nil {
			return errors.WithMessagef(err, "invalid line %d of environment file", lineIdx)
		}
	}

	return nil
}

// Parses a line of the environment file. The parsed key and value is used as
// environment variable for a current process.
func loadEnvironmentLine(line string) error {
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "#") {
		// Comment - skip.
		return nil
	}

	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return errors.Errorf("line must contain the key and value separated by the '=' sign")
	}

	err := os.Setenv(key, value)
	err = errors.Wrap(err, "invalid key or value")
	return err
}
