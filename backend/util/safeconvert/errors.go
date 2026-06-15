package safeconvert

import "github.com/pkg/errors"

var (
	ErrOverflowInt64  error = errors.New("value exceeds int64 limits")
	ErrOverflowUint32 error = errors.New("value exceeds uint32 limits")
)
