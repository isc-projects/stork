package dbmodel

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"isc.org/stork/appcfg/dnsconfig"
)

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
func GetDNSConfigRRs(dbi pg.DBI, localZoneID int64) ([]*dnsconfig.RR, error) {
	var rrs []*dnsconfig.RR
	err := dbi.Model((*LocalZoneRR)(nil)).
		Column("name", "ttl", "class", "type", "rdata").
		Where("local_zone_id = ?", localZoneID).
		Select(&rrs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to select resource records for local zone %d", localZoneID)
	}
	return rrs, nil
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
