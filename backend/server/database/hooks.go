package dbops

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
)

// Defines the go-pg hooks to enable the SQL query logging.
// It implements the "pg.QueryHook" interface.
type DBLogger struct{}

// The type used to define context keys for database handling.
type contextKeywordDB string

const suppressQueryLoggingKeyword contextKeywordDB = "suppress-query-logging"

// Hook run before SQL query execution.
func (d DBLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	if HasSuppressedQueryLogging(c) {
		return c, nil
	}

	// When making queries on the system_user table we want to make sure that
	// we don't expose actual data in the logs, especially password.
	if model, ok := q.Model.(orm.TableModel); ok {
		if model != nil {
			table := model.Table()
			if table != nil && table.SQLName == "system_user" {
				// Query on the system_user table. Don't print the actual data.
				fmt.Println(q.UnformattedQuery())
				return c, nil
			}
		}
	}
	query, err := q.FormattedQuery()
	// FormattedQuery returns a tuple of query and error. The error in most cases is nil, and
	// we don't want to print it. On the other hand, all logging is printed on stdout. We want
	// to print here to stderr, so it's possible to redirect just the queries to a file.
	if err != nil {
		// Let's print errors as SQL comments. This will allow trying to run the export as a script.
		fmt.Fprintf(os.Stderr, "%s -- error:%s\n", string(query), err)
	} else {
		fmt.Fprintln(os.Stderr, string(query))
	}
	return c, nil
}

// Hook run after SQL query execution.
func (d DBLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	return nil
}

// Checks the size of the SQL query and rejects it if it exceeds the uint32
// size (2^32-5 bytes) limit. It prevents against integer overflow in the go-pg
// internals.
//
// The go-pg library expects that the query size fits into a uint32
// and it casts the query buffer length to this type. However, it sends a full
// query to the database, which can be larger than 2^32-5 bytes. This behavior
// can lead to an SQL injection attack if the query is constructed from user
// input.
//
// The integer overflow occurs in FinishMessage method in:
// tools/golang/gopath/pkg/mod/github.com/go-pg/pg/v10@v10.14.0/internal/pool/write_buffer.go
//
// This attack is labeled as CVE-2024-44905 / GO-2025-3764.
// There is no patch from the go-pg team yet, and it is likely that it will not
// be fixed.
//
// This hook is a workaround to prevent the attack by rejecting queries that
// exceed 2^32-5 bytes limit. So, it is impossible to overflow the query
// buffer length.
//
// ToDo: Remove this hook when the go-pg team fixes the issue or when we
// migrate to bun or another library that does not have this issue.
type DBQuerySizeLimiter struct {
	// Maximum allowed query size in bytes.
	limit int
}

// Instantiate a new DBQuerySizeLimiter with the default limit.
func NewDBQuerySizeLimiterDefault() pg.QueryHook {
	return DBQuerySizeLimiter{
		// The full query buffer length must fit into uint32.
		// The query size counts itself that is 4 bytes. Additionally, all
		// queries are prepended by a type byte.
		// The type byte and query size are excluded from the formatted query
		// passed to the hook.
		// So, maximum allowed payload size that is validated by the hook is
		// 2^32-5 bytes.
		limit: math.MaxUint32 - 5,
	}
}

// Instantiate a new DBQuerySizeLimiter with a custom limit.
// Defined for testing purposes.
func NewDBQuerySizeLimiterCustom(limit int) pg.QueryHook {
	return DBQuerySizeLimiter{
		limit: limit,
	}
}

// It verifies if the query exceeds the uint32 size limit.
// The QueryEvent doesn't contain the query size calculated by the go-pg
// library. However, it contains the query string, which is a payload of the
// query. This payload is not limited by go-pg to its calculated size. It is
// send as-is. We can recognize if the integer overflow happened by checking
// this query string size.
func (d DBQuerySizeLimiter) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	// Query bytes that will be sent to the database.
	query, err := q.FormattedQuery()
	if err != nil {
		// There is no query to check, so we can skip the check.
		// However, it never happens because the go-pg library always returns
		// a nil error for the FormattedQuery method.
		return c, nil
	}

	// The query string is constructed from the full query buffer that is not
	// truncated to the query length.
	if len(query) > d.limit {
		return c, errors.Errorf(
			"query size exceeds %dB limit, got: %dB", d.limit, len(query),
		)
	}
	return c, nil
}

// Hook run after SQL query execution. Does nothing.
func (d DBQuerySizeLimiter) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	return nil
}
