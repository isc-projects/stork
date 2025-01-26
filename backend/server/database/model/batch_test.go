package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
)

type batchError struct{}

func (err batchError) Error() string {
	return "batch error"
}

// Test that items can be added to a batch and that the callback function
// is triggered when the batch hits a limit. Also test that calling Finish
// inserts remaining items in the batch.
func TestBatch(t *testing.T) {
	var (
		db            pg.DB
		callCount     int
		capturedItems []int
	)
	batch := NewBatch(db, 10, func(d pg.DBI, items ...int) error {
		callCount++
		capturedItems = items
		return nil
	})
	require.NotNil(t, batch)

	for i := 0; i < 9; i++ {
		_ = batch.Add(i)
		require.Zero(t, callCount)
	}

	_ = batch.Add(9)
	require.Equal(t, 1, callCount)
	require.Len(t, capturedItems, 10)
	require.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, capturedItems)

	for i := 10; i > 1; i-- {
		_ = batch.Add(i)
		require.Equal(t, 1, callCount)
	}

	_ = batch.Add(1)
	require.Equal(t, 2, callCount)
	require.Len(t, capturedItems, 10)
	require.Equal(t, []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}, capturedItems)

	_ = batch.Add(1)
	_ = batch.Add(5)
	require.Equal(t, 2, callCount)

	_ = batch.Finish()
	require.Equal(t, 3, callCount)
	require.Len(t, capturedItems, 2)
	require.Equal(t, []int{1, 5}, capturedItems)

	_ = batch.Finish()
	require.Equal(t, 3, callCount)
}

// Test that errors are propagated through a batch.
func TestBatchError(t *testing.T) {
	var (
		db            pg.DB
		expectedError *batchError
	)
	batch := NewBatch(db, 1, func(d pg.DBI, items ...int) error {
		return &batchError{}
	})
	require.NotNil(t, batch)
	err := batch.Add(1)
	require.ErrorAs(t, err, &expectedError)

	err = batch.Finish()
	require.ErrorAs(t, err, &expectedError)
}
