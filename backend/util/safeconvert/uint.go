package safeconvert

import (
	"math"
)

func FromUintToUint32(value uint) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, ErrOverflowUint32
	}
	return uint32(value), nil
}

func FromUint64ToInt64(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, ErrOverflowInt64
	}
	return int64(value), nil
}
