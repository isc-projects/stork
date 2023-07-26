package restservice

import (
	"time"

	"github.com/go-openapi/strfmt"
)

// Converts the Golang date to the pointer to OpenAPI datetime. It handles
// the zero value checking.
func convertToOptionalDatetime(date time.Time) *strfmt.DateTime {
	if date.IsZero() {
		return nil
	}
	datetime := strfmt.DateTime(date)
	return &datetime
}
