package dbmodel

import (
	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
)

type CalculatedNetworkMetrics struct {
	Label           string
	AddrUtilization int16
	PdUtilization   int16
}

type CalculatedMetrics struct {
	AuthorizedMachines   int64
	UnauthorizedMachines int64
	UnreachableMachines  int64
	SubnetMetrics        []CalculatedNetworkMetrics
	SharedNetworkMetrics []CalculatedNetworkMetrics
}

func GetCalculatedMetrics(db *pg.DB) (*CalculatedMetrics, error) {
	metrics := CalculatedMetrics{}
	err := db.Model().
		Table("machine").
		ColumnExpr("COUNT(*) FILTER (WHERE machine.authorized) AS \"authorized_machines\"").
		ColumnExpr("COUNT(*) FILTER (WHERE NOT(machine.authorized)) AS \"unauthorized_machines\"").
		ColumnExpr("COUNT(*) FILTER (WHERE machine.error IS NOT NULL) AS \"unreachable_machines\"").
		Select(&metrics)
	if err != nil {
		return nil, errors.Wrap(err, "cannot calculate global metrics")
	}

	err = db.Model().
		Table("subnet").
		ColumnExpr("\"prefix\" AS \"label\"").
		Column("addr_utilization", "pd_utilization").
		Select(&metrics.SubnetMetrics)

	if err != nil {
		return nil, errors.Wrap(err, "cannot calculate subnet metrics")
	}

	err = db.Model().
		Table("shared_network").
		ColumnExpr("\"name\" AS \"label\"").
		Column("addr_utilization", "pd_utilization").
		Select(&metrics.SharedNetworkMetrics)

	if err != nil {
		return nil, errors.Wrap(err, "cannot calculate shared network metrics")
	}

	return &metrics, nil
}
