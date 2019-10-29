package dbops

import (
	"github.com/go-pg/pg/v9"
)

func TestPgConnection(db *pg.DB) bool {
	var n int
	_, err := db.QueryOne(pg.Scan(&n), "SELECT 1")
	return err == nil
}
