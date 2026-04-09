package storkutil

import "iter"

// Returns an iterator that zips together two slices. The resulting iterator
// yields pairs of elements from the two slices at the same index.
func ZipPairs[A, B any](a []A, b []B) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		for i := 0; i < len(a) && i < len(b); i++ {
			if !yield(a[i], b[i]) {
				return
			}
		}
	}
}
