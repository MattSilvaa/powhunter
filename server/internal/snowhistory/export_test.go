package snowhistory

import "time"

// CacheTTL exposes cacheTTL to the external tests.
const CacheTTL = cacheTTL

// SetClock replaces the service's clock in tests.
func (s *Service) SetClock(now func() time.Time) {
	s.now = now
}
