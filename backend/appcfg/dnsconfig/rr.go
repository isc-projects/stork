package dnsconfig

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Represents a DNS resource record in Stork. It can be embedded in database
// model structures to represent the RRs in the database.
type RR struct {
	Name  string
	TTL   int64
	Type  string
	Class string
	Rdata string
}

// Instantiates a new RR from a string.
func NewRR(rrText string) (*RR, error) {
	rr, err := dns.NewRR(rrText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse RR: %s", rrText)
	}
	fields := strings.Fields(rr.String())
	// The full RR record has the following format:
	// <name> <ttl> <class> <type> <data>
	// We are interested in extracting the <data> field.
	var data string
	if len(fields) > 4 {
		data = strings.Join(fields[4:], " ")
	}
	return &RR{
		Name:  rr.Header().Name,
		TTL:   int64(rr.Header().Ttl),
		Type:  dns.TypeToString[rr.Header().Rrtype],
		Class: dns.ClassToString[rr.Header().Class],
		Rdata: data,
	}, nil
}

// Returns a string representation of the RR.
func (rr *RR) GetString() string {
	return fmt.Sprintf("%s %d %s %s %s", rr.Name, rr.TTL, rr.Class, rr.Type, rr.Rdata)
}
