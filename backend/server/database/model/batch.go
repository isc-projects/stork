package dbmodel

import "github.com/go-pg/pg/v10"

// Batch is a mechanism for collecting and inserting multiple database
// entries together. It is mostly useful for massive updates involving
// thousands or even millions of records. Grouping them into batches
// significantly improves performance of insertion. A batch has a limit
// for the number of collected entries. When this limit is exceeded the
// batch inserts all collected entries into the database and removes
// them from the queue making space for another set of entries.
type Batch[T any] struct {
	db    pg.DBI
	items []T
	limit int
	fn    func(pg.DBI, ...T) error
}

// Instantiates new batch. The limit specifies the number of entries held
// in the batch before they are all inserted into the database. The function
// implements insertion of the entries specified as variadic parameters.
func NewBatch[T any](db pg.DBI, limit int, fn func(pg.DBI, ...T) error) (batch *Batch[T]) {
	return &Batch[T]{
		db:    db,
		items: make([]T, 0, limit),
		limit: limit,
		fn:    fn,
	}
}

// Inserts new item into the batch. If the number of items in the batch hits
// the specified limit the items are inserted into the database.
func (buffer *Batch[T]) Add(item T) error {
	buffer.items = append(buffer.items, item)
	if (len(buffer.items) >= buffer.limit) && len(buffer.items) > 0 {
		if err := buffer.fn(buffer.db, buffer.items...); err != nil {
			return err
		}
		buffer.items = buffer.items[:0]
	}
	return nil
}

// Flushes the batch and adds a new item to it.
func (buffer *Batch[T]) FlushAndAdd(item T) error {
	if err := buffer.Flush(); err != nil {
		return err
	}
	return buffer.Add(item)
}

// Completes the batch insert. This function must be called when no new
// items are expected and the batch holds some not insert items. Calling
// this function immediately attempts to insert all items from the batch
// into the database.
func (buffer *Batch[T]) Flush() error {
	if len(buffer.items) > 0 {
		if err := buffer.fn(buffer.db, buffer.items...); err != nil {
			return err
		}
		buffer.items = buffer.items[:0]
	}
	return nil
}
