package notify_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/notify"
)

func TestNewLinkCode(t *testing.T) {
	first, err := notify.NewLinkCode()
	require.NoError(t, err)

	second, err := notify.NewLinkCode()
	require.NoError(t, err)

	assert.True(t, notify.IsLinkCode(first), "a generated code should pass its own shape check")
	assert.NotEqual(t, first, second)
}

func TestIsLinkCode(t *testing.T) {
	assert.True(t, notify.IsLinkCode("Ab12Cd34"))
	assert.False(t, notify.IsLinkCode("Ab12Cd3"), "too short")
	assert.False(t, notify.IsLinkCode("Ab12Cd34x"), "too long")
	assert.False(t, notify.IsLinkCode("Ab12Cd3/"), "not alphanumeric")
}
