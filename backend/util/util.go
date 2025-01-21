package storkutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// Returns a current time in the UTC time zone.
func UTCNow() time.Time {
	return time.Now().UTC()
}

// Returns URL of the host with port.
func HostWithPortURL(address string, port int64, secure bool) string {
	protocol := "http"
	if secure {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d/", protocol, address, port)
}

// Parses URL into host and port.
func ParseURL(url string) (host string, port int64, secure bool) {
	pattern := regexp.MustCompile(`https{0,1}:\/\/\[{1}(\S+)\]{1}(:([0-9]+)){0,1}`)
	m := pattern.FindStringSubmatch(url)

	if len(m) == 0 {
		pattern := regexp.MustCompile(`https{0,1}:\/\/([^\s\:\/]+)(:([0-9]+)){0,1}`)
		m = pattern.FindStringSubmatch(url)
	}

	if len(m) > 1 {
		host = m[1]
	}

	if len(m) > 3 {
		p, err := strconv.Atoi(m[3])
		if err == nil {
			port = int64(p)
		}
	}

	secure = strings.HasPrefix(url, "https://")

	// Set default ports
	if port == 0 {
		switch {
		case strings.HasPrefix(url, "http://"):
			port = 80
		case strings.HasPrefix(url, "https://"):
			port = 443
		}
	}

	return host, port, secure
}

// Formats provided string of hexadecimal digits to MAC address format
// using colon as separator. It returns formatted string and a boolean
// value indicating if the conversion was successful.
func FormatMACAddress(identifier string) (formatted string, ok bool) {
	// Check if the identifier is already in the desired format.
	identifier = strings.TrimSpace(identifier)
	pattern := regexp.MustCompile(`^[0-9A-Fa-f]{2}((:{1})[0-9A-Fa-f]{2})*$`)
	if pattern.MatchString(identifier) {
		// No conversion required. Return the input.
		return identifier, true
	}
	// We will have to convert it, but let's first check if this is a valid identifier.
	if !IsHexIdentifier(identifier) {
		return "", false
	}
	// Remove any colons and whitespace.
	replacer := strings.NewReplacer(" ", "", ":", "")
	numericOnly := replacer.Replace(identifier)
	for i, character := range numericOnly {
		formatted += string(character)
		// Divide the string into groups with two digits.
		if i > 0 && i%2 != 0 && i < len(numericOnly)-1 {
			formatted += ":"
		}
	}
	return formatted, true
}

// Detects if the provided string is an identifier consisting of
// hexadecimal digits and optionally whitespace or colons between
// the groups of digits. For example: 010203, 01:02:03, 01::02::03,
// 01 02 03 etc. It is useful in detecting if the string comprises
// a DHCP client identifier or MAC address.
func IsHexIdentifier(text string) bool {
	pattern := regexp.MustCompile(`^[0-9A-Fa-f]{2}((\s*|:{0,2})[0-9A-Fa-f]{2})*$`)
	return pattern.MatchString(strings.TrimSpace(text))
}

// Counts the number of bytes in the provided hex identifier.
// It doesn't check if the input is a valid hex identifier. It may return
// a value even for malformed input. Returns zero if the input is not a hex
// identifier.
func CountHexIdentifierBytes(text string) int {
	if !IsHexIdentifier(text) {
		return 0
	}

	// Remove any whitespace and colons.
	replacer := strings.NewReplacer(" ", "", ":", "")
	identifier := replacer.Replace(text)
	return len(identifier) / 2
}

// Sets up the logging level. It's set to INFO, unless STORK_LOG_LEVEL
// environment variable is present. If it is, its value governs the level.
// Supported levels are: DEBUG, INFO, WARN, ERROR.
func SetupLoggingLevel() {
	// Setup logging level.
	//
	// If the STORK_LOG_LEVEL is specified and has valid name in it, use
	// that level. If not specified or has garbage, use the default (INFO).
	if value, ok := os.LookupEnv("STORK_LOG_LEVEL"); ok {
		levels := map[string]log.Level{
			"DEBUG": log.DebugLevel,
			"INFO":  log.InfoLevel,
			"WARN":  log.WarnLevel,
			"ERROR": log.ErrorLevel,
		}

		if level, ok := levels[value]; ok {
			// STORK_LOG_LEVEL specified, setting logging level to whatever was specified.
			log.SetLevel(level)
		} else {
			// Invalid level. Let's initialize logging first and then log a
			// warning about it.
			log.SetLevel(log.InfoLevel)
			log.Warnf("STORK_LOG_LEVEL has invalid log level: %s, ignoring.", value)
		}
	} else {
		// STORK_LOG_LEVEL not specified, using default logging level (INFO).
		log.SetLevel(log.InfoLevel)
	}
}

// Parses the CLI boolean flag value. It returns true if the value is
// "true", "1", "t", (case insensitive). Otherwise, It returns false if the
// value is "false", "0", "f", (case insensitive). Otherwise, it returns an
// error.
func ParseBoolFlag(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1", "t":
		return true, nil
	case "false", "0", "f":
		return false, nil
	default:
		return false, errors.Errorf("invalid boolean value: '%s', allowed: [true, 1, false, 0]", value)
	}
}

