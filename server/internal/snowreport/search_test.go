package snowreport_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/snowreport"
)

func liveReport(names ...string) snowreport.Report {
	resorts := make([]snowreport.LiveResort, len(names))
	for i, name := range names {
		resorts[i] = snowreport.LiveResort{Rank: i + 1, Name: name}
	}

	return snowreport.Report{
		Mode: snowreport.ModeLive,
		Live: &snowreport.Live{Days: 10, ResortCount: len(resorts), Resorts: resorts},
	}
}

func TestTopKeepsTheFullCount(t *testing.T) {
	report := liveReport("A", "B", "C", "D", "E", "F", "G")

	top := report.Top(snowreport.TopResorts)

	assert.Len(t, top.Live.Resorts, 5)
	assert.Equal(t, 7, top.Live.ResortCount)
	assert.Len(t, report.Live.Resorts, 7, "the full report is not modified")

	assert.Len(t, liveReport("A", "B").Top(5).Live.Resorts, 2, "fewer resorts than n is fine")
}

func TestSearchIgnoresCaseAndPunctuation(t *testing.T) {
	report := liveReport("Palisades Tahoe", "Mt. Bachelor", "Alta", "Mount Hood Meadows")

	result := report.Search("  MT BACH ")

	require.Len(t, result.Live, 1)
	assert.Equal(t, "Mt. Bachelor", result.Live[0].Name)
	assert.Equal(t, 2, result.Live[0].Rank, "a match keeps its place on the full board")
	assert.Empty(t, result.Recap)
}

func TestSearchReturnsBoardOrderUpToTheCap(t *testing.T) {
	names := make([]string, 0, 30)
	for i := range 30 {
		names = append(names, fmt.Sprintf("Peak %02d", i+1))
	}

	result := liveReport(names...).Search("peak")

	require.Len(t, result.Live, snowreport.SearchMaxResults)
	assert.Equal(t, "Peak 01", result.Live[0].Name)

	assert.Empty(t, liveReport("Alta").Search("zzz").Live)
	assert.Empty(t, liveReport("Alta").Search("...").Live, "a query with no letters matches nothing")
}

func TestSearchUsesTheRecapOffSeason(t *testing.T) {
	report := snowreport.Report{
		Mode: snowreport.ModeRecap,
		Recap: &snowreport.Recap{ResortCount: 2, Resorts: []snowreport.RecapResort{
			{Rank: 1, Name: "Alta"}, {Rank: 2, Name: "Vail"},
		}},
	}

	result := report.Search("vail")

	assert.Equal(t, snowreport.ModeRecap, result.Mode)
	require.Len(t, result.Recap, 1)
	assert.Equal(t, 2, result.Recap[0].Rank)
}
