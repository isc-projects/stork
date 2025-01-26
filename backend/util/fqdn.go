package storkutil

import (
	"bytes"
	"slices"
	"strings"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Compares two DNS names. This yields the following zones ordering:
//
// - example.com (com is before .org and .version)
// - host.example.com (host is a child of example.com, so it is behind it)
// - example.org (.org is after com but before .version)
// - a.example.org (a is a child of example.org)
// - bind.version (.version is behind .com and .org)
//
// It returns -1 when name1 is before name2; it returns 0 when
// name1 and name2 are the same; it returns 1 when name1 is after name2.
func CompareNames(name1, name2 string) int {
	// Turn the domain names into labels.
	s1 := dns.SplitDomainName(name1)
	s2 := dns.SplitDomainName(name2)

	// Compare the labels that exist in both names.
	for i := 0; i < min(len(s1), len(s2)); i++ {
		switch {
		case s2[len(s2)-i-1] == s1[len(s1)-i-1]:
			continue
		case s1[len(s1)-i-1] < s2[len(s2)-i-1]:
			return -1
		default:
			return 1
		}
	}
	// It seems that all compared labels so far are equal.
	switch {
	// If the name lengths are the same it means that the names are
	// equal (remember that we have already compared the labels for
	// equality).
	case len(s1) == len(s2):
		return 0
	// The second name is longer so it goes behind the first name.
	// It is a child zone.
	case len(s1) < len(s2):
		return -1
	default:
		// The first name is longer so it goes behind the second
		// name. It is a child zone.
		return 1
	}
}

// Converts DNS name to the name with labels ordered backwards. For example:
// zone.example.org is converted to org.example.zone.
func ConvertNameToRname(name string) string {
	labels := dns.SplitDomainName(name)
	slices.Reverse(labels)
	return strings.Join(labels, ".")
}

// Represents Fully Qualified Domain Name (FQDN).
// See https://datatracker.ietf.org/doc/html/rfc1035.
type Fqdn struct {
	// A collection of labels forming the FQDN.
	labels []string
	// Indicates if the FQDN is partial or full.
	partial bool
}

// Returns true if the parsed FQDN is partial. Otherwise
// it returns false.
func (fqdn Fqdn) IsPartial() bool {
	return fqdn.partial
}

// Converts FQDN to bytes form as specified in RFC 1035. It is output
// as a collection of labels, each preceded with a label length.
func (fqdn Fqdn) ToBytes() (buf []byte, err error) {
	var buffer bytes.Buffer
	for _, label := range fqdn.labels {
		if err = buffer.WriteByte(byte(len(label))); err != nil {
			err = errors.WithStack(err)
			return
		}
		if _, err = buffer.WriteString(label); err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	if !fqdn.partial {
		if err = buffer.WriteByte(0); err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	buf = buffer.Bytes()
	return
}

// Parses an FQDN string. If the string does not contain a valid FQDN,
// it returns nil and an error.
func ParseFqdn(fqdn string) (*Fqdn, error) {
	// Remove leading and trailing whitespace.
	fqdn = strings.TrimSpace(fqdn)
	if len(fqdn) == 0 {
		return nil, errors.New("failed to parse an empty FQDN")
	}
	// Full FQDN has a terminating dot.
	full := fqdn[len(fqdn)-1] == '.'
	labels := strings.Split(fqdn, ".")
	if full {
		// If this is a full FQDN, remove last label (after trailing dot).
		labels = labels[:len(labels)-1]
		// Expect that full FQDN has at least 3 labels.
		if len(labels) < 3 {
			return nil, errors.Errorf("full FQDN %s must contain at least three labels", fqdn)
		}
	}
	// Validate the labels.
	for i, label := range labels {
		// Last label in the full FQDN must only contain letters and must be
		// at least two characters long.
		if full && i == len(labels)-1 {
			if len(label) < 2 {
				return nil, errors.Errorf("last label of the full FQDN %s must be at least two characters long", fqdn)
			}
			for _, c := range label {
				if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
					return nil, errors.Errorf("last label of the full FQDN %s must only contain letters and must be at least two characters long", fqdn)
				}
			}
		} else {
			// Other labels must not be empty, may contain digits, letters and hyphens
			// but the hyphens must not be at the start nor at the end of the label.
			if len(label) == 0 {
				return nil, errors.Errorf("empty label found in the FQDN %s", fqdn)
			}
			for i, c := range label {
				if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
					(c < '0' || c > '9') &&
					(i == 0 || i == len(label)-1 || c != '-') {
					return nil, errors.Errorf("first and middle labels in the FQDN %s may only contain digits, letters and hyphens but hyphens must not be at the start and the end of the FQDN", fqdn)
				}
			}
		}
	}
	// Everything good. Create the FQDN instance.
	parsed := &Fqdn{
		labels:  labels,
		partial: !full,
	}
	return parsed, nil
}
