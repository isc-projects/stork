package restservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that the time is converted to the datetime pointer properly.
func TestConvertToOptionalDatetime(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		require.Nil(t, convertToOptionalDatetime(time.Time{}))
	})

	t.Run("non-zero", func(t *testing.T) {
		require.NotNil(t, convertToOptionalDatetime(time.Now()))
	})
}