// Configures the main application logger. Additionally, it converts the
// values of the console color-related environment variables.
func SetupLogging() {
	// Normalizes the color environment variables from the standard Stork
	// convention.
	variables := []string{"CLICOLOR_FORCE", "CLICOLOR"}
	for _, variable := range variables {
		value, ok := os.LookupEnv(variable)
		if !ok {
			continue
		}
		boolValue, err := ParseBoolFlag(value)
		if err != nil {
			continue
		}

		if boolValue {
			os.Setenv(variable, "1")
		} else {
			os.Setenv(variable, "0")
		}
	}

	SetupLoggingLevel()

	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		ForceQuote:                true,
		TimestampFormat:           "2006-01-02 15:04:05",
		// TODO: do more research and enable if it brings value
		// PadLevelText: true,
		// FieldMap: log.FieldMap{
		// 	FieldKeyTime:  "@timestamp",
		// 	FieldKeyLevel: "@level",
		// 	FieldKeyMsg:   "@message",
		// },
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			// Grab filename and line of current frame and add it to log entry
			_, filename := path.Split(f.File)
			return "", fmt.Sprintf("%20v:%-5d", filename, f.Line)
		},
	})
}

// The command executor is an abstraction layer on top of the exec package to
// improve testability and allow mock the operating system operations.
type CommandExecutor interface {
	Output(string, ...string) ([]byte, error)
	LookPath(string) (string, error)
	IsFileExist(string) bool
}

// Executes the given command in the operating system.
type systemCommandExecutor struct{}

// Constructs the command executor that invokes the requests within the system
// shell.
func NewSystemCommandExecutor() CommandExecutor {
	return &systemCommandExecutor{}
}

// Executes a given command in the system shell and returns an output.
func (e *systemCommandExecutor) Output(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).Output()
}

// Looks for a given command in the system PATH and returns absolute path if found.
func (e *systemCommandExecutor) LookPath(command string) (string, error) {
	return exec.LookPath(command)
}

// Looks for a given file. Returns true is the path exist, is accessible, and
// points to a file.
func (e *systemCommandExecutor) IsFileExist(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		return stat.Mode().IsRegular()
	}
	return false
}

// Convert bytes to hex string.
func BytesToHex(bytesArray []byte) string {
	var buf bytes.Buffer
	for _, f := range bytesArray {
		fmt.Fprintf(&buf, "%02X", f)
	}
	return buf.String()
}

// Convert a string of hexadecimal digits to bytes array.
func HexToBytes(hexString string) []byte {
	replacer := strings.NewReplacer(" ", "", ":", "", "-", "")
	hexString = replacer.Replace(hexString)
	decoded, _ := hex.DecodeString(hexString)
	return decoded
}

// Prompts for secret in a terminal. The typed characters are not printed to
// standard output. If a terminal is unavailable (no TTY attached), returns
// an error.
func GetSecretInTerminal(prompt string) (string, error) {
	// Prompt the user for a secret
	descriptor := int(os.Stdin.Fd())
	if !term.IsTerminal(descriptor) {
		return "", errors.Errorf("Not running in a terminal")
	}

	fmt.Print(prompt)
	pass, err := term.ReadPassword(descriptor)
	fmt.Print("\n")

	if err != nil {
		err = errors.Wrapf(err, "cannot read password for the '%d' descriptor", descriptor)
		return "", err
	}
	return string(pass), nil
}

// Checks if the current process is running in a terminal.
func IsRunningInTerminal() bool {
	descriptor := int(os.Stdin.Fd())
	return term.IsTerminal(descriptor)
}

// Read a file and resolve all include statements.
func ReadFileWithIncludes(path string) (string, error) {
	parentPaths := map[string]bool{}
	return readFileWithIncludes(path, parentPaths)
}

