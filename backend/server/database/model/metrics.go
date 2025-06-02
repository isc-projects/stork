package dbmodel

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Metric values calculated for specific subnet or shared network.
type CalculatedNetworkMetrics struct {
	// Subnet prefix. Empty for shared networks.
	Prefix string
	// Shared network name. Empty if subnet isn't part of a shared network.
	SharedNetwork string
	// IP family.
	Family int
	// Address utilization.
	AddrUtilization Utilization
	// Delegated prefix utilization.
	PdUtilization Utilization
	// Statistics. It is nil for subnets.
	SharedNetworkStats SubnetStats
}

// Metric values calculated from the database.
type CalculatedMetrics struct {
	AuthorizedMachines   int64
	UnauthorizedMachines int64
	UnreachableMachines  int64
	SubnetMetrics        []CalculatedNetworkMetrics
	SharedNetworkMetrics []CalculatedNetworkMetrics
}

// Calculates various metrics using several SELECT queries.
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
		Join("LEFT JOIN shared_network").JoinOn("shared_network.id = subnet.shared_network_id").
		Column("prefix").
		ColumnExpr("shared_network.name AS \"shared_network\"").
		ColumnExpr("family(\"prefix\") AS \"family\"").
		Column("subnet.addr_utilization", "subnet.pd_utilization").
		Select(&metrics.SubnetMetrics)
	if err != nil {
		return nil, errors.Wrap(err, "cannot calculate subnet metrics")
	}

	err = db.Model().
		Table("shared_network").
		ColumnExpr("\"name\" AS \"shared_network\"").
		ColumnExpr("\"inet_family\" AS \"family\"").
		Column("addr_utilization", "pd_utilization").
		ColumnExpr("\"stats\" AS \"shared_network_stats\"").
		Select(&metrics.SharedNetworkMetrics)
	if err != nil {
		return nil, errors.Wrap(err, "cannot calculate shared network metrics")
	}

	return &metrics, nil
}
