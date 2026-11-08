package snowreport

import (
	"strings"
	"unicode"
)

const (
	// TopResorts is how many resorts the home page board shows. The rest are
	// reached through search, so the page stays the same size however many
	// resorts there are.
	TopResorts = 5
	// SearchMaxResults caps one search's response.
	SearchMaxResults = 10
	// SearchMinQueryLen and SearchMaxQueryLen bound the search text.
	SearchMinQueryLen = 2
	SearchMaxQueryLen = 64
)

// Top returns a copy of the report holding only the first n resorts of the
// active block. ResortCount still reports the full count.
func (r Report) Top(n int) Report {
	top := r

	if r.Live != nil {
		live := *r.Live
		live.Resorts = live.Resorts[:min(n, len(live.Resorts))]
		top.Live = &live
	}

	if r.Recap != nil {
		recap := *r.Recap
		recap.Resorts = recap.Resorts[:min(n, len(recap.Resorts))]
		top.Recap = &recap
	}

	return top
}

// SearchResult is the /api/snow-report/search response. Only the list for
// Mode is set, so a match is shown with the same numbers as the board.
type SearchResult struct {
	Mode  Mode          `json:"mode"`
	Live  []LiveResort  `json:"live"`
	Recap []RecapResort `json:"recap"`
}

// SearchIndex holds a report with every resort name normalized up front, so
// a search costs one pass of string comparisons and allocates only for the
// query. Build one per report, not per search.
type SearchIndex struct {
	report     Report
	liveNames  []string
	recapNames []string
}

// NewSearchIndex normalizes the names in the report's lists once.
func NewSearchIndex(r Report) *SearchIndex {
	index := &SearchIndex{report: r}

	if r.Live != nil {
		index.liveNames = make([]string, len(r.Live.Resorts))
		for i, resort := range r.Live.Resorts {
			index.liveNames[i] = normalize(resort.Name)
		}
	}

	if r.Recap != nil {
		index.recapNames = make([]string, len(r.Recap.Resorts))
		for i, resort := range r.Recap.Resorts {
			index.recapNames[i] = normalize(resort.Name)
		}
	}

	return index
}

// Search finds resorts whose name contains query, ignoring case, spaces and
// punctuation, so "mt bachelor" finds "Mt. Bachelor". Matches come back in
// board order, at most SearchMaxResults of them.
func (i *SearchIndex) Search(query string) SearchResult {
	result := SearchResult{Mode: i.report.Mode, Live: []LiveResort{}, Recap: []RecapResort{}}

	needle := normalize(query)
	if needle == "" {
		return result
	}

	for n, name := range i.liveNames {
		if len(result.Live) == SearchMaxResults {
			break
		}

		if strings.Contains(name, needle) {
			result.Live = append(result.Live, i.report.Live.Resorts[n])
		}
	}

	for n, name := range i.recapNames {
		if len(result.Recap) == SearchMaxResults {
			break
		}

		if strings.Contains(name, needle) {
			result.Recap = append(result.Recap, i.report.Recap.Resorts[n])
		}
	}

	return result
}

// Search is a one-off search of the report. Callers that search the same
// report repeatedly should keep a SearchIndex instead.
func (r Report) Search(query string) SearchResult {
	return NewSearchIndex(r).Search(query)
}

// normalize lowercases text and keeps only letters and digits.
func normalize(text string) string {
	var b strings.Builder

	for _, char := range strings.ToLower(text) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			b.WriteRune(char)
		}
	}

	return b.String()
}