// Recursive function to read a file and resolve all include statements.
func readFileWithIncludes(path string, parentPaths map[string]bool) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		err = errors.Wrap(err, "cannot read the configuration file")
		return "", err
	}

	text := string(raw)

	// Include pattern definition:
	// - Must start with prefix: <?include
	// - Must end with suffix: ?>
	// - Path may be relative to parent file or absolute
	// - Path must be escaped with double quotes
	// - May to contains spacing before and after the path quotes
	// Produce two groups: first for the whole statement and second for path.
	includePattern := regexp.MustCompile(`<\?include\s*\"([^"]+\..*)\"\s*\?>`)
	matchesGroupIndices := includePattern.FindAllStringSubmatchIndex(text, -1)
	matchesGroups := includePattern.FindAllStringSubmatch(text, -1)

	// Probably never met
	if (matchesGroupIndices == nil) != (matchesGroups == nil) {
		return "", errors.New("include statement recognition failed")
	}

	// No matches - nothing to expand
	if matchesGroupIndices == nil {
		return text, nil
	}

	// Probably never met
	if len(matchesGroupIndices) != len(matchesGroups) {
		return "", errors.New("include statement recognition asymmetric")
	}

	// The root directory for includes
	baseDirectory := filepath.Dir(path)

	// Iteration from the end to keep correct index values because when the pattern
	// is replaced with an include content the positions of next patterns are shifting
	for i := len(matchesGroupIndices) - 1; i >= 0; i-- {
		matchedGroupIndex := matchesGroupIndices[i]
		matchedGroup := matchesGroups[i]

		statementStartIndex := matchedGroupIndex[0]
		matchedPath := matchedGroup[1]
		matchedStatementLength := len(matchedGroup[0])
		statementEndIndex := statementStartIndex + matchedStatementLength

		// Include path may be absolute or relative to a parent file
		nestedIncludePath := matchedPath
		if !filepath.IsAbs(nestedIncludePath) {
			nestedIncludePath = filepath.Join(baseDirectory, nestedIncludePath)
		}
		nestedIncludePath = filepath.Clean(nestedIncludePath)

		// Check for infinite loop
		_, isVisited := parentPaths[nestedIncludePath]
		if isVisited {
			err := errors.Errorf("detected infinite loop on include '%s' in file '%s'", matchedPath, path)
			return "", err
		}

		// Prepare the parent paths for the nested level
		nestedParentPaths := make(map[string]bool, len(parentPaths)+1)
		for k, v := range parentPaths {
			nestedParentPaths[k] = v
		}
		nestedParentPaths[nestedIncludePath] = true

		// Recursive call
		content, err := readFileWithIncludes(nestedIncludePath, nestedParentPaths)
		if err != nil {
			return "", errors.Wrapf(err, "problem with inner include: '%s' of '%s': '%s'", matchedPath, path, nestedIncludePath)
		}

		// Replace include statement with included content
		text = text[:statementStartIndex] + content + text[statementEndIndex:]
	}

	return text, nil
}

// Check if it is possible to create a file
// with the provided filename.
func IsValidFilename(filename string) bool {
	if strings.ContainsAny(filename, "*") {
		return false
	}
	file, err := os.CreateTemp("", filename+"*")
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(file.Name())
	return true
}

// Returns a string comprising a count and a noun in the plural or
// singular form, depending on the count. The third parameter is a
// postfix making the plural form.
func FormatNoun(count int64, noun, postfix string) string {
	formatted := fmt.Sprintf("%d %s", count, noun)
	if count != 1 && count != -1 {
		formatted += postfix
	}
	return formatted
}

// Check if the interface is nil pointer - (*T)(nil). It is helpful
// in the functions that accept an interface type. If the real
// type is a pointer to a struct that implements the interface,
// then standard nil checking (with == operator) always returns
// false even if the pointer is nil. It isn't a big problem in
// most cases because you can call the interface methods on the
// nil interface (the receiver will be nil). It's dangerous if
// you use the type composition. If you don't override the methods
// in derived types, then all calls are piped to the base type.
// But if you try to do this on nil interface, then GO panics.
// You cannot prevent it by standard nil checking in the functions
// that use the interface as an argument type. You have to use this
// helper function. Very confusing.
//
// Warning! If you need to use this function, then probably
// your code is inconsistent. It would be best if you didn't
// allow that nil pointers will be cast to interface{}.
//
// Source: https://stackoverflow.com/a/50487104 .
// See: https://groups.google.com/g/golang-nuts/c/wnH302gBa4I
func IsNilPtr(obj interface{}) bool {
	return obj == nil || reflect.ValueOf(obj).Kind() == reflect.Ptr && reflect.ValueOf(obj).IsNil()
}

// Returns a pointer to the specified value. It is useful to
// create pointers from the literals.
func Ptr[T any](value T) *T {
	return &value
}

// Checks if the specified value is a whole number.
func IsWholeNumber(value interface{}) bool {
	if value == nil {
		return false
	}
	valueType := reflect.TypeOf(value)
	switch valueType.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return true
	default:
		return false
	}
}

// Combine multiple errors into a single one.
func CombineErrors(topErrorMsg string, errs []error) error {
	var noNils []error
	for _, err := range errs {
		if err != nil {
			noNils = append(noNils, err)
		}
	}

	if len(noNils) == 0 {
		return nil
	}

	combinedErr := errors.New(topErrorMsg)

	for _, err := range noNils {
		combinedErr = errors.WithMessage(combinedErr, err.Error())
	}

	return combinedErr
}

// Checks if the path is a socket.
func IsSocket(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.Mode().Type() == fs.ModeSocket
}

// Convenience function returning nil when an input string parameter
// is nil or empty.
func NullifyEmptyString(s *string) *string {
	if s != nil && len(*s) > 0 {
		return s
	}
	return nil
}
