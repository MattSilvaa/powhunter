package metrics

import (
	"testing"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/stretchr/testify/assert"
)

func TestCapSourcesFoldsTheLongTailIntoOther(t *testing.T) {
	rows := make([]dbgen.GetSignupsBySourceRow, 0, maxSourceLabels+2)
	for i := range maxSourceLabels + 2 {
		rows = append(rows, dbgen.GetSignupsBySourceRow{Source: string(rune('a' + i)), Signups: 1})
	}

	counts := capSources(rows)
	assert.Len(t, counts, maxSourceLabels+1)
	assert.Equal(t, int64(2), counts["other"])
}
