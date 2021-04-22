package storkutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// IP protocol type.
type IPType int

// IP protocol type enum.
const (
	IPv4 IPType = iota
	IPv6
)

// Structure returned by ParseIP function. It comprises the information about
// the parsed IP address or delegated prefix.
type ParsedIP struct {
	NetworkAddress string // Full address or delegated prefix e.g. 192.0.2.0/24.
	Protocol       IPType // Detected IP type: IPv4 or IPv6.
	NetworkPrefix  string // Prefix part (slash and length excluded).
	PrefixLength   int    // Network address mask.
	Prefix         bool   // Boolean indicating if it is an address or prefix.
	CIDR           bool   // Boolean indicating if parsed value was in CIDR form.
}

func UTCNow() time.Time {
	return time.Now().UTC()
}

// Returns URL of the host with port.
func HostWithPortURL(address string, port int64) string {
	return fmt.Sprintf("http://%s:%d/", address, port)
}

// Parses URL into host and port.
func ParseURL(url string) (host string, port int64) {
	ptrn := regexp.MustCompile(`https{0,1}:\/\/\[{1}(\S+)\]{1}(:([0-9]+)){0,1}`)
	m := ptrn.FindStringSubmatch(url)

	if len(m) == 0 {
		ptrn := regexp.MustCompile(`https{0,1}:\/\/([^\s\:\/]+)(:([0-9]+)){0,1}`)
		m = ptrn.FindStringSubmatch(url)
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
	return host, port
}

// Turns IP address into CIDR. If the IP address already seems to be using
// CIDR notation, it is returned.
func MakeCIDR(address string) (string, error) {
	if !strings.Contains(address, "/") {
		ip := net.ParseIP(address)
		if ip == nil {
			return address, errors.Errorf("provided string %s is not a valid IP address", address)
		}
		ip4 := ip.To4()
		if ip4 != nil {
			address += "/32"
		} else {
			address += "/128"
		}
	}
	return address, nil
}

// Parses an IP address or prefix and returns parsed information in the
// structure. If the specified value is invalid, a nil structure is
// returned.
func ParseIP(address string) *ParsedIP {
	var ipNet *net.IPNet

	// Check if this is an IP address without a prefix length.
	ipAddr := net.ParseIP(address)
	if ipAddr == nil {
		// Apparently it comprises a prefix length.
		ip, net, err := net.ParseCIDR(address)
		if err != nil {
			// It is neither an IP address nor prefix.
			return nil
		}
		// Gather prefix information from the network.
		ipAddr = ip
		ipNet = net
	}

	parsedIP := &ParsedIP{}

	if ipNet != nil {
		// Check prefix length.
		ones, bits := ipNet.Mask.Size()
		if ones != bits {
			// This seems to be a prefix.
			parsedIP.NetworkAddress = ipNet.String()
			parsedIP.NetworkPrefix = ipNet.IP.String()
			parsedIP.PrefixLength = ones
			parsedIP.Prefix = true
		}
		// Caller specified it with a prefix length.
		parsedIP.CIDR = true
	}

	// Prefix length wasn't specified, so we need to parse the
	// IP address.
	if !parsedIP.Prefix {
		parsedIP.NetworkAddress = ipAddr.String()
		parsedIP.NetworkPrefix = ipAddr.String()
	}

	// Check if this is IPv4 or IPv6 address/prefix.
	if ipAddr.To4() != nil {
		if parsedIP.PrefixLength == 0 {
			parsedIP.PrefixLength = 32
		}
		parsedIP.Protocol = IPv4
	} else {
		if parsedIP.PrefixLength == 0 {
			parsedIP.PrefixLength = 128
		}
		parsedIP.Protocol = IPv6
	}
	// Parsing finished.
	return parsedIP
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
	// Remove any colons and whitespaces.
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

func SetupLogging() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
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

// Helper code for mocking os/exec stuff... pathetic.
type Commander interface {
	Output(string, ...string) ([]byte, error)
}

type RealCommander struct{}

func (c RealCommander) Output(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).Output()
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
	hexString = strings.ReplaceAll(hexString, ":", "")
	decoded, _ := hex.DecodeString(hexString)
	return decoded
}
