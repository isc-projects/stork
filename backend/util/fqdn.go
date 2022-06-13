package storkutil

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

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
// it returns nil.
func ParseFqdn(fqdn string) *Fqdn {
	// Remove leading and trailing whitespace.
	fqdn = strings.TrimSpace(fqdn)
	if len(fqdn) == 0 {
		return nil
	}
	// Full FQDN has a terminating dot.
	full := fqdn[len(fqdn)-1] == '.'
	labels := strings.Split(fqdn, ".")
	if full {
		// If this is a full FQDN, remove last label (after trailing dot).
		labels = labels[:len(labels)-1]
		// Expect that full FQDN has at least 3 labels.
		if len(labels) < 3 {
			return nil
		}
	}
	// Validate the labels.
	var lastLabelRegexp *regexp.Regexp
	if full {
		lastLabelRegexp = regexp.MustCompile("^[A-Za-z]{2,63}$")
	}
	middleLabelRegexp := regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]$")
	for i, label := range labels {
		// Last label in the full FQDN must only contain letters and must be
		// at least two characters long.
		if full && i == len(labels)-1 {
			if matched := lastLabelRegexp.MatchString(label); !matched {
				return nil
			}
		}
		// Other labels may contain digits, letters and hyphens but the hyphens
		// must not be at the start or an end of the label.
		if matched := middleLabelRegexp.MatchString(label); !matched {
			return nil
		}
	}
	// Everything good. Create the FQDN instance.
	parsed := &Fqdn{
		labels:  labels,
		partial: !full,
	}
	return parsed
}
