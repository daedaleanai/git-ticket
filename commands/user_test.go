package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func runCompareUsername(t *testing.T, username, search string, expected bool) {
	result, err := compareUsername(username, search)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestCompareUsername(t *testing.T) {
	runCompareUsername(t, "Mickey Mouse", "Mickey Mouse", true)
	runCompareUsername(t, "Mickey Mouse", "Mickey", true)
	runCompareUsername(t, "Mickey Mouse", "Mick", true)
	runCompareUsername(t, "Mickey Mouse", "mick", true)
	runCompareUsername(t, "Mickey Mouse", "MICK", true)

	runCompareUsername(t, "Mìckéy Möüse", "Mìckéy Möüse", true)
	runCompareUsername(t, "Mìckéy Möüse", "Mìckéy", true)
	runCompareUsername(t, "Mìckéy Möüse", "Mickey", true)
	runCompareUsername(t, "Mìckéy Möüse", "Mouse", true)

	runCompareUsername(t, "Mickey Mouse", "Myckey Mouse", false)
	runCompareUsername(t, "Mickey Mouse", "House", false)
}
