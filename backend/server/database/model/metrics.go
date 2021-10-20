package dbmodel

import "github.com/go-pg/pg/v9"

type CalculatedMetrics struct {
	AuthorizedMachines   int64
	UnauthorizedMachines int64
	UnreachableMachines  int64
}

func GetCalculatedMetrics(db *pg.DB) (*CalculatedMetrics, error) {
	metrics := CalculatedMetrics{}
	q := db.Model()
	q = q.Table("machine")
	q = q.ColumnExpr("COUNT(*) FILTER (WHERE machine.authorized) AS \"authorized_machines\"")
	q = q.ColumnExpr("COUNT(*) FILTER (WHERE NOT(machine.authorized)) AS \"unauthorized_machines\"")
	q = q.ColumnExpr("COUNT(*) FILTER (WHERE machine.error IS NOT NULL) AS \"unreachable_machines\"")
	err := q.Select(&metrics)
	return &metrics, err
}
