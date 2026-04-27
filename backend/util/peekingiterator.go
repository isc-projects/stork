package storkutil

// The peeking iterator implements walking over the arbitrary list of items
// with peeking a forward and/or backward element.
type PeekingIterator[T any] struct {
	items []T
	index int
}

// Creates a new peeking iterator with the given items.
func NewPeekingIterator[T any](items []T) *PeekingIterator[T] {
	return &PeekingIterator[T]{
		items: items,
		index: 0,
	}
}

// Consumes and returns the next item from the iterator. It returns false if
// there is no next item.
func (i *PeekingIterator[T]) Next() (item T, ok bool) {
	item, ok = i.Peek()
	if ok {
		i.index++
	}
	return
}

// Peeks the next item from the iterator without consuming it. It returns false
// if there is no next item.
func (i *PeekingIterator[T]) Peek() (item T, ok bool) {
	if i.index < len(i.items) {
		item = i.items[i.index]
		ok = true
	}
	return
}

// Peeks all subsequent items from the iterator without consuming them.
func (i *PeekingIterator[T]) PeekSubsequent() (items []T) {
	index := i.index
	for index < len(i.items) {
		items = append(items, i.items[index])
		index++
	}
	return items
}

// Peeks the previous item from the iterator. It returns false if there is no
// previous item.
func (i *PeekingIterator[T]) PeekBack() (item T, ok bool) {
	if i.index-2 >= 0 {
		item = i.items[i.index-2]
		ok = true
	}
	return
}
