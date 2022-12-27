package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests that the correct indentation coefficient is returned for
// different indentation types.
func TestIndentationCoefficient(t *testing.T) {
	require.Equal(t, 1, getIndentationCoefficient(tabs))
	require.Equal(t, 4, getIndentationCoefficient(fourSpaces))
}
