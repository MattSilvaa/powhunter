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

// Search finds resorts whose name contains query, ignoring case, spaces and
// punctuation, so "mt bachelor" finds "Mt. Bachelor". Matches come back in
// board order, at most SearchMaxResults of them. It scans names in memory,
// which takes microseconds even for thousands of resorts.
func (r Report) Search(query string) SearchResult {
	result := SearchResult{Mode: r.Mode, Live: []LiveResort{}, Recap: []RecapResort{}}

	needle := normalize(query)
	if needle == "" {
		return result
	}

	if r.Live != nil {
		for _, resort := range r.Live.Resorts {
			if len(result.Live) == SearchMaxResults {
				break
			}

			if strings.Contains(normalize(resort.Name), needle) {
				result.Live = append(result.Live, resort)
			}
		}
	}

	if r.Recap != nil {
		for _, resort := range r.Recap.Resorts {
			if len(result.Recap) == SearchMaxResults {
				break
			}

			if strings.Contains(normalize(resort.Name), needle) {
				result.Recap = append(result.Recap, resort)
			}
		}
	}

	return result
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
