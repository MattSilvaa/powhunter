package middleware

// SetMaxVisitors lets tests shrink the visitor table instead of tracking
// 100,000 clients.
func (l *RateLimiter) SetMaxVisitors(n int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.maxVisitors = n
}

// VisitorCount reports how many clients are tracked.
func (l *RateLimiter) VisitorCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.visitors)
}
