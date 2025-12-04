package dbmodel

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"isc.org/stork/daemoncfg/dnsconfig"
)

// A structure specifying zone RRs filtering.
type GetZoneRRsFilter struct {
	// Limit the number of zones returned.
	limit int
	// Paging offset.
	offset int
	// Filter by selected RR type.
	rrTypes map[string]bool
	// Filter by text in the RR name or rdata.
	text string
}

// Instantiates zone RRs filter with all filters disabled.
func NewGetZoneRRsFilter() *GetZoneRRsFilter {
	return &GetZoneRRsFilter{}
}

// Instantiates zone RRs filter with the specified optional parameters.
// If a parameter is nil, it is not set in the filter. This function is
// useful when creating a filter from the REST API parameters.
func NewGetZoneRRsFilterWithParams(offset *int64, limit *int64, rrTypes []string, text *string) *GetZoneRRsFilter {
	filter := NewGetZoneRRsFilter()
	if offset != nil {
		filter.SetOffset(int(*offset))
	}
	if limit != nil {
		filter.SetLimit(int(*limit))
	}
	for _, rrType := range rrTypes {
		filter.EnableType(rrType)
	}
	if text != nil {
		filter.SetText(*text)
	}
	return filter
}

// Enables filtering by selected RR type.
func (filter *GetZoneRRsFilter) EnableType(rrType string) {
	if filter.rrTypes == nil {
		filter.rrTypes = make(map[string]bool)
	}
	filter.rrTypes[strings.ToUpper(rrType)] = true
}

// Returns true if filtering by the specified RR type is enabled.
func (filter *GetZoneRRsFilter) IsTypeEnabled(rrType string) bool {
	return len(filter.rrTypes) == 0 || filter.rrTypes[strings.ToUpper(rrType)]
}

// Returns a slice of enabled RR types.
func (filter *GetZoneRRsFilter) GetTypes() []string {
	return slices.Collect(maps.Keys(filter.rrTypes))
}

// Enables filtering by text in the RR name or rdata.
func (filter *GetZoneRRsFilter) SetText(text string) {
	filter.text = text
}

// Returns the text used for filtering by text in the RR name or rdata.
// Filtering by text is disabled if the text is empty.
func (filter *GetZoneRRsFilter) GetText() string {
	return filter.text
}

// Returns true if the specified RR matches the text filter. It checks
// if the text is contained in the RR name or rdata.
func (filter *GetZoneRRsFilter) IsTextMatches(rr *dnsconfig.RR) bool {
	if filter.text == "" {
		return true
	}
	filterText := strings.ToLower(filter.text)
	for _, field := range []string{rr.Name, rr.Rdata} {
		if strings.Contains(strings.ToLower(field), filterText) {
			return true
		}
	}
	return false
}

// Sets the limit on the number of RRs returned.
func (filter *GetZoneRRsFilter) SetLimit(limit int) {
	if limit < 0 {
		filter.limit = 0
		return
	}
	filter.limit = limit
}

// Returns the limit on the number of RRs returned. If the receiver
// is nil, 0 is returned.
func (filter *GetZoneRRsFilter) GetLimit() int {
	if filter == nil {
		return 0
	}
	return filter.limit
}

// Sets the offset on the number of RRs returned.
func (filter *GetZoneRRsFilter) SetOffset(offset int) {
	if offset < 0 {
		filter.offset = 0
		return
	}
	filter.offset = offset
}

// Returns the offset on the number of RRs returned. If the receiver
// is nil, 0 is returned.
func (filter *GetZoneRRsFilter) GetOffset() int {
	if filter == nil {
		return 0
	}
	return filter.offset
}

// Represents a DNS resource record in the database.
type LocalZoneRR struct {
	dnsconfig.RR
	ID          int64
	LocalZoneID int64

	LocalZone *LocalZone `pg:"rel:has-one"`
}

// Adds a set of RRs to the database within transaction.
func addLocalZoneRRs(tx *pg.Tx, rrs ...*LocalZoneRR) error {
	if _, err := tx.Model(&rrs).Insert(); err != nil {
		return errors.Wrapf(err, "failed to insert %d resource records into the database", len(rrs))
	}
	return nil
}

// Adds a set of RRs to the database.
func AddLocalZoneRRs(dbi pg.DBI, rrs ...*LocalZoneRR) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addLocalZoneRRs(tx, rrs...)
		})
	}
	return addLocalZoneRRs(dbi.(*pg.Tx), rrs...)
}

// Returns a set of RRs converted to an array of []*dnsconfig.RR for
// specified local zone.
func GetDNSConfigRRs(dbi pg.DBI, localZoneID int64, filter *GetZoneRRsFilter) ([]*dnsconfig.RR, int, error) {
	var rrs []*dnsconfig.RR
	q := dbi.Model((*LocalZoneRR)(nil)).
		Column("name", "ttl", "class", "type", "rdata")

	// Filtering is optional. If the filter is nil, return all RRs.
	if filter != nil {
		types := filter.GetTypes()
		if len(types) > 0 {
			q = q.WhereIn("type IN (?)", types)
		}
		if filter.GetText() != "" {
			// Case insensitive match of name or rdata.
			q = q.WhereGroup(func(qq *pg.Query) (*pg.Query, error) {
				qq = qq.WhereOr("name ILIKE ?", "%"+filter.GetText()+"%")
				qq = qq.WhereOr("rdata ILIKE ?", "%"+filter.GetText()+"%")
				return qq, nil
			})
		}
		// Apply paging filters.
		if filter.offset > 0 {
			q = q.Offset(filter.offset)
		}
		if filter.limit > 0 {
			q = q.Limit(filter.limit)
		}
	}
	q = q.Where("local_zone_id = ?", localZoneID).Order("id ASC")
	total, err := q.SelectAndCount(&rrs)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to select resource records for local zone %d", localZoneID)
	}
	return rrs, total, nil
}

// Deletes a set of RRs from the database within transaction for
// a specified local zone.
func deleteLocalZoneRRs(tx *pg.Tx, localZoneID int64) error {
	if _, err := tx.Model(&LocalZoneRR{}).Where("local_zone_id = ?", localZoneID).Delete(); err != nil {
		return errors.Wrapf(err, "failed to delete resource records for local zone %d", localZoneID)
	}
	return nil
}

// Deletes a set of RRs from the database for a specified local zone.
func DeleteLocalZoneRRs(dbi pg.DBI, localZoneID int64) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return deleteLocalZoneRRs(tx, localZoneID)
		})
	}
	return deleteLocalZoneRRs(dbi.(*pg.Tx), localZoneID)
}
