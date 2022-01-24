package dbmodel

import (
	"math/big"

	"github.com/go-pg/pg/v10/types"
	"github.com/pkg/errors"
)

// Custom support for decimal/numeric in Go-PG.
// It is dedicated to store integer-only numbers. The Postgres decimal/numeric
// type must be defined with scale equals to 0, e.g.: pg:"type:decimal(60,0)".
// See: https://github.com/go-pg/pg/blob/v10/example_custom_test.go
type IntegerDecimal struct {
	big.Int
}

// Interface check for serialization.
var _ types.ValueAppender = (*IntegerDecimal)(nil)

// Custom big.Int serializing to the database record.
func (d IntegerDecimal) AppendValue(b []byte, quote int) ([]byte, error) {
	if quote == 1 {
		b = append(b, '\'')
	}

	b = append(b, []byte(d.String())...)
	if quote == 1 {
		b = append(b, '\'')
	}
	return b, nil
}

// Interface check for deserialization.
var _ types.ValueScanner = (*IntegerDecimal)(nil)

// Custom decimal/numeric parsing to big.Int.
func (d *IntegerDecimal) ScanValue(rd types.Reader, n int) error {
	if n <= 0 {
		d.Int = *big.NewInt(0)
		return nil
	}

	tmp, err := rd.ReadFullTemp()
	if err != nil {
		return err
	}

	_, ok := d.Int.SetString(string(tmp), 10)
	if !ok {
		return errors.Errorf("invalid decimal")
	}

	return nil
}
